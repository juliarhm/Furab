//go:build functional
// +build functional

// Package functional contains functional tests for the wallet service.
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

	"furab-backend/services/wallet-service/internal/model"
	"furab-backend/services/wallet-service/internal/repository"
	"furab-backend/services/wallet-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.WalletRepository
	testSvc  service.WalletService
)

// TestMain sets up the test database connection, creates schema, runs tests, and cleans up.
func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "wallet_service")

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
	testRepo = repository.NewPostgresWalletRepository(testDB)
	testSvc = service.NewWalletService(testRepo)

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

// setupSchema creates the wallets and wallet_transactions tables.
func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS wallets (
			wallet_id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) UNIQUE NOT NULL,
			balance DOUBLE PRECISION NOT NULL DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS wallet_transactions (
			transaction_id VARCHAR(36) PRIMARY KEY,
			wallet_id VARCHAR(36) NOT NULL REFERENCES wallets(wallet_id),
			reference_id VARCHAR(100),
			amount DOUBLE PRECISION NOT NULL,
			type VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL,
			current_balance DOUBLE PRECISION NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		DELETE FROM wallet_transactions;
		DELETE FROM wallets;
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS wallet_transactions")
	testDB.Exec("DROP TABLE IF EXISTS wallets")
}

func cleanupTables() {
	testDB.Exec("DELETE FROM wallet_transactions")
	testDB.Exec("DELETE FROM wallets")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Helper Functions ---

func createTestWallet(t *testing.T, userID string, balance float64) string {
	walletID := fmt.Sprintf("wallet-%s", userID)
	_, err := testDB.Exec(`
		INSERT INTO wallets (wallet_id, user_id, balance, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`,
		walletID, userID, balance, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("failed to create test wallet: %v", err)
	}
	return walletID
}

// --- Functional Test Cases ---

func TestFunctional_WalletLifecycle(t *testing.T) {
	cleanupTables()
	ctx := context.Background()
	userID := "user-func-001"
	
	// 1. Create Wallet with initial 0 balance
	createTestWallet(t, userID, 0)

	// 2. Credit Balance (Top up 100,000)
	res, err := testSvc.CreditBalance(ctx, userID, 100000.0, "TOPUP-001")
	if err != nil {
		t.Fatalf("failed to credit balance: %v", err)
	}
	if res.CurrentBalance != 100000.0 {
		t.Errorf("expected balance 100000, got %.2f", res.CurrentBalance)
	}
	t.Log("Step 1: Top up success")

	// 3. Hold Balance (Reserve 50,000 for order)
	refHold := "ORDER-HOLD-001"
	res, err = testSvc.HoldBalance(ctx, userID, 50000.0, refHold)
	if err != nil {
		t.Fatalf("failed to hold balance: %v", err)
	}
	if res.CurrentBalance != 50000.0 {
		t.Errorf("expected balance 50000, got %.2f", res.CurrentBalance)
	}
	t.Log("Step 2: Balance held success")

	// 4. Debit Balance (Capture 50,000)
	refDebit := "ORDER-PAY-001"
	res, err = testSvc.DebitBalance(ctx, userID, 50000.0, refDebit)
	if err != nil {
		t.Fatalf("failed to debit balance: %v", err)
	}
	if res.CurrentBalance != 0.0 {
		t.Errorf("expected balance 0, got %.2f", res.CurrentBalance)
	}
	t.Log("Step 3: Debit success")

	// 5. Refund (Refund 20,000)
	res, err = testSvc.Refund(ctx, userID, 20000.0, refDebit) // Uses refDebit as valid reference for refund
	if err != nil {
		t.Fatalf("failed to refund: %v", err)
	}
	if res.CurrentBalance != 20000.0 {
		t.Errorf("expected balance 20000, got %.2f", res.CurrentBalance)
	}
	t.Log("Step 4: Refund success")
}

func TestFunctional_Wallet_Idempotency(t *testing.T) {
	cleanupTables()
	ctx := context.Background()
	userID := "user-func-002"
	createTestWallet(t, userID, 10000.0)

	ref := "IDEM-001"
	// First call
	res1, err := testSvc.CreditBalance(ctx, userID, 5000.0, ref)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call with same reference
	res2, err := testSvc.CreditBalance(ctx, userID, 5000.0, ref)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if res1.TransactionID != res2.TransactionID {
		t.Errorf("expected same transaction ID, got %s and %s", res1.TransactionID, res2.TransactionID)
	}
	if res2.CurrentBalance != 15000.0 {
		t.Errorf("expected balance 15000 (no double credit), got %.2f", res2.CurrentBalance)
	}
	t.Log("Idempotency verified: no double credit for same reference")
}

func TestFunctional_Wallet_InsufficientBalance(t *testing.T) {
	cleanupTables()
	ctx := context.Background()
	userID := "user-func-003"
	createTestWallet(t, userID, 1000.0)

	_, err := testSvc.DebitBalance(ctx, userID, 5000.0, "FAIL-001")
	if err != service.ErrInsufficientBalance {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}

	// Verify balance hasn't changed
	w, _ := testRepo.GetByUserID(ctx, userID)
	if w.Balance != 1000.0 {
		t.Errorf("expected balance 1000 to remain, got %.2f", w.Balance)
	}
	t.Log("Correctly rejected debit due to insufficient balance")
}

func TestFunctional_Wallet_ReleaseHold(t *testing.T) {
	cleanupTables()
	ctx := context.Background()
	userID := "user-func-004"
	createTestWallet(t, userID, 50000.0)

	ref := "HOLD-REF-001"
	// 1. Hold
	_, err := testSvc.HoldBalance(ctx, userID, 20000.0, ref)
	if err != nil {
		t.Fatalf("hold failed: %v", err)
	}

	// 2. Release
	res, err := testSvc.ReleaseBalance(ctx, userID, 20000.0, ref)
	if err != nil {
		t.Fatalf("release failed: %v", err)
	}

	if res.CurrentBalance != 50000.0 {
		t.Errorf("expected balance back to 50000, got %.2f", res.CurrentBalance)
	}
	t.Log("Release balance verified")
}

func TestFunctional_Wallet_InvalidRelease(t *testing.T) {
	cleanupTables()
	ctx := context.Background()
	userID := "user-func-005"
	createTestWallet(t, userID, 10000.0)

	// Try to release without a previous hold
	_, err := testSvc.ReleaseBalance(ctx, userID, 1000.0, "NON-EXISTENT-HOLD")
	if err != service.ErrReferenceNotFound {
		t.Fatalf("expected ErrReferenceNotFound, got %v", err)
	}
	t.Log("Correctly rejected release for non-existent hold")
}
