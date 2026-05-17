//go:build functional
// +build functional

// Package functional contains functional tests for cart-service.
// Functional tests access a REAL PostgreSQL database.
//
// Run with: go test ./test/functional/... -v -tags=functional
//
// Prerequisites:
//   - Docker Desktop running
//   - Run: docker compose -f deploy/docker/docker-compose.yml up -d postgres
//   - Database "cart_service" created automatically by init script
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/cart-service/internal/model"
	"furab-backend/services/cart-service/internal/repository"
	"furab-backend/services/cart-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.CartRepository
	testSvc  service.CartService
)

// TestMain sets up the test database connection, creates schema, runs tests, and cleans up.
func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "cart_service")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	testDB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
	defer testDB.Close()

	// Wait for DB to be ready
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

	if err := setupSchema(); err != nil {
		log.Fatalf("Failed to setup schema: %v", err)
	}

	testRepo = repository.NewPostgresCartRepository(testDB)
	testSvc = service.NewCartService(testRepo)

	code := m.Run()

	teardownSchema()
	os.Exit(code)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS carts (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL UNIQUE,
			merchant_id VARCHAR(36),
			items JSONB NOT NULL DEFAULT '[]',
			total_price DOUBLE PRECISION NOT NULL DEFAULT 0,
			item_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_carts_user_id ON carts(user_id);
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS carts")
}

func cleanupCarts() {
	testDB.Exec("DELETE FROM carts")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// --- Functional Test Cases ---

// TestFunctional_AddItemAndGetCart tests adding items and retrieving from real DB.
func TestFunctional_AddItemAndGetCart(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()
	userID := "func-user-001"

	// Add first item → creates new cart
	req := &model.AddItemRequest{
		MenuItemID: "menu-nasi-goreng",
		MerchantID: "merchant-warteg",
		Name:       "Nasi Goreng Spesial",
		Price:      25000,
		Quantity:   2,
		Notes:      "pedas level 3",
	}

	cart, err := testSvc.AddItem(ctx, userID, req)
	if err != nil {
		t.Fatalf("add item: %v", err)
	}
	if len(cart.Items) != 1 {
		t.Errorf("expected 1 item, got: %d", len(cart.Items))
	}
	t.Logf("Added item: %s (qty: %d, total: Rp %.0f)", cart.Items[0].Name, cart.Items[0].Quantity, cart.TotalPrice)

	// Get cart from DB
	fetched, err := testSvc.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("get cart: %v", err)
	}
	if len(fetched.Items) != 1 {
		t.Errorf("expected 1 item from DB, got: %d", len(fetched.Items))
	}
	if fetched.Items[0].Name != "Nasi Goreng Spesial" {
		t.Errorf("expected Nasi Goreng Spesial, got: %s", fetched.Items[0].Name)
	}
	if fetched.TotalPrice != 50000 { // 25000 * 2
		t.Errorf("expected total 50000, got: %.0f", fetched.TotalPrice)
	}
	if fetched.MerchantID != "merchant-warteg" {
		t.Errorf("expected merchant-warteg, got: %s", fetched.MerchantID)
	}
}

// TestFunctional_AddMultipleItems tests adding multiple items to the cart.
func TestFunctional_AddMultipleItems(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()
	userID := "func-user-002"

	items := []*model.AddItemRequest{
		{MenuItemID: "menu-001", MerchantID: "merchant-A", Name: "Nasi Goreng", Price: 25000, Quantity: 1},
		{MenuItemID: "menu-002", MerchantID: "merchant-A", Name: "Es Teh Manis", Price: 5000, Quantity: 2},
		{MenuItemID: "menu-003", MerchantID: "merchant-A", Name: "Kerupuk", Price: 3000, Quantity: 3},
	}

	var cart *model.Cart
	var err error
	for _, req := range items {
		cart, err = testSvc.AddItem(ctx, userID, req)
		if err != nil {
			t.Fatalf("add item %s: %v", req.Name, err)
		}
	}

	if len(cart.Items) != 3 {
		t.Errorf("expected 3 items, got: %d", len(cart.Items))
	}

	// Total: 25000*1 + 5000*2 + 3000*3 = 44000
	expectedTotal := 44000.0
	if cart.TotalPrice != expectedTotal {
		t.Errorf("expected total %.0f, got: %.0f", expectedTotal, cart.TotalPrice)
	}
	t.Logf("3 items added, total: Rp %.0f", cart.TotalPrice)

	// Verify from DB
	fetched, err := testSvc.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("get cart: %v", err)
	}
	if len(fetched.Items) != 3 {
		t.Errorf("DB: expected 3 items, got: %d", len(fetched.Items))
	}
	if fetched.TotalPrice != expectedTotal {
		t.Errorf("DB: expected total %.0f, got: %.0f", expectedTotal, fetched.TotalPrice)
	}
}

