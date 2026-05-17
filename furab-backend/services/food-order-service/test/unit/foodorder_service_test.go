// Package unit contains unit tests for the food order service.
// Tests cover the full food flowchart:
// PENDING → CONFIRMED → PREPARING → READY → PICKED_UP → DELIVERING → COMPLETED
// Plus: merchant reject, user cancel, driver cancel (re-match), wallet lock/unlock
package unit

import (
	"context"
	"testing"
	"time"

	"furab-backend/services/food-order-service/internal/model"
	"furab-backend/services/food-order-service/internal/repository"
	"furab-backend/services/food-order-service/internal/service"
	"furab-backend/services/food-order-service/test/unit/mock"
	"furab-backend/shared/event"

	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (service.FoodOrderService, *mock.MockFoodOrderRepository, *mock.MockEventPublisher, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockFoodOrderRepository(ctrl)
	mockPub := mock.NewMockEventPublisher(ctrl)
	svc := service.NewFoodOrderService(mockRepo, mockPub)
	return svc, mockRepo, mockPub, ctrl
}

func validCreateRequest() *model.CreateFoodOrderRequest {
	return &model.CreateFoodOrderRequest{
		UserID:     "user-123",
		MerchantID: "merchant-456",
		Items: []model.OrderItem{
			{ID: "item-1", MenuItemID: "menu-1", Name: "Nasi Goreng", Price: 25000, Quantity: 2},
			{ID: "item-2", MenuItemID: "menu-2", Name: "Es Teh", Price: 5000, Quantity: 1},
		},
		DeliveryAddress: model.Location{Latitude: -6.2088, Longitude: 106.8456, Address: "Rumah User"},
		MerchantAddress: model.Location{Latitude: -6.1751, Longitude: 106.8650, Address: "Restoran ABC"},
	}
}

func sampleOrder() *model.FoodOrder {
	return &model.FoodOrder{
		ID:         "order-food-001",
		UserID:     "user-123",
		MerchantID: "merchant-456",
		Items: []model.OrderItem{
			{ID: "item-1", MenuItemID: "menu-1", Name: "Nasi Goreng", Price: 25000, Quantity: 2},
		},
		Status:        model.FoodStatusPending,
		PaymentStatus: model.PaymentStatusNone,
		SubTotal:      50000,
		DeliveryFee:   8000,
		TotalAmount:   58000,
		DeliveryAddress: model.Location{Latitude: -6.2088, Longitude: 106.8456, Address: "Rumah"},
		MerchantAddress: model.Location{Latitude: -6.1751, Longitude: 106.8650, Address: "Resto"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// ========================================
// CreateOrder
// ========================================

func TestCreateOrder_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()

	mockRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicFoodCreated, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicWalletLock, gomock.Any()).Return(nil)

	order, err := svc.CreateOrder(ctx, validCreateRequest())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if order.Status != model.FoodStatusPending {
		t.Errorf("expected PENDING, got: %s", order.Status)
	}
	if order.TotalAmount <= 0 {
		t.Error("expected total > 0")
	}
}

func TestCreateOrder_NilRequest(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	_, err := svc.CreateOrder(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateOrder_EmptyUserID(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	req := validCreateRequest()
	req.UserID = ""
	_, err := svc.CreateOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateOrder_EmptyItems(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	req := validCreateRequest()
	req.Items = nil
	_, err := svc.CreateOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateOrder_EmptyMerchantID(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	req := validCreateRequest()
	req.MerchantID = ""
	_, err := svc.CreateOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ========================================
// GetOrder
// ========================================

func TestGetOrder_Success(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	expected := sampleOrder()

	mockRepo.EXPECT().GetByID(ctx, expected.ID).Return(expected, nil)

	order, err := svc.GetOrder(ctx, expected.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if order.ID != expected.ID {
		t.Errorf("expected ID %s, got: %s", expected.ID, order.ID)
	}
}

func TestGetOrder_NotFound(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(ctx, "nope").Return(nil, repository.ErrOrderNotFound)
	_, err := svc.GetOrder(ctx, "nope")
	if err != service.ErrOrderNotFound {
		t.Fatalf("expected ErrOrderNotFound, got: %v", err)
	}
}

// ========================================
// MerchantConfirm (PENDING → CONFIRMED)
// ========================================

func TestMerchantConfirm_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusPending

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicFoodConfirmed, gomock.Any()).Return(nil)

	result, err := svc.MerchantConfirm(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.FoodStatusConfirmed {
		t.Errorf("expected CONFIRMED, got: %s", result.Status)
	}
	if result.PaymentStatus != model.PaymentStatusAuthorized {
		t.Errorf("expected AUTHORIZED, got: %s", result.PaymentStatus)
	}
}

func TestMerchantConfirm_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusCompleted

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	_, err := svc.MerchantConfirm(ctx, order.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// MerchantReject (PENDING → CANCELLED)
// ========================================

func TestMerchantReject_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusPending

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicRideCancelled, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicWalletUnlock, gomock.Any()).Return(nil)

	result, err := svc.MerchantReject(ctx, order.ID, "out of stock")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.FoodStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", result.Status)
	}
	if result.CancelledBy != "merchant" {
		t.Errorf("expected cancelled_by merchant, got: %s", result.CancelledBy)
	}
}

func TestMerchantReject_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusDelivering

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	_, err := svc.MerchantReject(ctx, order.ID, "reason")
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// StartPreparing (CONFIRMED → PREPARING)
// ========================================

func TestStartPreparing_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusConfirmed

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicFoodPreparing, gomock.Any()).Return(nil)

	result, err := svc.StartPreparing(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.FoodStatusPreparing {
		t.Errorf("expected PREPARING, got: %s", result.Status)
	}
}

func TestStartPreparing_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusPending

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	_, err := svc.StartPreparing(ctx, order.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// MarkReady (PREPARING → READY)
// ========================================

func TestMarkReady_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusPreparing

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicFoodReady, gomock.Any()).Return(nil)

	result, err := svc.MarkReady(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.FoodStatusReady {
		t.Errorf("expected READY, got: %s", result.Status)
	}
}

// ========================================
// AssignDriver
// ========================================

func TestAssignDriver_Success(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusReady

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

	result, err := svc.AssignDriver(ctx, order.ID, "driver-789")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.DriverID != "driver-789" {
		t.Errorf("expected driver-789, got: %s", result.DriverID)
	}
}

func TestAssignDriver_AlreadyAssigned(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.DriverID = "existing"

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	_, err := svc.AssignDriver(ctx, order.ID, "new-driver")
	if err != service.ErrDriverAlreadyAssigned {
		t.Fatalf("expected ErrDriverAlreadyAssigned, got: %v", err)
	}
}

// ========================================
// PickedUp (READY → PICKED_UP)
// ========================================

func TestPickedUp_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusReady

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicFoodPickedUp, gomock.Any()).Return(nil)

	result, err := svc.PickedUp(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.FoodStatusPickedUp {
		t.Errorf("expected PICKED_UP, got: %s", result.Status)
	}
}

// ========================================
// Delivering (PICKED_UP → DELIVERING)
// ========================================

func TestDelivering_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusPickedUp

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, "food.delivering", gomock.Any()).Return(nil)

	result, err := svc.Delivering(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.FoodStatusDelivering {
		t.Errorf("expected DELIVERING, got: %s", result.Status)
	}
}

// ========================================
// CompleteOrder (DELIVERING → COMPLETED)
// ========================================

func TestCompleteOrder_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusDelivering
	order.DriverID = "driver-789"

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicFoodCompleted, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicPaymentCaptured, gomock.Any()).Return(nil)

	result, err := svc.CompleteOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.FoodStatusCompleted {
		t.Errorf("expected COMPLETED, got: %s", result.Status)
	}
	if result.PaymentStatus != model.PaymentStatusCaptured {
		t.Errorf("expected CAPTURED, got: %s", result.PaymentStatus)
	}
}

