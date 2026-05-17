//go:build functional
// +build functional

// Package functional contains functional tests for matching-service.
// Functional tests access a REAL PostgreSQL database.
//
// Run with: go test ./test/functional/... -v -tags=functional
//
// Prerequisites:
//   - Docker Desktop running
//   - Run: docker compose -f deploy/docker/docker-compose.yml up -d postgres
//   - Database "matching_service" created automatically by init script
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/matching-service/internal/model"
	"furab-backend/services/matching-service/internal/repository"
	"furab-backend/services/matching-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.MatchRepository
	testSvc  service.MatchService
)

func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "matching_service")

	// Step 1: Connect to admin 'postgres' database to ensure DB exists
	adminDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort)
	
	var err error
	adminDB, err := sql.Open("pgx", adminDSN)
	if err != nil {
		log.Printf("failed to connect to admin db: %v", err)
	} else {
		// Wait for postgres to be ready
		for i := 0; i < 30; i++ {
			if err = adminDB.Ping(); err == nil {
				break
			}
			log.Printf("Waiting for Postgres... (%d/30)", i+1)
			time.Sleep(1 * time.Second)
		}

		if err == nil {
			// Create database if not exists
			_, _ = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
			adminDB.Close()
		}
	}

	// Step 2: Connect to target database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	testDB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to open db connection: %v", err)
	}
	defer testDB.Close()

	// Final check to ensure target DB is ready
	for i := 0; i < 10; i++ {
		if err = testDB.Ping(); err == nil {
			break
		}
		log.Printf("Waiting for target database %s... (%d/10)", dbName, i+1)
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		log.Fatalf("Database %s not ready: %v", dbName, err)
	}
	log.Println("Database connected!")

	if err := setupSchema(); err != nil {
		log.Fatalf("Schema setup failed: %v", err)
	}

	testRepo = repository.NewPostgresMatchRepository(testDB)
	testSvc = service.NewMatchService(testRepo, nil) // nil publisher

	code := m.Run()
	teardownSchema()
	os.Exit(code)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS match_requests (
			id VARCHAR(36) PRIMARY KEY,
			order_id VARCHAR(36) NOT NULL,
			order_type VARCHAR(10) NOT NULL,
			user_id VARCHAR(36) NOT NULL,
			pickup_lat DOUBLE PRECISION NOT NULL DEFAULT 0,
			pickup_lng DOUBLE PRECISION NOT NULL DEFAULT 0,
			pickup_address TEXT NOT NULL DEFAULT '',
			dropoff_lat DOUBLE PRECISION NOT NULL DEFAULT 0,
			dropoff_lng DOUBLE PRECISION NOT NULL DEFAULT 0,
			dropoff_address TEXT NOT NULL DEFAULT '',
			status VARCHAR(20) NOT NULL DEFAULT 'SEARCHING',
			driver_id VARCHAR(36),
			attempt_count INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 3,
			radius DOUBLE PRECISION NOT NULL DEFAULT 5.0,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_match_order_id ON match_requests(order_id);
		CREATE INDEX IF NOT EXISTS idx_match_status ON match_requests(status);
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS match_requests")
}