// TestFunctional_DuplicateItemMerge tests that adding same menu item increments quantity.
func TestFunctional_DuplicateItemMerge(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()
	userID := "func-user-003"

	req := &model.AddItemRequest{
		MenuItemID: "menu-ayam", MerchantID: "merchant-X",
		Name: "Ayam Goreng", Price: 20000, Quantity: 1,
	}

	// Add 1 Ayam Goreng
	_, err := testSvc.AddItem(ctx, userID, req)
	if err != nil {
		t.Fatalf("first add: %v", err)
	}

	// Add 2 more Ayam Goreng (same menu ID)
	req.Quantity = 2
	cart, err := testSvc.AddItem(ctx, userID, req)
	if err != nil {
		t.Fatalf("second add: %v", err)
	}

	// Should be 1 item with qty 3, not 2 items
	if len(cart.Items) != 1 {
		t.Errorf("expected 1 item (merged), got: %d", len(cart.Items))
	}
	if cart.Items[0].Quantity != 3 {
		t.Errorf("expected qty 3, got: %d", cart.Items[0].Quantity)
	}
	// Total: 20000 * 3 = 60000
	if cart.TotalPrice != 60000 {
		t.Errorf("expected total 60000, got: %.0f", cart.TotalPrice)
	}
	t.Logf("Merged: %s qty=%d, total=Rp %.0f", cart.Items[0].Name, cart.Items[0].Quantity, cart.TotalPrice)
}

// TestFunctional_DifferentMerchantRejected tests that adding item from different merchant fails.
func TestFunctional_DifferentMerchantRejected(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()
	userID := "func-user-004"

	// Add item from merchant A
	_, err := testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-001", MerchantID: "merchant-A",
		Name: "Item A", Price: 10000, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("add from merchant A: %v", err)
	}

	// Try to add item from merchant B → should fail
	_, err = testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-999", MerchantID: "merchant-B",
		Name: "Item B", Price: 15000, Quantity: 1,
	})
	if err == nil {
		t.Fatal("expected error when adding from different merchant")
	}
	t.Logf("Correctly rejected different merchant: %v", err)
}

// TestFunctional_UpdateQuantity tests updating item quantity in DB.
func TestFunctional_UpdateQuantity(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()
	userID := "func-user-005"

	// Add item
	cart, err := testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-sate", MerchantID: "merchant-Z",
		Name: "Sate Ayam", Price: 30000, Quantity: 1,
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	itemID := cart.Items[0].ID

	// Update quantity to 5
	updated, err := testSvc.UpdateItemQuantity(ctx, userID, itemID, 5)
	if err != nil {
		t.Fatalf("update qty: %v", err)
	}
	if updated.Items[0].Quantity != 5 {
		t.Errorf("expected qty 5, got: %d", updated.Items[0].Quantity)
	}
	// Total: 30000 * 5 = 150000
	if updated.TotalPrice != 150000 {
		t.Errorf("expected total 150000, got: %.0f", updated.TotalPrice)
	}
	t.Logf("Updated qty to 5, total: Rp %.0f", updated.TotalPrice)

	// Verify from DB
	fetched, err := testSvc.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if fetched.Items[0].Quantity != 5 {
		t.Errorf("DB: expected qty 5, got: %d", fetched.Items[0].Quantity)
	}
}

