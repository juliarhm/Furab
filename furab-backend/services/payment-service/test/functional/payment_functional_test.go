//go:build functional
// +build functional

// Package functional contains functional tests for payment-service.
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

	"furab-backend/services/payment-service/internal/model"
	"furab-backend/services/payment-service/internal/repository"
	"furab-backend/services/payment-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.PaymentRepository
	testSvc  service.PaymentService
)

// TestMain sets up the test database connection, creates schema, runs tests, and cleans up.
func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "payment_service")

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

	// Initialize repository and service with mocks for external clients
	testRepo = repository.NewPostgresPaymentRepository(testDB)
	testSvc = service.NewPaymentService(
		testRepo,
		&mockPricingClient{},
		&mockPromoClient{},
		&mockWalletClient{},
		&mockSettlementClient{},
	)

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

// setupSchema creates the necessary tables for payment-service.
func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS payment_methods (
			id VARCHAR(36) PRIMARY KEY,
			method_name VARCHAR(100) NOT NULL,
			provider VARCHAR(50) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS payments (
			id VARCHAR(36) PRIMARY KEY,
			order_id VARCHAR(36) NOT NULL,
			user_id VARCHAR(36) NOT NULL,
			amount DOUBLE PRECISION NOT NULL,
			final_amount DOUBLE PRECISION NOT NULL,
			method_id VARCHAR(36) NOT NULL REFERENCES payment_methods(id),
			payment_detail TEXT NOT NULL,
			payment_status VARCHAR(20) NOT NULL DEFAULT 'pending',
			transaction_reference VARCHAR(100) NOT NULL,
			idempotency_key VARCHAR(100) UNIQUE,
			transaction_time TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS payment_logs (
			id SERIAL PRIMARY KEY,
			payment_id VARCHAR(36) NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		-- Seed a default payment method
		INSERT INTO payment_methods (id, method_name, provider)
		VALUES ('WALLET', 'Furab Wallet', 'Internal')
		ON CONFLICT (id) DO NOTHING;
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS payment_logs")
	testDB.Exec("DROP TABLE IF EXISTS payments")
	testDB.Exec("DROP TABLE IF EXISTS payment_methods")
}

func cleanupTables() {
	testDB.Exec("DELETE FROM payment_logs")
	testDB.Exec("DELETE FROM payments")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Mocks for External Clients ---

type mockPricingClient struct{}
func (m *mockPricingClient) GetTotalAmount(ctx context.Context, orderID string) (float64, error) {
	return 50000, nil
}

type mockPromoClient struct{}
func (m *mockPromoClient) ApplyPromo(ctx context.Context, promoCode string, totalAmount float64) (float64, float64, error) {
	if promoCode == "DISKON10" {
		return totalAmount - 10000, 10000, nil
	}
	return totalAmount, 0, nil
}

type mockWalletClient struct{}
func (m *mockWalletClient) LockBalance(ctx context.Context, userID string, amount float64, reference string) error { return nil }
func (m *mockWalletClient) UnlockBalance(ctx context.Context, userID string, amount float64, reference string) error { return nil }
func (m *mockWalletClient) DeductBalance(ctx context.Context, userID string, amount float64, reference string) error { return nil }
func (m *mockWalletClient) CreditBalance(ctx context.Context, userID string, amount float64, reference string) error { return nil }

type mockSettlementClient struct{}
func (m *mockSettlementClient) TriggerSettlement(ctx context.Context, paymentID, orderID string, finalAmount float64) error { return nil }

// --- Functional Test Cases ---

func TestFunctional_PaymentLifecycle(t *testing.T) {
	cleanupTables()
	ctx := context.Background()

	// Step 1: Initiate Payment
	req := &model.InitiatePaymentRequest{
		OrderID:        "order-func-001",
		UserID:         "user-func-001",
		PaymentMethod:  "WALLET",
		PaymentDetail:  "{}",
		Amount:         50000,
		IdempotencyKey: "idem-001",
	}

	p, err := testSvc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("failed to initiate payment: %v", err)
	}

	if p.ID == "" {
		t.Fatal("expected non-empty payment ID")
	}
	if p.PaymentStatus != model.StatusAuthorized {
		t.Errorf("expected status AUTHORIZED, got: %s", p.PaymentStatus)
	}
	t.Logf("Step 1: Payment initiated and authorized: %s", p.ID)

	// Verify idempotency
	p2, err := testSvc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("idempotency check failed: %v", err)
	}
	if p2.ID != p.ID {
		t.Errorf("expected same payment ID for same idempotency key, got %s and %s", p.ID, p2.ID)
	}
	t.Log("Idempotency verified")

	// Step 2: Capture Payment
	captured, err := testSvc.CapturePayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to capture payment: %v", err)
	}
	if captured.PaymentStatus != model.StatusCaptured {
		t.Errorf("expected status CAPTURED, got: %s", captured.PaymentStatus)
	}
	t.Logf("Step 2: Payment captured: %s", captured.ID)

	// Step 3: Refund Payment
	refunded, err := testSvc.RefundPayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to refund payment: %v", err)
	}
	if refunded.PaymentStatus != model.StatusRefunded {
		t.Errorf("expected status REFUNDED, got: %s", refunded.PaymentStatus)
	}
	t.Logf("Step 3: Payment refunded: %s", refunded.ID)

	// Verify final state in DB
	final, err := testSvc.GetPayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("failed to get payment from DB: %v", err)
	}
	if final.PaymentStatus != model.StatusRefunded {
		t.Errorf("DB state mismatch: expected REFUNDED, got: %s", final.PaymentStatus)
	}
}

