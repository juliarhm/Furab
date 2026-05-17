//go:build functional
// +build functional

// Package functional contains functional tests for the driver service.
// Functional tests access a REAL PostgreSQL database.
//
// Run with: go test ./test/functional/... -v -tags=functional
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/driver-service/internal/model"
	"furab-backend/services/driver-service/internal/repository"
	"furab-backend/services/driver-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.DriverRepository
	testSvc  service.DriverService
)

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// TestMain sets up the test database connection, creates schema, runs tests, and cleans up.
func TestMain(m *testing.M) {
	// Standard DSN pattern used in ride-order-service tests
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "driver_service")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// Allow overriding entirely via TEST_DB_DSN
	if envDsn := os.Getenv("TEST_DB_DSN"); envDsn != "" {
		dsn = envDsn
	}

	var err error
	testDB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	defer testDB.Close()

	// Wait for DB to be ready (max 30 seconds)
	for i := 0; i < 30; i++ {
		err = testDB.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Database is not ready: %v", err)
	}

	// Setup schema
	if err := setupSchema(); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}

	// Initialize repository and service
	testRepo = repository.NewPostgresDriverRepository(testDB)
	testSvc = service.NewDriverService(testRepo)

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS drivers (
			driver_id VARCHAR(36) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			phone VARCHAR(20) NOT NULL UNIQUE,
			vehicle_type VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'OFFLINE',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS driver_locations (
			driver_id VARCHAR(36) PRIMARY KEY,
			latitude DOUBLE PRECISION NOT NULL,
			longitude DOUBLE PRECISION NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS driver_locations")
	testDB.Exec("DROP TABLE IF EXISTS drivers")
}

func cleanupDrivers() {
	testDB.Exec("DELETE FROM driver_locations")
	testDB.Exec("DELETE FROM drivers")
}

// --- Functional Test Cases ---

func TestFunctional_CreateAndGetDriver(t *testing.T) {
	cleanupDrivers()
	ctx := context.Background()

	req := &model.CreateDriverRequest{
		DriverID:    "driver-001",
		Name:        "Budi Santoso",
		Phone:       "081234567890",
		VehicleType: "motorcycle",
	}

	// Create
	resp, err := testSvc.CreateDriver(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}
	if resp.Status != "success" {
		t.Errorf("Expected success, got %s", resp.Status)
	}
	if resp.DriverID != "driver-001" {
		t.Errorf("Expected driver-001, got %s", resp.DriverID)
	}

	// Get
	driver, err := testSvc.GetDriver(ctx, "driver-001")
	if err != nil {
		t.Fatalf("Failed to get driver: %v", err)
	}
	if driver.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, driver.Name)
	}
	if driver.Status != model.DriverStatusOffline {
		t.Errorf("Expected default status OFFLINE, got %s", driver.Status)
	}
}

func TestFunctional_DriverStatusFlow(t *testing.T) {
	cleanupDrivers()
	ctx := context.Background()

	driverID := "driver-002"
	_, err := testSvc.CreateDriver(ctx, &model.CreateDriverRequest{
		DriverID:    driverID,
		Name:        "Agus",
		Phone:       "089876543210",
		VehicleType: "car",
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Update status to ONLINE
	_, err = testSvc.UpdateStatus(ctx, driverID, string(model.DriverStatusOnline))
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify in DB
	driver, err := testSvc.GetDriver(ctx, driverID)
	if err != nil {
		t.Fatalf("GetDriver failed: %v", err)
	}
	if driver.Status != model.DriverStatusOnline {
		t.Errorf("Expected status ONLINE, got %s", driver.Status)
	}

	// Update status to BUSY
	_, err = testSvc.UpdateStatus(ctx, driverID, string(model.DriverStatusBusy))
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	driver, _ = testSvc.GetDriver(ctx, driverID)
	if driver.Status != model.DriverStatusBusy {
		t.Errorf("Expected status BUSY, got %s", driver.Status)
	}
}

func TestFunctional_DriverLocationUpdate(t *testing.T) {
	cleanupDrivers()
	ctx := context.Background()

	driverID := "driver-loc-001"
	_, err := testSvc.CreateDriver(ctx, &model.CreateDriverRequest{
		DriverID:    driverID,
		Name:        "Siti",
		Phone:       "085544332211",
		VehicleType: "motorcycle",
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Update location (INSERT via UPSERT)
	_, err = testSvc.UpdateLocation(ctx, driverID, -6.2, 106.8)
	if err != nil {
		t.Fatalf("UpdateLocation 1 failed: %v", err)
	}

	// Update location again (UPDATE via UPSERT)
	_, err = testSvc.UpdateLocation(ctx, driverID, -6.21, 106.81)
	if err != nil {
		t.Fatalf("UpdateLocation 2 failed: %v", err)
	}

	// Verify DB explicitly since there's no GetLocation in service yet
	var lat, long float64
	err = testDB.QueryRow("SELECT latitude, longitude FROM driver_locations WHERE driver_id = $1", driverID).Scan(&lat, &long)
	if err != nil {
		t.Fatalf("Failed to query location from DB: %v", err)
	}

	if lat != -6.21 || long != 106.81 {
		t.Errorf("Expected lat/long -6.21/106.81, got %f/%f", lat, long)
	}
}
