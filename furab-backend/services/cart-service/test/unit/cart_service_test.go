// Package unit contains unit tests for the cart service.
// Unit tests do NOT access any database or external service.
// All dependencies are mocked using gomock.
package unit

import (
	"context"
	"testing"
	"time"

	"furab-backend/services/cart-service/internal/model"
	"furab-backend/services/cart-service/internal/repository"
	"furab-backend/services/cart-service/internal/service"
	"furab-backend/services/cart-service/test/unit/mock"

	"go.uber.org/mock/gomock"
)

// --- Helper Functions ---

func newTestService(t *testing.T) (service.CartService, *mock.MockCartRepository, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockCartRepository(ctrl)
	svc := service.NewCartService(mockRepo)
	return svc, mockRepo, ctrl
}

func sampleCart() *model.Cart {
	return &model.Cart{
		ID:         "cart-001",
		UserID:     "user-123",
		MerchantID: "merchant-456",
		Items: []model.CartItem{
			{
				ID:         "item-001",
				MenuItemID: "menu-001",
				MerchantID: "merchant-456",
				Name:       "Nasi Goreng",
				Price:      25000,
				Quantity:   2,
			},
			{
				ID:         "item-002",
				MenuItemID: "menu-002",
				MerchantID: "merchant-456",
				Name:       "Es Teh Manis",
				Price:      5000,
				Quantity:   1,
			},
		},
		TotalPrice: 55000,
		ItemCount:  3,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
}

func validAddItemRequest() *model.AddItemRequest {
	return &model.AddItemRequest{
		MenuItemID: "menu-003",
		MerchantID: "merchant-456",
		Name:       "Mie Goreng",
		Price:      22000,
		Quantity:   1,
	}
}

// ========================================
// Test Cases: GetCart
// ========================================

func TestGetCart_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expected := sampleCart()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(expected, nil)

	cart, err := svc.GetCart(ctx, "user-123")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cart.UserID != "user-123" {
		t.Errorf("expected userID user-123, got: %s", cart.UserID)
	}
	if len(cart.Items) != 2 {
		t.Errorf("expected 2 items, got: %d", len(cart.Items))
	}
}

func TestGetCart_EmptyCart(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-new").
		Return(nil, repository.ErrCartNotFound)

	cart, err := svc.GetCart(ctx, "user-new")
	if err != nil {
		t.Fatalf("expected no error for new user, got: %v", err)
	}
	if len(cart.Items) != 0 {
		t.Errorf("expected empty cart, got: %d items", len(cart.Items))
	}
}