func TestFunctional_CancelPayment(t *testing.T) {
	cleanupTables()
	ctx := context.Background()

	// Initiate
	req := &model.InitiatePaymentRequest{
		OrderID:       "order-func-002",
		UserID:        "user-func-002",
		PaymentMethod: "WALLET",
		PaymentDetail: "{}",
		Amount:        30000,
	}

	p, err := testSvc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}

	// Cancel
	cancelled, err := testSvc.CancelPayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.PaymentStatus != model.StatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", cancelled.PaymentStatus)
	}
	t.Logf("Payment cancelled: %s", cancelled.ID)

	// Verify in DB
	final, err := testSvc.GetPayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if final.PaymentStatus != model.StatusCancelled {
		t.Errorf("DB state mismatch: expected CANCELLED, got: %s", final.PaymentStatus)
	}
}

func TestFunctional_InitiateWithPromo(t *testing.T) {
	cleanupTables()
	ctx := context.Background()

	req := &model.InitiatePaymentRequest{
		OrderID:       "order-func-003",
		UserID:        "user-func-003",
		PaymentMethod: "WALLET",
		PaymentDetail: "{}",
		Amount:        50000,
		PromoCode:     "DISKON10",
	}

	p, err := testSvc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}

	if p.Amount != 50000 {
		t.Errorf("expected amount 50000, got: %.2f", p.Amount)
	}
	if p.FinalAmount != 40000 {
		t.Errorf("expected final amount 40000, got: %.2f", p.FinalAmount)
	}
	t.Logf("Promo applied: Amount=%.0f, FinalAmount=%.0f", p.Amount, p.FinalAmount)
}

func TestFunctional_InvalidCapture(t *testing.T) {
	cleanupTables()
	ctx := context.Background()

	// Initiate
	req := &model.InitiatePaymentRequest{
		OrderID:       "order-func-004",
		UserID:        "user-func-004",
		PaymentMethod: "WALLET",
		PaymentDetail: "{}",
		Amount:        25000,
	}

	p, err := testSvc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("initiate: %v", err)
	}

	// Cancel it first
	_, err = testSvc.CancelPayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}

	// Try to capture cancelled payment
	_, err = testSvc.CapturePayment(ctx, p.ID)
	if err == nil {
		t.Fatal("expected error when capturing cancelled payment")
	}
	t.Logf("Correctly rejected capture on cancelled payment: %v", err)
}
