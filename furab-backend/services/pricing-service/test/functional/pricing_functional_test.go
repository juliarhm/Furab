//go:build functional
// +build functional

// Package functional contains functional tests for the pricing service.
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

	"furab-backend/services/pricing-service/internal/model"
	"furab-backend/services/pricing-service/internal/repository"
	"furab-backend/services/pricing-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.PriceRepository
	testSvc  service.PriceService
)

// TestMain sets up the test database connection, creates schema, runs tests, and cleans up.
func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "pricing_service")

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
	testRepo = repository.NewPostgresPriceRepository(testDB)
	testSvc = service.NewPriceService(testRepo, &mockOrderClient{}, &mockLocationClient{})

	// Run tests
	code := m.Run()

	// Cleanup
	teardownSchema()
	os.Exit(code)
}

// setupSchema creates the pricing_rules table and seeds initial data.
func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS pricing_rules (
			rule_id VARCHAR(36) PRIMARY KEY,
			type VARCHAR(50) NOT NULL,
			value DOUBLE PRECISION NOT NULL,
			description TEXT
		);

		DELETE FROM pricing_rules;

		INSERT INTO pricing_rules (rule_id, type, value, description) VALUES
		('delivery-per-km', 'delivery', 5000, 'Delivery fee per kilometer'),
		('service-percent', 'service', 0.05, 'Service fee percentage');
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS pricing_rules")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Mocks for External Clients ---

type mockOrderClient struct{}

func (m *mockOrderClient) GetOrderItems(ctx context.Context, orderID string) ([]model.OrderItem, error) {
	if orderID == "non-existent" {
		return nil, fmt.Errorf("order not found")
	}
	return []model.OrderItem{
		{ProductID: "p1", Quantity: 2, UnitPrice: 10000}, // 20000
		{ProductID: "p2", Quantity: 1, UnitPrice: 5000},  // 5000
	}, nil // Total item price: 25000
}

type mockLocationClient struct{}

func (m *mockLocationClient) GetDeliveryDistance(ctx context.Context, orderID string) (float64, error) {
	return 2.5, nil // Distance: 2.5 km
}

// --- Functional Test Cases ---

func TestFunctional_CalculatePrice(t *testing.T) {
	ctx := context.Background()
	orderID := "test-order-001"

	// CalculatePrice calls repo, orderClient, and locationClient
	res, err := testSvc.CalculatePrice(ctx, orderID)
	if err != nil {
		t.Fatalf("failed to calculate price: %v", err)
	}

	// Expectations:
	// itemPrice = (2 * 10000) + (1 * 5000) = 25000
	// deliveryFee = 2.5 km * 5000 = 12500
	// serviceFee = 25000 * 0.05 = 1250
	// totalAmount = 25000 + 12500 + 1250 = 38750

	if res.ItemPrice != 25000 {
		t.Errorf("expected item price 25000, got %.2f", res.ItemPrice)
	}
	if res.DeliveryFee != 12500 {
		t.Errorf("expected delivery fee 12500, got %.2f", res.DeliveryFee)
	}
	if res.ServiceFee != 1250 {
		t.Errorf("expected service fee 1250, got %.2f", res.ServiceFee)
	}
	if res.TotalAmount != 38750 {
		t.Errorf("expected total amount 38750, got %.2f", res.TotalAmount)
	}

	t.Logf("Price calculated successfully for order %s: Total = %.2f", res.OrderID, res.TotalAmount)
}

func TestFunctional_CalculatePrice_OrderNotFound(t *testing.T) {
	ctx := context.Background()
	orderID := "non-existent"

	_, err := testSvc.CalculatePrice(ctx, orderID)
	if err == nil {
		t.Fatal("expected error for non-existent order, got nil")
	}

	t.Logf("Correctly received error for non-existent order: %v", err)
}

func TestFunctional_GetRulesFromDB(t *testing.T) {
	ctx := context.Background()

	rules, err := testRepo.GetPricingRules(ctx)
	if err != nil {
		t.Fatalf("failed to get rules from DB: %v", err)
	}

	if len(rules) < 2 {
		t.Errorf("expected at least 2 rules, got %d", len(rules))
	}

	t.Logf("Found %d pricing rules in database", len(rules))
}