// TestFunctional_UpdateQuantityZeroRemoves tests that setting qty=0 removes the item.
func TestFunctional_UpdateQuantityZeroRemoves(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()
	userID := "func-user-006"

	// Add 2 items
	cart, _ := testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-A", MerchantID: "merchant-Z", Name: "Item A", Price: 10000, Quantity: 1,
	})
	cart, _ = testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-B", MerchantID: "merchant-Z", Name: "Item B", Price: 20000, Quantity: 1,
	})
	if len(cart.Items) != 2 {
		t.Fatalf("expected 2 items, got: %d", len(cart.Items))
	}

	// Set item A qty to 0 → removes it
	itemA_ID := cart.Items[0].ID
	updated, err := testSvc.UpdateItemQuantity(ctx, userID, itemA_ID, 0)
	if err != nil {
		t.Fatalf("update qty 0: %v", err)
	}
	if len(updated.Items) != 1 {
		t.Errorf("expected 1 item after remove, got: %d", len(updated.Items))
	}
	t.Logf("Removed item via qty=0, remaining: %d items", len(updated.Items))
}

// TestFunctional_RemoveItem tests removing an item from cart in DB.
func TestFunctional_RemoveItem(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()
	userID := "func-user-007"

	// Add 2 items
	cart, _ := testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-X", MerchantID: "merchant-Z", Name: "Item X", Price: 15000, Quantity: 2,
	})
	cart, _ = testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-Y", MerchantID: "merchant-Z", Name: "Item Y", Price: 8000, Quantity: 1,
	})

	// Remove Item X
	itemX_ID := cart.Items[0].ID
	result, err := testSvc.RemoveItem(ctx, userID, itemX_ID)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got: %d", len(result.Items))
	}
	// Only Item Y left: 8000 * 1 = 8000
	if result.TotalPrice != 8000 {
		t.Errorf("expected total 8000, got: %.0f", result.TotalPrice)
	}
	t.Logf("Removed item, remaining total: Rp %.0f", result.TotalPrice)
}

// TestFunctional_ClearCart tests clearing all items from DB.
func TestFunctional_ClearCart(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()
	userID := "func-user-008"

	// Add items
	testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-001", MerchantID: "merchant-A", Name: "Item 1", Price: 10000, Quantity: 1,
	})
	testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-002", MerchantID: "merchant-A", Name: "Item 2", Price: 20000, Quantity: 1,
	})

	// Clear
	err := testSvc.ClearCart(ctx, userID)
	if err != nil {
		t.Fatalf("clear: %v", err)
	}

	// Get should return empty cart
	cart, err := testSvc.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("get after clear: %v", err)
	}
	if len(cart.Items) != 0 {
		t.Errorf("expected empty cart, got: %d items", len(cart.Items))
	}
	t.Logf("Cart cleared successfully")
}

// TestFunctional_GetCartTotal tests total calculation from real DB.
func TestFunctional_GetCartTotal(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()
	userID := "func-user-009"

	testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-A", MerchantID: "merchant-Z", Name: "Nasi", Price: 15000, Quantity: 2,
	})
	testSvc.AddItem(ctx, userID, &model.AddItemRequest{
		MenuItemID: "menu-B", MerchantID: "merchant-Z", Name: "Ayam", Price: 25000, Quantity: 1,
	})

	total, err := testSvc.GetCartTotal(ctx, userID)
	if err != nil {
		t.Fatalf("get total: %v", err)
	}

	// 15000*2 + 25000*1 = 55000
	expected := 55000.0
	if total != expected {
		t.Errorf("expected total %.0f, got: %.0f", expected, total)
	}
	t.Logf("Cart total: Rp %.0f", total)
}

// TestFunctional_GetCartEmptyUser tests getting cart for new user (no cart in DB).
func TestFunctional_GetCartEmptyUser(t *testing.T) {
	cleanupCarts()
	ctx := context.Background()

	cart, err := testSvc.GetCart(ctx, "user-never-existed")
	if err != nil {
		t.Fatalf("expected no error for new user, got: %v", err)
	}
	if len(cart.Items) != 0 {
		t.Errorf("expected empty cart, got: %d items", len(cart.Items))
	}
	t.Logf("New user gets empty cart (0 items)")
}