func cleanupMatches() {
	testDB.Exec("DELETE FROM match_requests")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func rideMatchRequest() *model.FindDriverRequest {
	return &model.FindDriverRequest{
		OrderID:   "ride-order-001",
		OrderType: "ride",
		UserID:    "func-user-001",
		PickupLocation: model.Location{
			Latitude: -6.2088, Longitude: 106.8456, Address: "Monas, Jakarta",
		},
		DropoffLocation: model.Location{
			Latitude: -6.1751, Longitude: 106.8650, Address: "Ancol, Jakarta",
		},
	}
}

func foodMatchRequest() *model.FindDriverRequest {
	return &model.FindDriverRequest{
		OrderID:   "food-order-001",
		OrderType: "food",
		UserID:    "func-user-002",
		PickupLocation: model.Location{
			Latitude: -6.1751, Longitude: 106.8650, Address: "Warteg Bahari",
		},
		DropoffLocation: model.Location{
			Latitude: -6.2088, Longitude: 106.8456, Address: "Rumah User",
		},
	}
}

// --- Functional Test Cases ---

// TestFunctional_FindDriverRide tests creating a ride match request in real DB.
func TestFunctional_FindDriverRide(t *testing.T) {
	cleanupMatches()
	ctx := context.Background()

	match, err := testSvc.FindDriver(ctx, rideMatchRequest())
	if err != nil {
		t.Fatalf("find driver: %v", err)
	}
	if match.Status != model.MatchStatusSearching {
		t.Errorf("expected SEARCHING, got: %s", match.Status)
	}
	if match.OrderType != "ride" {
		t.Errorf("expected ride, got: %s", match.OrderType)
	}
	if match.Radius != 5.0 {
		t.Errorf("expected radius 5.0, got: %.1f", match.Radius)
	}
	t.Logf("Match created: %s (type: %s, status: %s)", match.ID, match.OrderType, match.Status)

	// Verify from DB
	fetched, err := testSvc.GetMatchStatus(ctx, match.ID)
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if fetched.OrderID != "ride-order-001" {
		t.Errorf("expected order ride-order-001, got: %s", fetched.OrderID)
	}
	if fetched.PickupLocation.Address != "Monas, Jakarta" {
		t.Errorf("pickup address mismatch: %s", fetched.PickupLocation.Address)
	}
}

// TestFunctional_FindDriverFood tests creating a food match request.
func TestFunctional_FindDriverFood(t *testing.T) {
	cleanupMatches()
	ctx := context.Background()

	match, err := testSvc.FindDriver(ctx, foodMatchRequest())
	if err != nil {
		t.Fatalf("find driver: %v", err)
	}
	if match.OrderType != "food" {
		t.Errorf("expected food, got: %s", match.OrderType)
	}
	t.Logf("Food match created: %s", match.ID)
}

// TestFunctional_FullMatchFlow tests the complete matching lifecycle:
// FindDriver (SEARCHING) → OfferToDriver (OFFERED) → DriverAccept (MATCHED)
func TestFunctional_FullMatchFlow(t *testing.T) {
	cleanupMatches()
	ctx := context.Background()

	// Step 1: Find driver → SEARCHING
	match, err := testSvc.FindDriver(ctx, rideMatchRequest())
	if err != nil {
		t.Fatalf("step 1 find: %v", err)
	}
	t.Logf("Step 1 - Searching: %s", match.ID)

	// Step 2: Offer to driver → OFFERED
	offered, err := testSvc.OfferToDriver(ctx, match.ID, "driver-func-001")
	if err != nil {
		t.Fatalf("step 2 offer: %v", err)
	}
	if offered.Status != model.MatchStatusOffered {
		t.Errorf("expected OFFERED, got: %s", offered.Status)
	}
	if offered.DriverID != "driver-func-001" {
		t.Errorf("expected driver-func-001, got: %s", offered.DriverID)
	}
	if offered.AttemptCount != 1 {
		t.Errorf("expected attempt 1, got: %d", offered.AttemptCount)
	}
	t.Logf("Step 2 - Offered to: %s (attempt: %d)", offered.DriverID, offered.AttemptCount)

	// Step 3: Driver accepts → MATCHED
	matched, err := testSvc.DriverAccept(ctx, match.ID)
	if err != nil {
		t.Fatalf("step 3 accept: %v", err)
	}
	if matched.Status != model.MatchStatusMatched {
		t.Errorf("expected MATCHED, got: %s", matched.Status)
	}
	t.Logf("Step 3 - Matched! driver: %s", matched.DriverID)

	// Verify final state from DB
	final, err := testSvc.GetMatchStatus(ctx, match.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if final.Status != model.MatchStatusMatched {
		t.Errorf("DB: expected MATCHED, got: %s", final.Status)
	}
	if final.DriverID != "driver-func-001" {
		t.Errorf("DB: expected driver-func-001, got: %s", final.DriverID)
	}
}

// TestFunctional_DriverRejectAndRetry tests driver reject → back to SEARCHING → offer new driver.
func TestFunctional_DriverRejectAndRetry(t *testing.T) {
	cleanupMatches()
	ctx := context.Background()

	match, err := testSvc.FindDriver(ctx, rideMatchRequest())
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	// Offer to driver 1
	_, err = testSvc.OfferToDriver(ctx, match.ID, "driver-reject-001")
	if err != nil {
		t.Fatalf("offer 1: %v", err)
	}

	// Driver 1 rejects → back to SEARCHING
	rejected, err := testSvc.DriverReject(ctx, match.ID)
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != model.MatchStatusSearching {
		t.Errorf("expected SEARCHING after reject, got: %s", rejected.Status)
	}
	if rejected.DriverID != "" {
		t.Errorf("expected empty driver after reject, got: %s", rejected.DriverID)
	}
	t.Logf("Driver 1 rejected → back to SEARCHING (attempt: %d)", rejected.AttemptCount)

	// Verify from DB
	fromDB, err := testSvc.GetMatchStatus(ctx, match.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if fromDB.Status != model.MatchStatusSearching {
		t.Errorf("DB: expected SEARCHING, got: %s", fromDB.Status)
	}

	// Offer to driver 2
	offered2, err := testSvc.OfferToDriver(ctx, match.ID, "driver-accept-002")
	if err != nil {
		t.Fatalf("offer 2: %v", err)
	}
	if offered2.AttemptCount != 2 {
		t.Errorf("expected attempt 2, got: %d", offered2.AttemptCount)
	}
	t.Logf("Offered to driver 2: %s (attempt: %d)", offered2.DriverID, offered2.AttemptCount)

	// Driver 2 accepts
	matched, err := testSvc.DriverAccept(ctx, match.ID)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if matched.Status != model.MatchStatusMatched {
		t.Errorf("expected MATCHED, got: %s", matched.Status)
	}
	t.Logf("Matched with driver 2: %s", matched.DriverID)
}

// TestFunctional_MaxAttemptsReached tests that exceeding max attempts → FAILED.
func TestFunctional_MaxAttemptsReached(t *testing.T) {
	cleanupMatches()
	ctx := context.Background()

	match, err := testSvc.FindDriver(ctx, rideMatchRequest())
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	// Exhaust all 3 attempts
	for i := 1; i <= 3; i++ {
		driverID := fmt.Sprintf("driver-reject-%d", i)
		_, err := testSvc.OfferToDriver(ctx, match.ID, driverID)
		if err != nil {
			t.Fatalf("offer %d: %v", i, err)
		}

		result, err := testSvc.DriverReject(ctx, match.ID)
		if i < 3 {
			// Should go back to SEARCHING
			if err != nil {
				t.Fatalf("reject %d: %v", i, err)
			}
			if result.Status != model.MatchStatusSearching {
				t.Errorf("attempt %d: expected SEARCHING, got: %s", i, result.Status)
			}
			t.Logf("Attempt %d: rejected → SEARCHING", i)
		} else {
			// Attempt 3: should FAIL
			if err != service.ErrMaxAttempts {
				t.Fatalf("attempt 3: expected ErrMaxAttempts, got: %v", err)
			}
			if result.Status != model.MatchStatusFailed {
				t.Errorf("expected FAILED, got: %s", result.Status)
			}
			t.Logf("Attempt %d: max reached → FAILED", i)
		}
	}

	// Verify FAILED in DB
	final, err := testSvc.GetMatchStatus(ctx, match.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if final.Status != model.MatchStatusFailed {
		t.Errorf("DB: expected FAILED, got: %s", final.Status)
	}
}

// TestFunctional_CancelMatch tests cancelling a match (user cancel / timeout).
func TestFunctional_CancelMatch(t *testing.T) {
	cleanupMatches()
	ctx := context.Background()

	match, err := testSvc.FindDriver(ctx, rideMatchRequest())
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	cancelled, err := testSvc.CancelMatch(ctx, match.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Status != model.MatchStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", cancelled.Status)
	}
	t.Logf("Match cancelled: %s", cancelled.Status)

	// Verify in DB
	final, err := testSvc.GetMatchStatus(ctx, match.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if final.Status != model.MatchStatusCancelled {
		t.Errorf("DB: expected CANCELLED, got: %s", final.Status)
	}
}

// TestFunctional_CancelAlreadyMatched tests that matched requests cannot be cancelled.
func TestFunctional_CancelAlreadyMatched(t *testing.T) {
	cleanupMatches()
	ctx := context.Background()

	match, err := testSvc.FindDriver(ctx, rideMatchRequest())
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	testSvc.OfferToDriver(ctx, match.ID, "driver-X")
	testSvc.DriverAccept(ctx, match.ID)

	// Try to cancel MATCHED → should fail
	_, err = testSvc.CancelMatch(ctx, match.ID)
	if err == nil {
		t.Fatal("expected error: cannot cancel MATCHED")
	}
	t.Logf("Correctly rejected cancel on MATCHED: %v", err)
}

// TestFunctional_RetryExpandsRadius tests that retry expands the search radius.
func TestFunctional_RetryExpandsRadius(t *testing.T) {
	cleanupMatches()
	ctx := context.Background()

	match, err := testSvc.FindDriver(ctx, rideMatchRequest())
	if err != nil {
		t.Fatalf("find: %v", err)
	}

	// Initial radius should be 5.0
	if match.Radius != 5.0 {
		t.Errorf("expected initial radius 5.0, got: %.1f", match.Radius)
	}

	// Retry → radius should expand to 7.0
	retried, err := testSvc.RetryMatch(ctx, match.ID)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if retried.Radius != 7.0 {
		t.Errorf("expected radius 7.0 after retry, got: %.1f", retried.Radius)
	}
	t.Logf("Retry: radius expanded %.1f → %.1f km", 5.0, retried.Radius)

	// Verify in DB
	fromDB, err := testSvc.GetMatchStatus(ctx, match.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if fromDB.Radius != 7.0 {
		t.Errorf("DB: expected radius 7.0, got: %.1f", fromDB.Radius)
	}
}
