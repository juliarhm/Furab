//go:build functional
// +build functional

// Package functional contains functional tests for the settlement service.
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

	"furab-backend/services/settlement-service/internal/model"
	"furab-backend/services/settlement-service/internal/repository"
	"furab-backend/services/settlement-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.SettlementRepository
	testSvc  service.SettlementService
)

// TestMain sets up the test database connection, creates schema, runs tests, and cleans up.
func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "settlement_service")

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
	testRepo = repository.NewPostgresSettlementRepository(testDB)
	testSvc = service.NewSettlementService(
		testRepo,
		&mockWalletClient{},
		&mockDriverClient{},
		&mockMerchantClient{},
	)

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

// setupSchema creates the settlements table.
func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS settlements (
			settlement_id VARCHAR(36) PRIMARY KEY,
			payment_id VARCHAR(36) NOT NULL,
			order_id VARCHAR(36) NOT NULL,
			total_amount DOUBLE PRECISION NOT NULL,
			driver_amount DOUBLE PRECISION NOT NULL,
			merchant_amount DOUBLE PRECISION NOT NULL,
			platform_fee DOUBLE PRECISION NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			idempotency_key VARCHAR(100) UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		DELETE FROM settlements;
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS settlements")
}

func cleanupSettlements() {
	testDB.Exec("DELETE FROM settlements")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Mocks for External Clients ---

type mockWalletClient struct{}

func (m *mockWalletClient) CreditBalance(ctx context.Context, walletID string, amount float64, referenceID string) error {
	return nil
}

type mockDriverClient struct{}

func (m *mockDriverClient) GetDriverWalletIDByOrderID(ctx context.Context, orderID string) (string, error) {
	if orderID == "invalid-order" {
		return "", fmt.Errorf("driver not found")
	}
	return "drv-wallet-001", nil
}

type mockMerchantClient struct{}

func (m *mockMerchantClient) GetMerchantWalletIDByOrderID(ctx context.Context, orderID string) (string, error) {
	return "mer-wallet-001", nil
}

// --- Functional Test Cases ---

func TestFunctional_ProcessSettlement_Success(t *testing.T) {
	cleanupSettlements()
	ctx := context.Background()

	req := &model.ProcessSettlementRequest{
		PaymentID:   "pay-001",
		OrderID:     "order-001",
		TotalAmount: 100000.0,
	}

	// 1. Process Settlement
	res, err := testSvc.ProcessSettlement(ctx, req)
	if err != nil {
		t.Fatalf("failed to process settlement: %v", err)
	}

	if res.Status != "SUCCESS" {
		t.Errorf("expected status SUCCESS, got %s", res.Status)
	}

	// 80% driver, 15% merchant, 5% platform
	if res.DriverAmount != 80000 {
		t.Errorf("expected driver amount 80000, got %.2f", res.DriverAmount)
	}
	if res.MerchantAmount != 15000 {
		t.Errorf("expected merchant amount 15000, got %.2f", res.MerchantAmount)
	}
	if res.PlatformFee != 5000 {
		t.Errorf("expected platform fee 5000, got %.2f", res.PlatformFee)
	}

	// 2. Verify in DB
	settlement, err := testRepo.GetSettlementByPaymentID(ctx, "pay-001")
	if err != nil {
		t.Fatalf("failed to get from repo: %v", err)
	}
	if settlement == nil {
		t.Fatal("settlement not found in DB")
	}
	if settlement.Status != model.StatusSuccess {
		t.Errorf("expected DB status success, got %s", settlement.Status)
	}

	// 3. Test Idempotency (Repeat call with same PaymentID)
	res2, err := testSvc.ProcessSettlement(ctx, req)
	if err != nil {
		t.Fatalf("idempotent call failed: %v", err)
	}
	if res2.Status != "SUCCESS" {
		t.Errorf("expected status SUCCESS on second call, got %s", res2.Status)
	}
	if res2.DriverAmount != res.DriverAmount {
		t.Error("amounts should match on idempotent call")
	}

	t.Logf("Settlement processed and verified: %s", settlement.ID)
}

func TestFunctional_ProcessSettlement_FailedDriver(t *testing.T) {
	cleanupSettlements()
	ctx := context.Background()

	req := &model.ProcessSettlementRequest{
		PaymentID:   "pay-002",
		OrderID:     "invalid-order", // mockDriverClient will fail this
		TotalAmount: 50000.0,
	}

	_, err := testSvc.ProcessSettlement(ctx, req)
	if err == nil {
		t.Fatal("expected error due to invalid driver, got nil")
	}

	// Verify status in DB is FAILED
	settlement, _ := testRepo.GetSettlementByPaymentID(ctx, "pay-002")
	if settlement == nil {
		t.Fatal("settlement should still be created in DB")
	}
	if settlement.Status != model.StatusFailed {
		t.Errorf("expected status FAILED in DB, got %s", settlement.Status)
	}

	t.Log("Correctly handled settlement failure for invalid driver")
}
