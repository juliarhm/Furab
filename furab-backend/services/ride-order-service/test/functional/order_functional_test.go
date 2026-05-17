//go:build functional
// +build functional

// Package functional contains functional tests for the ride order service.
// Functional tests access a REAL PostgreSQL database (bukan mock).
//
// Run with: go test ./test/functional/... -v -tags=functional
//
// Prerequisites:
//   - Docker Desktop running
//   - Run: docker compose -f deploy/docker/docker-compose.yml up -d postgres
//   - Database "ride_order_service" created automatically by init script
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"furab-backend/services/ride-order-service/internal/model"
	"furab-backend/services/ride-order-service/internal/repository"
	"furab-backend/services/ride-order-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.OrderRepository
	testSvc  service.OrderService
)

// TestMain sets up the test database connection, creates schema, runs tests, and cleans up.
func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "ride_order_service")

	// Step 1: Connect to default 'postgres' database to auto-create target DB
	adminDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort)
	adminDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		log.Fatalf("Failed to connect to admin database: %v", err)
	}
	for i := 0; i < 30; i++ {
		if err = adminDB.Ping(); err == nil {
			break
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Database is not ready: %v", err)
	}
	_, err = adminDB.Exec("CREATE DATABASE " + dbName)
	if err != nil && !contains(err.Error(), "already exists") {
		log.Printf("Warning: could not create database %s: %v", dbName, err)
	}
	adminDB.Close()

	// Step 2: Connect to the target database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	testDB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	defer testDB.Close()

	for i := 0; i < 30; i++ {
		if err = testDB.Ping(); err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Database is not ready: %v", err)
	}
	log.Println("Database connected!")

	// Setup schema
	if err := setupSchema(); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}

	// Initialize repository and service
	// Publisher = nil → no events published during functional tests
	testRepo = repository.NewPostgresOrderRepository(testDB)
	testSvc = service.NewOrderService(testRepo, nil)

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

// setupSchema creates the ride_orders table with all columns matching the updated model.
func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS ride_orders (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL,
			driver_id VARCHAR(36),
			pickup_lat DOUBLE PRECISION NOT NULL,
			pickup_lng DOUBLE PRECISION NOT NULL,
			pickup_address TEXT NOT NULL,
			dropoff_lat DOUBLE PRECISION NOT NULL,
			dropoff_lng DOUBLE PRECISION NOT NULL,
			dropoff_address TEXT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			payment_status VARCHAR(20) NOT NULL DEFAULT 'NONE',
			fare DOUBLE PRECISION NOT NULL DEFAULT 0,
			distance DOUBLE PRECISION NOT NULL DEFAULT 0,
			estimated_duration INTEGER NOT NULL DEFAULT 0,
			cancelled_by VARCHAR(20),
			cancel_reason TEXT,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_ride_orders_user_id ON ride_orders(user_id);
		CREATE INDEX IF NOT EXISTS idx_ride_orders_driver_id ON ride_orders(driver_id);
		CREATE INDEX IF NOT EXISTS idx_ride_orders_status ON ride_orders(status);
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS ride_orders")
}

func cleanupOrders() {
	testDB.Exec("DELETE FROM ride_orders")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// --- Functional Test Cases ---

// TestFunctional_CreateAndGetOrder tests creating and retrieving a ride order from real DB.
func TestFunctional_CreateAndGetOrder(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	req := &model.CreateRideOrderRequest{
		UserID: "func-user-001",
		PickupLocation: model.Location{
			Latitude: -6.2088, Longitude: 106.8456, Address: "Monas, Jakarta Pusat",
		},
		DropoffLocation: model.Location{
			Latitude: -6.1751, Longitude: 106.8650, Address: "Ancol, Jakarta Utara",
		},
	}

	// Create order → INSERT ke PostgreSQL beneran
	order, err := testSvc.CreateOrder(ctx, req)
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	if order.ID == "" {
		t.Fatal("expected non-empty order ID")
	}
	if order.Status != model.RideStatusPending {
		t.Errorf("expected status PENDING, got: %s", order.Status)
	}
	if order.Fare <= 0 {
		t.Error("expected fare > 0")
	}
	t.Logf("Created order: %s (fare: Rp %.0f)", order.ID, order.Fare)

	// Get order → SELECT dari PostgreSQL beneran
	fetched, err := testSvc.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to get order: %v", err)
	}

	if fetched.ID != order.ID {
		t.Errorf("expected ID %s, got: %s", order.ID, fetched.ID)
	}
	if fetched.PickupLocation.Address != req.PickupLocation.Address {
		t.Errorf("expected pickup %s, got: %s", req.PickupLocation.Address, fetched.PickupLocation.Address)
	}
	if fetched.Fare != order.Fare {
		t.Errorf("expected fare %.2f, got: %.2f", order.Fare, fetched.Fare)
	}
}

