//go:build functional
// +build functional

// Package functional contains functional tests for the promo service.
// Functional tests access a REAL PostgreSQL database (bukan mock).
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

	"furab-backend/services/promo-service/internal/model"
	"furab-backend/services/promo-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.PromoRepository
	testSvc  service.PromoService
)

// TestMain sets up the test database connection, creates schema, runs tests, and cleans up.
func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "promo_service")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

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
	log.Println("Database connected!")

	// Setup schema
	if err := setupSchema(); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}

	// Initialize repository and service
	testRepo = repository.NewPostgresPromoRepository(testDB)
	testSvc = service.NewPromoService(testRepo, &mockOrderClient{}, &mockUserClient{})

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

// setupSchema creates the promos table and seeds initial data.
func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS promos (
			promo_id VARCHAR(36) PRIMARY KEY,
			code VARCHAR(50) UNIQUE NOT NULL,
			discount_type VARCHAR(20) NOT NULL,
			discount_value DOUBLE PRECISION NOT NULL,
			min_purchase DOUBLE PRECISION NOT NULL DEFAULT 0,
			max_discount DOUBLE PRECISION NOT NULL DEFAULT 0,
			expiry_date TIMESTAMP WITH TIME ZONE NOT NULL,
			usage_limit INTEGER NOT NULL DEFAULT 0,
			usage_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		DELETE FROM promos;
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS promos")
}

func cleanupPromos() {
	testDB.Exec("DELETE FROM promos")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Mocks for External Clients ---

type mockOrderClient struct{}

func (m *mockOrderClient) ValidateOrderPromo(ctx context.Context, orderID, promoCode string) (bool, error) {
	return true, nil
}

type mockUserClient struct{}

func (m *mockUserClient) ValidateUserPromo(ctx context.Context, userID, promoCode string) (bool, error) {
	return true, nil
}

// --- Functional Test Cases ---

func TestFunctional_ValidatePromo_Percentage(t *testing.T) {
	cleanupPromos()
	ctx := context.Background()

	// Insert a percentage promo
	promoCode := "PERCENT10"
	_, err := testDB.Exec(`
		INSERT INTO promos (promo_id, code, discount_type, discount_value, min_purchase, max_discount, expiry_date, usage_limit, usage_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		"p1", promoCode, "percentage", 0.1, 50000.0, 10000.0, time.Now().Add(24*time.Hour), 100, 0)
	if err != nil {
		t.Fatalf("failed to seed promo: %v", err)
	}

	// Case 1: Valid usage
	res, err := testSvc.ValidatePromo(ctx, promoCode, "user-1", "order-1", 100000.0)
	if err != nil {
		t.Fatalf("failed to validate promo: %v", err)
	}

	if !res.IsValid {
		t.Errorf("expected promo to be valid, got error: %s", res.ErrorMessage)
	}
	if res.DiscountAmount != 10000 {
		t.Errorf("expected 10000 discount (10%% of 100k), got %.2f", res.DiscountAmount)
	}
	if res.FinalAmount != 90000 {
		t.Errorf("expected final amount 90000, got %.2f", res.FinalAmount)
	}

	// Verify usage_count incremented in DB
	var count int
	err = testDB.QueryRow("SELECT usage_count FROM promos WHERE code = $1", promoCode).Scan(&count)
	if err != nil {
		t.Fatalf("failed to check usage count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected usage_count 1, got %d", count)
	}

	t.Logf("Percentage promo validated and usage incremented: %s", promoCode)
}

func TestFunctional_ValidatePromo_Fixed(t *testing.T) {
	cleanupPromos()
	ctx := context.Background()

	// Insert a fixed discount promo
	promoCode := "FIXED5K"
	_, err := testDB.Exec(`
		INSERT INTO promos (promo_id, code, discount_type, discount_value, min_purchase, max_discount, expiry_date, usage_limit, usage_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		"p2", promoCode, "fixed", 5000.0, 20000.0, 0.0, time.Now().Add(24*time.Hour), 50, 0)
	if err != nil {
		t.Fatalf("failed to seed promo: %v", err)
	}

	res, err := testSvc.ValidatePromo(ctx, promoCode, "user-2", "order-2", 30000.0)
	if err != nil {
		t.Fatalf("failed to validate promo: %v", err)
	}

	if !res.IsValid {
		t.Errorf("expected promo to be valid, got error: %s", res.ErrorMessage)
	}
	if res.DiscountAmount != 5000 {
		t.Errorf("expected 5000 discount, got %.2f", res.DiscountAmount)
	}
	if res.FinalAmount != 25000 {
		t.Errorf("expected final amount 25000, got %.2f", res.FinalAmount)
	}

	t.Logf("Fixed promo validated: %s", promoCode)
}

func TestFunctional_ValidatePromo_Expired(t *testing.T) {
	cleanupPromos()
	ctx := context.Background()

	promoCode := "EXPIRED"
	_, err := testDB.Exec(`
		INSERT INTO promos (promo_id, code, discount_type, discount_value, min_purchase, expiry_date, usage_limit, usage_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		"p3", promoCode, "fixed", 1000.0, 0.0, time.Now().Add(-1*time.Hour), 10, 0)
	if err != nil {
		t.Fatalf("failed to seed promo: %v", err)
	}

	res, err := testSvc.ValidatePromo(ctx, promoCode, "user-3", "order-3", 10000.0)
	if err != nil {
		t.Fatalf("svc error: %v", err)
	}

	if res.IsValid {
		t.Error("expected promo to be invalid (expired)")
	}
	if res.ErrorMessage != "promo has expired" {
		t.Errorf("expected 'promo has expired' error, got: %s", res.ErrorMessage)
	}

	t.Log("Correctly rejected expired promo")
}

func TestFunctional_ValidatePromo_LimitReached(t *testing.T) {
	cleanupPromos()
	ctx := context.Background()

	promoCode := "FULL"
	_, err := testDB.Exec(`
		INSERT INTO promos (promo_id, code, discount_type, discount_value, usage_limit, usage_count, expiry_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		"p4", promoCode, "fixed", 1000.0, 5, 5, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("failed to seed promo: %v", err)
	}

	res, err := testSvc.ValidatePromo(ctx, promoCode, "user-4", "order-4", 10000.0)
	if err != nil {
		t.Fatalf("svc error: %v", err)
	}

	if res.IsValid {
		t.Error("expected promo to be invalid (limit reached)")
	}
	if res.ErrorMessage != "promo usage limit exceeded" {
		t.Errorf("expected 'promo usage limit exceeded' error, got: %s", res.ErrorMessage)
	}

	t.Log("Correctly rejected full promo")
}

func TestFunctional_ValidatePromo_MinPurchase(t *testing.T) {
	cleanupPromos()
	ctx := context.Background()

	promoCode := "MIN100K"
	_, err := testDB.Exec(`
		INSERT INTO promos (promo_id, code, discount_type, discount_value, min_purchase, expiry_date)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		"p5", promoCode, "fixed", 10000.0, 100000.0, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("failed to seed promo: %v", err)
	}

	// Try with 50k
	res, err := testSvc.ValidatePromo(ctx, promoCode, "user-5", "order-5", 50000.0)
	if err != nil {
		t.Fatalf("svc error: %v", err)
	}

	if res.IsValid {
		t.Error("expected promo to be invalid (min purchase not met)")
	}
	t.Logf("Correctly rejected due to min purchase: %s", res.ErrorMessage)
}
