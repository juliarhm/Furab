//go:build functional
// +build functional

// Package functional contains functional tests for food-order-service.
// Functional tests access a REAL PostgreSQL database.
//
// Run with: go test ./test/functional/... -v -tags=functional
//
// Prerequisites:
//   - Docker Desktop running
//   - Run: docker compose -f deploy/docker/docker-compose.yml up -d postgres
//   - Database "food_order_service" created automatically by init script
package functional

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"furab-backend/services/food-order-service/internal/model"
	"furab-backend/services/food-order-service/internal/repository"
	"furab-backend/services/food-order-service/internal/service"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testDB   *sql.DB
	testRepo repository.FoodOrderRepository
	testSvc  service.FoodOrderService
)

func TestMain(m *testing.M) {
	dbHost := getEnvOrDefault("DB_HOST", "127.0.0.1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "furab")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "furab_secret")
	dbName := getEnvOrDefault("DB_NAME", "food_order_service")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	testDB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer testDB.Close()

	for i := 0; i < 30; i++ {
		err = testDB.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for database... (%d/30)", i+1)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Database not ready: %v", err)
	}
	log.Println("Database connected!")

	if err := setupSchema(); err != nil {
		log.Fatalf("Schema setup failed: %v", err)
	}

	testRepo = repository.NewPostgresFoodOrderRepository(testDB)
	testSvc = service.NewFoodOrderService(testRepo, nil) // nil publisher

	code := m.Run()
	teardownSchema()
	os.Exit(code)
}

func setupSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS food_orders (
			id VARCHAR(36) PRIMARY KEY,
			user_id VARCHAR(36) NOT NULL,
			merchant_id VARCHAR(36) NOT NULL,
			driver_id VARCHAR(36),
			items JSONB NOT NULL DEFAULT '[]',
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			payment_status VARCHAR(20) NOT NULL DEFAULT 'NONE',
			sub_total DOUBLE PRECISION NOT NULL DEFAULT 0,
			delivery_fee DOUBLE PRECISION NOT NULL DEFAULT 0,
			discount DOUBLE PRECISION NOT NULL DEFAULT 0,
			total_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			delivery_lat DOUBLE PRECISION NOT NULL DEFAULT 0,
			delivery_lng DOUBLE PRECISION NOT NULL DEFAULT 0,
			delivery_address TEXT NOT NULL DEFAULT '',
			merchant_lat DOUBLE PRECISION NOT NULL DEFAULT 0,
			merchant_lng DOUBLE PRECISION NOT NULL DEFAULT 0,
			merchant_address TEXT NOT NULL DEFAULT '',
			cancelled_by VARCHAR(20),
			cancel_reason TEXT,
			notes TEXT,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_food_orders_user_id ON food_orders(user_id);
		CREATE INDEX IF NOT EXISTS idx_food_orders_merchant_id ON food_orders(merchant_id);
		CREATE INDEX IF NOT EXISTS idx_food_orders_status ON food_orders(status);
	`
	_, err := testDB.Exec(query)
	return err
}

func teardownSchema() {
	testDB.Exec("DROP TABLE IF EXISTS food_orders")
}

func cleanupOrders() {
	testDB.Exec("DELETE FROM food_orders")
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func validCreateRequest() *model.CreateFoodOrderRequest {
	return &model.CreateFoodOrderRequest{
		UserID:     "func-user-001",
		MerchantID: "merchant-warteg",
		Items: []model.OrderItem{
			{ID: "item-1", MenuItemID: "menu-nasi", Name: "Nasi Goreng", Price: 25000, Quantity: 2},
			{ID: "item-2", MenuItemID: "menu-teh", Name: "Es Teh", Price: 5000, Quantity: 1},
		},
		DeliveryAddress: model.Location{Latitude: -6.2088, Longitude: 106.8456, Address: "Rumah User, Sudirman"},
		MerchantAddress: model.Location{Latitude: -6.1751, Longitude: 106.8650, Address: "Warteg Bahari, Ancol"},
	}
}

// --- Functional Test Cases ---

// TestFunctional_CreateAndGetOrder tests create + get from real DB.
func TestFunctional_CreateAndGetOrder(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	order, err := testSvc.CreateOrder(ctx, validCreateRequest())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if order.Status != model.FoodStatusPending {
		t.Errorf("expected PENDING, got: %s", order.Status)
	}
	if order.TotalAmount <= 0 {
		t.Error("expected total > 0")
	}
	t.Logf("Created: %s (total: Rp %.0f, delivery: Rp %.0f)", order.ID, order.TotalAmount, order.DeliveryFee)

	// Get from DB
	fetched, err := testSvc.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fetched.ID != order.ID {
		t.Errorf("expected ID %s, got: %s", order.ID, fetched.ID)
	}
	if len(fetched.Items) != 2 {
		t.Errorf("expected 2 items from DB, got: %d", len(fetched.Items))
	}
	if fetched.DeliveryAddress.Address != "Rumah User, Sudirman" {
		t.Errorf("delivery address mismatch: %s", fetched.DeliveryAddress.Address)
	}
}

// TestFunctional_FullFoodFlow tests the COMPLETE food delivery lifecycle:
// PENDING → CONFIRMED → PREPARING → READY → assign driver → PICKED_UP → DELIVERING → COMPLETED
func TestFunctional_FullFoodFlow(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	// Step 1: Create
	order, err := testSvc.CreateOrder(ctx, validCreateRequest())
	if err != nil {
		t.Fatalf("step 1 create: %v", err)
	}
	t.Logf("Step 1 - Created: %s (status: %s)", order.ID, order.Status)

	// Step 2: Merchant confirm
	confirmed, err := testSvc.MerchantConfirm(ctx, order.ID)
	if err != nil {
		t.Fatalf("step 2 confirm: %v", err)
	}
	if confirmed.Status != model.FoodStatusConfirmed {
		t.Errorf("expected CONFIRMED, got: %s", confirmed.Status)
	}
	if confirmed.PaymentStatus != model.PaymentStatusAuthorized {
		t.Errorf("expected AUTHORIZED, got: %s", confirmed.PaymentStatus)
	}
	t.Logf("Step 2 - Confirmed (payment: %s)", confirmed.PaymentStatus)

	// Step 3: Start preparing
	preparing, err := testSvc.StartPreparing(ctx, order.ID)
	if err != nil {
		t.Fatalf("step 3 prepare: %v", err)
	}
	if preparing.Status != model.FoodStatusPreparing {
		t.Errorf("expected PREPARING, got: %s", preparing.Status)
	}
	t.Logf("Step 3 - Preparing")

	// Step 4: Mark ready
	ready, err := testSvc.MarkReady(ctx, order.ID)
	if err != nil {
		t.Fatalf("step 4 ready: %v", err)
	}
	if ready.Status != model.FoodStatusReady {
		t.Errorf("expected READY, got: %s", ready.Status)
	}
	t.Logf("Step 4 - Ready for pickup")

	// Step 5: Assign driver
	assigned, err := testSvc.AssignDriver(ctx, order.ID, "driver-food-001")
	if err != nil {
		t.Fatalf("step 5 assign: %v", err)
	}
	if assigned.DriverID != "driver-food-001" {
		t.Errorf("expected driver-food-001, got: %s", assigned.DriverID)
	}
	t.Logf("Step 5 - Driver assigned: %s", assigned.DriverID)

	// Step 6: Picked up
	pickedUp, err := testSvc.PickedUp(ctx, order.ID)
	if err != nil {
		t.Fatalf("step 6 pickup: %v", err)
	}
	if pickedUp.Status != model.FoodStatusPickedUp {
		t.Errorf("expected PICKED_UP, got: %s", pickedUp.Status)
	}
	t.Logf("Step 6 - Picked up by driver")

	// Step 7: Delivering
	delivering, err := testSvc.Delivering(ctx, order.ID)
	if err != nil {
		t.Fatalf("step 7 delivering: %v", err)
	}
	if delivering.Status != model.FoodStatusDelivering {
		t.Errorf("expected DELIVERING, got: %s", delivering.Status)
	}
	t.Logf("Step 7 - Delivering to customer")

	// Step 8: Complete
	completed, err := testSvc.CompleteOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("step 8 complete: %v", err)
	}
	if completed.Status != model.FoodStatusCompleted {
		t.Errorf("expected COMPLETED, got: %s", completed.Status)
	}
	if completed.PaymentStatus != model.PaymentStatusCaptured {
		t.Errorf("expected CAPTURED, got: %s", completed.PaymentStatus)
	}
	t.Logf("Step 8 - Completed! (total: Rp %.0f, payment: %s)", completed.TotalAmount, completed.PaymentStatus)

	// Verify final state from DB
	final, err := testSvc.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if final.Status != model.FoodStatusCompleted {
		t.Errorf("DB: expected COMPLETED, got: %s", final.Status)
	}
}

// TestFunctional_MerchantReject tests merchant rejecting an order.
func TestFunctional_MerchantReject(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	order, err := testSvc.CreateOrder(ctx, validCreateRequest())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rejected, err := testSvc.MerchantReject(ctx, order.ID, "habis bahan")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != model.FoodStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", rejected.Status)
	}
	if rejected.CancelledBy != "merchant" {
		t.Errorf("expected cancelled_by merchant, got: %s", rejected.CancelledBy)
	}
	if rejected.PaymentStatus != model.PaymentStatusRefunded {
		t.Errorf("expected REFUNDED, got: %s", rejected.PaymentStatus)
	}
	t.Logf("Merchant rejected: %s (reason: %s)", rejected.CancelledBy, rejected.CancelReason)

	// Verify in DB
	final, err := testSvc.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if final.CancelledBy != "merchant" {
		t.Errorf("DB: cancelled_by expected merchant, got: %s", final.CancelledBy)
	}
}

// TestFunctional_UserCancel tests user cancelling during preparation.
func TestFunctional_UserCancel(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	order, err := testSvc.CreateOrder(ctx, validCreateRequest())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Confirm → Preparing → Cancel
	testSvc.MerchantConfirm(ctx, order.ID)
	testSvc.StartPreparing(ctx, order.ID)

	cancelled, err := testSvc.CancelOrder(ctx, order.ID, &model.CancelFoodOrderRequest{
		CancelledBy: "user", CancelReason: "terlalu lama",
	})
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Status != model.FoodStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", cancelled.Status)
	}
	if cancelled.CancelledBy != "user" {
		t.Errorf("expected user, got: %s", cancelled.CancelledBy)
	}
	t.Logf("User cancelled from PREPARING: %s", cancelled.CancelReason)
}

// TestFunctional_DriverCancelAndRematch tests driver cancel → back to READY for re-match.
func TestFunctional_DriverCancelAndRematch(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	order, err := testSvc.CreateOrder(ctx, validCreateRequest())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Confirm → Prepare → Ready → Assign driver
	testSvc.MerchantConfirm(ctx, order.ID)
	testSvc.StartPreparing(ctx, order.ID)
	testSvc.MarkReady(ctx, order.ID)
	testSvc.AssignDriver(ctx, order.ID, "driver-cancel-001")

	// Driver cancels
	rematched, err := testSvc.DriverCancelOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("driver cancel: %v", err)
	}
	if rematched.Status != model.FoodStatusReady {
		t.Errorf("expected READY (re-match), got: %s", rematched.Status)
	}
	if rematched.DriverID != "" {
		t.Errorf("expected empty driver, got: %s", rematched.DriverID)
	}
	t.Logf("Driver cancelled → back to READY for re-match")

	// Assign new driver
	reassigned, err := testSvc.AssignDriver(ctx, order.ID, "driver-new-002")
	if err != nil {
		t.Fatalf("re-assign: %v", err)
	}
	if reassigned.DriverID != "driver-new-002" {
		t.Errorf("expected driver-new-002, got: %s", reassigned.DriverID)
	}
	t.Logf("Re-matched with: %s", reassigned.DriverID)
}

// TestFunctional_InvalidTransitions tests rejected invalid status changes.
func TestFunctional_InvalidTransitions(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()

	order, err := testSvc.CreateOrder(ctx, validCreateRequest())
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Cannot complete PENDING order directly
	_, err = testSvc.CompleteOrder(ctx, order.ID)
	if err == nil {
		t.Fatal("expected error: PENDING → COMPLETED")
	}
	t.Logf("Rejected PENDING → COMPLETED: %v", err)

	// Cannot start preparing before confirm
	_, err = testSvc.StartPreparing(ctx, order.ID)
	if err == nil {
		t.Fatal("expected error: PENDING → PREPARING")
	}
	t.Logf("Rejected PENDING → PREPARING: %v", err)

	// Cannot pick up before ready
	_, err = testSvc.PickedUp(ctx, order.ID)
	if err == nil {
		t.Fatal("expected error: PENDING → PICKED_UP")
	}
	t.Logf("Rejected PENDING → PICKED_UP: %v", err)
}

// TestFunctional_GetUserOrders tests pagination from real DB.
func TestFunctional_GetUserOrders(t *testing.T) {
	cleanupOrders()
	ctx := context.Background()
	userID := "func-user-pagination"

	// Create 3 orders
	for i := 0; i < 3; i++ {
		req := validCreateRequest()
		req.UserID = userID
		req.Notes = fmt.Sprintf("Order %d", i+1)
		_, err := testSvc.CreateOrder(ctx, req)
		if err != nil {
			t.Fatalf("create %d: %v", i+1, err)
		}
	}

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

	// Pagination: limit 2
	page1, _, err := testSvc.GetUserOrders(ctx, userID, 2, 0)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 in page 1, got: %d", len(page1))
	}
}