// TestFunctional_FullRideFlow tests the COMPLETE ride lifecycle in real DB.
// Flow: PENDING → ASSIGNED → PICKING_UP → ON_THE_WAY → COMPLETED
func TestFunctional_FullRideFlow(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	// Step 1: Create order
	req := &model.CreateRideOrderRequest{
		UserID: "func-user-002",
		PickupLocation: model.Location{
			Latitude: -6.2088, Longitude: 106.8456, Address: "Sudirman, Jakarta",
		},
		DropoffLocation: model.Location{
			Latitude: -6.2600, Longitude: 106.7810, Address: "Senayan, Jakarta",
		},
	}

	order, err := testSvc.CreateOrder(ctx, req)
	if err != nil {
		t.Fatalf("step 1 - create: %v", err)
	}
	t.Logf("Step 1 - Created: %s (status: %s)", order.ID, order.Status)

	// Step 2: Assign driver
	assigned, err := testSvc.AssignDriver(ctx, order.ID, "driver-func-001")
	if err != nil {
		t.Fatalf("step 2 - assign: %v", err)
	}
	if assigned.Status != model.RideStatusAssigned {
		t.Errorf("expected ASSIGNED, got: %s", assigned.Status)
	}
	if assigned.PaymentStatus != model.PaymentStatusAuthorized {
		t.Errorf("expected payment AUTHORIZED, got: %s", assigned.PaymentStatus)
	}
	t.Logf("Step 2 - Assigned driver: %s (status: %s)", assigned.DriverID, assigned.Status)

	// Step 3: Picking up (driver heading to pickup)
	pickingUp, err := testSvc.PickingUp(ctx, order.ID)
	if err != nil {
		t.Fatalf("step 3 - picking up: %v", err)
	}
	if pickingUp.Status != model.RideStatusPickingUp {
		t.Errorf("expected PICKING_UP, got: %s", pickingUp.Status)
	}
	t.Logf("Step 3 - Picking up (status: %s)", pickingUp.Status)

	// Step 4: On the way (passenger picked up)
	onTheWay, err := testSvc.OnTheWay(ctx, order.ID)
	if err != nil {
		t.Fatalf("step 4 - on the way: %v", err)
	}
	if onTheWay.Status != model.RideStatusOnTheWay {
		t.Errorf("expected ON_THE_WAY, got: %s", onTheWay.Status)
	}
	t.Logf("Step 4 - On the way (status: %s)", onTheWay.Status)

	// Step 5: Complete ride
	completed, err := testSvc.CompleteRide(ctx, order.ID)
	if err != nil {
		t.Fatalf("step 5 - complete: %v", err)
	}
	if completed.Status != model.RideStatusCompleted {
		t.Errorf("expected COMPLETED, got: %s", completed.Status)
	}
	if completed.PaymentStatus != model.PaymentStatusCaptured {
		t.Errorf("expected payment CAPTURED, got: %s", completed.PaymentStatus)
	}
	t.Logf("Step 5 - Completed (status: %s, fare: Rp %.0f)", completed.Status, completed.Fare)

	// Verify final state from DB
	final, err := testSvc.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("failed to verify final state: %v", err)
	}
	if final.Status != model.RideStatusCompleted {
		t.Errorf("DB final status: expected COMPLETED, got: %s", final.Status)
	}
	if final.PaymentStatus != model.PaymentStatusCaptured {
		t.Errorf("DB payment status: expected CAPTURED, got: %s", final.PaymentStatus)
	}
}