func TestGetCart_EmptyUserID(t *testing.T) {
	svc, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.GetCart(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

// ========================================
// Test Cases: AddItem
// ========================================

func TestAddItem_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cart := sampleCart()
	req := validAddItemRequest()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(cart, nil)

	mockRepo.EXPECT().
		Save(ctx, gomock.Any()).
		Return(nil)

	result, err := svc.AddItem(ctx, "user-123", req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items after add, got: %d", len(result.Items))
	}
}

func TestAddItem_NewCart(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := validAddItemRequest()

	// No existing cart
	mockRepo.EXPECT().
		GetByUserID(ctx, "user-new").
		Return(nil, repository.ErrCartNotFound)

	mockRepo.EXPECT().
		Save(ctx, gomock.Any()).
		Return(nil)

	result, err := svc.AddItem(ctx, "user-new", req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item in new cart, got: %d", len(result.Items))
	}
	if result.MerchantID != "merchant-456" {
		t.Errorf("expected merchant-456, got: %s", result.MerchantID)
	}
}

func TestAddItem_DuplicateMenuItem(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cart := sampleCart()

	// Add the same menu item that already exists
	req := &model.AddItemRequest{
		MenuItemID: "menu-001", // already in cart (Nasi Goreng, qty 2)
		MerchantID: "merchant-456",
		Name:       "Nasi Goreng",
		Price:      25000,
		Quantity:   1,
	}

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(cart, nil)

	mockRepo.EXPECT().
		Save(ctx, gomock.Any()).
		Return(nil)

	result, err := svc.AddItem(ctx, "user-123", req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// Should still be 2 items (not 3), but quantity of Nasi Goreng should be 3
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items (duplicate merged), got: %d", len(result.Items))
	}
	if result.Items[0].Quantity != 3 {
		t.Errorf("expected quantity 3 after merge, got: %d", result.Items[0].Quantity)
	}
}

func TestAddItem_DifferentMerchant(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cart := sampleCart() // merchant-456

	req := &model.AddItemRequest{
		MenuItemID: "menu-099",
		MerchantID: "merchant-999", // different!
		Name:       "Pizza",
		Price:      50000,
		Quantity:   1,
	}

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(cart, nil)

	_, err := svc.AddItem(ctx, "user-123", req)
	if err != service.ErrDiffMerchant {
		t.Fatalf("expected ErrDiffMerchant, got: %v", err)
	}
}

func TestAddItem_EmptyMenuItemID(t *testing.T) {
	svc, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	req := &model.AddItemRequest{
		MenuItemID: "",
		MerchantID: "merchant-456",
		Name:       "Test",
		Price:      10000,
		Quantity:   1,
	}

	_, err := svc.AddItem(context.Background(), "user-123", req)
	if err == nil {
		t.Fatal("expected error for empty menu_item_id")
	}
}

func TestAddItem_InvalidQuantity(t *testing.T) {
	svc, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	req := &model.AddItemRequest{
		MenuItemID: "menu-001",
		MerchantID: "merchant-456",
		Name:       "Test",
		Price:      10000,
		Quantity:   0, // invalid
	}

	_, err := svc.AddItem(context.Background(), "user-123", req)
	if err == nil {
		t.Fatal("expected error for invalid quantity")
	}
}

func TestAddItem_NilRequest(t *testing.T) {
	svc, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.AddItem(context.Background(), "user-123", nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ========================================
// Test Cases: UpdateItemQuantity
// ========================================

func TestUpdateQuantity_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cart := sampleCart()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(cart, nil)

	mockRepo.EXPECT().
		Save(ctx, gomock.Any()).
		Return(nil)

	result, err := svc.UpdateItemQuantity(ctx, "user-123", "item-001", 5)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Items[0].Quantity != 5 {
		t.Errorf("expected quantity 5, got: %d", result.Items[0].Quantity)
	}
}

func TestUpdateQuantity_ItemNotFound(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cart := sampleCart()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(cart, nil)

	_, err := svc.UpdateItemQuantity(ctx, "user-123", "nonexistent", 5)
	if err != service.ErrItemNotFound {
		t.Fatalf("expected ErrItemNotFound, got: %v", err)
	}
}

func TestUpdateQuantity_ZeroRemovesItem(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cart := sampleCart() // 2 items

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(cart, nil)

	mockRepo.EXPECT().
		Save(ctx, gomock.Any()).
		Return(nil)

	result, err := svc.UpdateItemQuantity(ctx, "user-123", "item-001", 0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item after removing, got: %d", len(result.Items))
	}
}

// ========================================
// Test Cases: RemoveItem
// ========================================

func TestRemoveItem_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cart := sampleCart()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(cart, nil)

	mockRepo.EXPECT().
		Save(ctx, gomock.Any()).
		Return(nil)

	result, err := svc.RemoveItem(ctx, "user-123", "item-001")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item after remove, got: %d", len(result.Items))
	}
}

func TestRemoveItem_NotFound(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cart := sampleCart()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(cart, nil)

	_, err := svc.RemoveItem(ctx, "user-123", "nonexistent")
	if err != service.ErrItemNotFound {
		t.Fatalf("expected ErrItemNotFound, got: %v", err)
	}
}

// ========================================
// Test Cases: ClearCart
// ========================================

func TestClearCart_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockRepo.EXPECT().
		Delete(ctx, "user-123").
		Return(nil)

	err := svc.ClearCart(ctx, "user-123")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestClearCart_EmptyUserID(t *testing.T) {
	svc, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	err := svc.ClearCart(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

// ========================================
// Test Cases: GetCartTotal
// ========================================

func TestGetCartTotal_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cart := sampleCart()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123").
		Return(cart, nil)

	total, err := svc.GetCartTotal(ctx, "user-123")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// 25000*2 + 5000*1 = 55000
	if total != 55000 {
		t.Errorf("expected total 55000, got: %v", total)
	}
}

func TestGetCartTotal_EmptyCart(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-empty").
		Return(nil, repository.ErrCartNotFound)

	total, err := svc.GetCartTotal(ctx, "user-empty")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got: %v", total)
	}
}