func TestCompleteOrder_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusPending

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	_, err := svc.CompleteOrder(ctx, order.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// CancelOrder (user cancel)
// ========================================

func TestCancelOrder_FromPending(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusPending

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicRideCancelled, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicWalletUnlock, gomock.Any()).Return(nil)

	result, err := svc.CancelOrder(ctx, order.ID, &model.CancelFoodOrderRequest{CancelledBy: "user"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.FoodStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", result.Status)
	}
}

func TestCancelOrder_AlreadyCompleted(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusCompleted

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	_, err := svc.CancelOrder(ctx, order.ID, nil)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// DriverCancelOrder (re-match)
// ========================================

func TestDriverCancel_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusReady
	order.DriverID = "driver-789"

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, event.TopicRideDriverCancelled, gomock.Any()).Return(nil)

	result, err := svc.DriverCancelOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.FoodStatusReady {
		t.Errorf("expected READY (re-match), got: %s", result.Status)
	}
	if result.DriverID != "" {
		t.Errorf("expected empty driver, got: %s", result.DriverID)
	}
}

func TestDriverCancel_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusDelivering
	order.DriverID = "driver-789"

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	_, err := svc.DriverCancelOrder(ctx, order.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

func TestDriverCancel_NoDriver(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.FoodStatusReady
	order.DriverID = ""

	mockRepo.EXPECT().GetByID(ctx, order.ID).Return(order, nil)
	_, err := svc.DriverCancelOrder(ctx, order.ID)
	if err != service.ErrNoDriverAssigned {
		t.Fatalf("expected ErrNoDriverAssigned, got: %v", err)
	}
}

// ========================================
// GetUserOrders
// ========================================

func TestGetUserOrders_Success(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()

	mockRepo.EXPECT().GetByUserID(ctx, "user-123", 10, 0).Return([]*model.FoodOrder{sampleOrder()}, nil)
	mockRepo.EXPECT().CountByUserID(ctx, "user-123").Return(1, nil)

	orders, total, err := svc.GetUserOrders(ctx, "user-123", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("expected 1 order, got: %d", len(orders))
	}
	if total != 1 {
		t.Errorf("expected total 1, got: %d", total)
	}
}

// ========================================
// State Machine
// ========================================

func TestFoodStateMachine(t *testing.T) {
	transitions := []struct {
		from model.FoodOrderStatus
		to   model.FoodOrderStatus
		ok   bool
	}{
		{model.FoodStatusPending, model.FoodStatusConfirmed, true},
		{model.FoodStatusConfirmed, model.FoodStatusPreparing, true},
		{model.FoodStatusPreparing, model.FoodStatusReady, true},
		{model.FoodStatusReady, model.FoodStatusPickedUp, true},
		{model.FoodStatusPickedUp, model.FoodStatusDelivering, true},
		{model.FoodStatusDelivering, model.FoodStatusCompleted, true},
		// Cancel paths
		{model.FoodStatusPending, model.FoodStatusCancelled, true},
		{model.FoodStatusConfirmed, model.FoodStatusCancelled, true},
		{model.FoodStatusPreparing, model.FoodStatusCancelled, true},
		// Invalid
		{model.FoodStatusReady, model.FoodStatusCancelled, false},
		{model.FoodStatusDelivering, model.FoodStatusCancelled, false},
		{model.FoodStatusCompleted, model.FoodStatusCancelled, false},
		{model.FoodStatusPending, model.FoodStatusCompleted, false},
	}

	for _, tc := range transitions {
		result := tc.from.CanTransitionTo(tc.to)
		if result != tc.ok {
			t.Errorf("%s → %s: expected %v, got %v", tc.from, tc.to, tc.ok, result)
		}
	}
}