// TestFunctional_UserCancelRide tests user cancelling a ride.
// Flow: PENDING → CANCELLED (wallet unlocked)
func TestFunctional_UserCancelRide(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	req := &model.CreateRideOrderRequest{
		UserID: "func-user-003",
		PickupLocation: model.Location{
			Latitude: -6.2088, Longitude: 106.8456, Address: "Thamrin, Jakarta",
		},
		DropoffLocation: model.Location{
			Latitude: -6.3000, Longitude: 106.8500, Address: "Kuningan, Jakarta",
		},
	}

	order, err := testSvc.CreateOrder(ctx, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	cancelReq := &model.CancelRideRequest{
		CancelledBy:  "user",
		CancelReason: "changed my mind",
	}
	cancelled, err := testSvc.CancelRide(ctx, order.ID, cancelReq)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Status != model.RideStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", cancelled.Status)
	}
	if cancelled.CancelledBy != "user" {
		t.Errorf("expected cancelled_by user, got: %s", cancelled.CancelledBy)
	}
	if cancelled.PaymentStatus != model.PaymentStatusRefunded {
		t.Errorf("expected payment REFUNDED, got: %s", cancelled.PaymentStatus)
	}
	t.Logf("Cancelled by %s: %s", cancelled.CancelledBy, cancelled.CancelReason)

	// Verify in DB
	final, err := testSvc.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if final.CancelledBy != "user" {
		t.Errorf("DB cancelled_by: expected user, got: %s", final.CancelledBy)
	}
}

// TestFunctional_DriverCancelAndRematch tests driver cancel → re-match.
// Flow: PENDING → ASSIGNED → (driver cancel) → PENDING (re-match)
func TestFunctional_DriverCancelAndRematch(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	req := &model.CreateRideOrderRequest{
		UserID: "func-user-004",
		PickupLocation: model.Location{
			Latitude: -6.2088, Longitude: 106.8456, Address: "Blok M, Jakarta",
		},
		DropoffLocation: model.Location{
			Latitude: -6.3500, Longitude: 106.8300, Address: "Pondok Indah, Jakarta",
		},
	}

	order, err := testSvc.CreateOrder(ctx, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Assign driver
	_, err = testSvc.AssignDriver(ctx, order.ID, "driver-func-cancel")
	if err != nil {
		t.Fatalf("assign: %v", err)
	}

	// Driver cancels → order goes back to PENDING for re-matching
	rematched, err := testSvc.DriverCancelRide(ctx, order.ID)
	if err != nil {
		t.Fatalf("driver cancel: %v", err)
	}
	if rematched.Status != model.RideStatusPending {
		t.Errorf("expected PENDING after driver cancel, got: %s", rematched.Status)
	}
	if rematched.DriverID != "" {
		t.Errorf("expected empty driver after cancel, got: %s", rematched.DriverID)
	}
	if rematched.PaymentStatus != model.PaymentStatusAuthorized {
		t.Errorf("expected payment still AUTHORIZED, got: %s", rematched.PaymentStatus)
	}
	t.Logf("Driver cancelled → re-match (status: %s, driver: '%s')", rematched.Status, rematched.DriverID)

	// Assign new driver
	reassigned, err := testSvc.AssignDriver(ctx, order.ID, "driver-func-new")
	if err != nil {
		t.Fatalf("re-assign: %v", err)
	}
	if reassigned.DriverID != "driver-func-new" {
		t.Errorf("expected new driver, got: %s", reassigned.DriverID)
	}
	t.Logf("Re-matched with new driver: %s", reassigned.DriverID)
}

// TestFunctional_InvalidTransitions tests that invalid status transitions are rejected.
func TestFunctional_InvalidTransitions(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	req := &model.CreateRideOrderRequest{
		UserID: "func-user-005",
		PickupLocation: model.Location{
			Latitude: -6.2088, Longitude: 106.8456, Address: "Gambir, Jakarta",
		},
		DropoffLocation: model.Location{
			Latitude: -6.3000, Longitude: 106.8500, Address: "Cikini, Jakarta",
		},
	}

	order, err := testSvc.CreateOrder(ctx, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Cannot complete PENDING order directly
	_, err = testSvc.CompleteRide(ctx, order.ID)
	if err == nil {
		t.Fatal("expected error when completing PENDING order")
	}
	t.Logf("Correctly rejected PENDING → COMPLETED: %v", err)

	// Cannot picking_up from PENDING (must be ASSIGNED first)
	_, err = testSvc.PickingUp(ctx, order.ID)
	if err == nil {
		t.Fatal("expected error when picking up PENDING order")
	}
	t.Logf("Correctly rejected PENDING → PICKING_UP: %v", err)

	// Cannot on_the_way from PENDING
	_, err = testSvc.OnTheWay(ctx, order.ID)
	if err == nil {
		t.Fatal("expected error when on_the_way PENDING order")
	}
	t.Logf("Correctly rejected PENDING → ON_THE_WAY: %v", err)
}

// TestFunctional_GetUserOrders tests retrieving multiple orders with pagination.
func TestFunctional_GetUserOrders(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()
	userID := "func-user-006"

	// Create 3 orders
	for i := 0; i < 3; i++ {
		req := &model.CreateRideOrderRequest{
			UserID: userID,
			PickupLocation: model.Location{
				Latitude:  -6.2088 + float64(i)*0.01,
				Longitude: 106.8456,
				Address:   fmt.Sprintf("Pickup %d", i+1),
			},
			DropoffLocation: model.Location{
				Latitude:  -6.3000 + float64(i)*0.01,
				Longitude: 106.8500,
				Address:   fmt.Sprintf("Dropoff %d", i+1),
			},
		}

		_, err := testSvc.CreateOrder(ctx, req)
		if err != nil {
			t.Fatalf("create order %d: %v", i+1, err)
		}
	}

	// Get all orders
	orders, total, err := testSvc.GetUserOrders(ctx, userID, 10, 0)
	if err != nil {
		t.Fatalf("get orders: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got: %d", total)
	}
	if len(orders) != 3 {
		t.Errorf("expected 3 orders, got: %d", len(orders))
	}
	t.Logf("Found %d orders (total: %d)", len(orders), total)

	// Test pagination: page 1, limit 2
	page1, _, err := testSvc.GetUserOrders(ctx, userID, 2, 0)
	if err != nil {
		t.Fatalf("pagination: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 orders in page 1, got: %d", len(page1))
	}

	// Test pagination: page 2, limit 2
	page2, _, err := testSvc.GetUserOrders(ctx, userID, 2, 2)
	if err != nil {
		t.Fatalf("pagination page 2: %v", err)
	}
	if len(page2) != 1 {
		t.Errorf("expected 1 order in page 2, got: %d", len(page2))
	}
}

// TestFunctional_CancelFromPickingUp tests cancelling while driver is heading to pickup.
func TestFunctional_CancelFromPickingUp(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	req := &model.CreateRideOrderRequest{
		UserID: "func-user-007",
		PickupLocation: model.Location{
			Latitude: -6.2088, Longitude: 106.8456, Address: "Menteng, Jakarta",
		},
		DropoffLocation: model.Location{
			Latitude: -6.1751, Longitude: 106.8650, Address: "Kemang, Jakarta",
		},
	}

	order, err := testSvc.CreateOrder(ctx, req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Assign → Picking up → Cancel
	_, err = testSvc.AssignDriver(ctx, order.ID, "driver-func-007")
	if err != nil {
		t.Fatalf("assign: %v", err)
	}

	_, err = testSvc.PickingUp(ctx, order.ID)
	if err != nil {
		t.Fatalf("picking up: %v", err)
	}

	cancelled, err := testSvc.CancelRide(ctx, order.ID, &model.CancelRideRequest{
		CancelledBy: "user", CancelReason: "emergency",
	})
	if err != nil {
		t.Fatalf("cancel from picking_up: %v", err)
	}
	if cancelled.Status != model.RideStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", cancelled.Status)
	}
	t.Logf("Cancelled from PICKING_UP (reason: %s)", cancelled.CancelReason)
}
