// Package unit contains unit tests for the ride order service.
// Unit tests do NOT access any database or external service.
// All dependencies are mocked using gomock.
//
// Tests cover the full ride flowchart:
// PENDING → ASSIGNED → PICKING_UP → ON_THE_WAY → COMPLETED
// Plus: user cancel, driver cancel (re-match), wallet lock/unlock, payment capture
package unit

import (
	"context"
	"testing"
	"time"

	"furab-backend/services/ride-order-service/internal/model"
	"furab-backend/services/ride-order-service/internal/repository"
	"furab-backend/services/ride-order-service/internal/service"
	"furab-backend/services/ride-order-service/test/unit/mock"
	"furab-backend/shared/event"

	"go.uber.org/mock/gomock"
)

// --- Helper Functions ---

// newTestService creates a new OrderService with mocked dependencies.
func newTestService(t *testing.T) (service.OrderService, *mock.MockOrderRepository, *mock.MockEventPublisher, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockOrderRepository(ctrl)
	mockPublisher := mock.NewMockEventPublisher(ctrl)
	svc := service.NewOrderService(mockRepo, mockPublisher)
	return svc, mockRepo, mockPublisher, ctrl
}

// validCreateRequest returns a valid CreateRideOrderRequest for testing.
func validCreateRequest() *model.CreateRideOrderRequest {
	return &model.CreateRideOrderRequest{
		UserID: "user-123",
		PickupLocation: model.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
			Address:   "Monas, Jakarta Pusat",
		},
		DropoffLocation: model.Location{
			Latitude:  -6.1751,
			Longitude: 106.8650,
			Address:   "Ancol, Jakarta Utara",
		},
	}
}

// sampleOrder returns a sample RideOrder for testing.
func sampleOrder() *model.RideOrder {
	return &model.RideOrder{
		ID:     "order-abc-123",
		UserID: "user-123",
		PickupLocation: model.Location{
			Latitude:  -6.2088,
			Longitude: 106.8456,
			Address:   "Monas, Jakarta Pusat",
		},
		DropoffLocation: model.Location{
			Latitude:  -6.1751,
			Longitude: 106.8650,
			Address:   "Ancol, Jakarta Utara",
		},
		Status:            model.RideStatusPending,
		PaymentStatus:     model.PaymentStatusNone,
		Fare:              18500,
		Distance:          4.2,
		EstimatedDuration: 9,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

// ========================================
// Test Cases: CreateOrder
// ========================================

// TestCreateOrder_Success tests creating a ride order with valid data.
// Expected: order created with PENDING status, ride.created + wallet.lock events published.
func TestCreateOrder_Success(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := validCreateRequest()

	// Expect repository Create to be called
	mockRepo.EXPECT().
		Create(ctx, gomock.Any()).
		Return(nil)

	// Expect ride.created event
	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRideCreated, gomock.Any()).
		Return(nil)

	// Expect wallet.lock event (saldo locked)
	mockPublisher.EXPECT().
		Publish(ctx, event.TopicWalletLock, gomock.Any()).
		Return(nil)

	order, err := svc.CreateOrder(ctx, req)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if order == nil {
		t.Fatal("expected order, got nil")
	}
	if order.Status != model.RideStatusPending {
		t.Errorf("expected status PENDING, got: %s", order.Status)
	}
	if order.PaymentStatus != model.PaymentStatusNone {
		t.Errorf("expected payment status NONE, got: %s", order.PaymentStatus)
	}
	if order.UserID != req.UserID {
		t.Errorf("expected user ID %s, got: %s", req.UserID, order.UserID)
	}
	if order.Fare <= 0 {
		t.Error("expected fare > 0")
	}
	if order.ID == "" {
		t.Error("expected non-empty order ID")
	}
}

// TestCreateOrder_NilRequest tests creating a ride order with nil request.
func TestCreateOrder_NilRequest(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.CreateOrder(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// TestCreateOrder_InvalidPickup tests creating a ride order with empty pickup address.
func TestCreateOrder_InvalidPickup(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	req := validCreateRequest()
	req.PickupLocation.Address = "" // invalid

	_, err := svc.CreateOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid pickup")
	}
}

// TestCreateOrder_InvalidDropoff tests creating a ride order with empty dropoff address.
func TestCreateOrder_InvalidDropoff(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	req := validCreateRequest()
	req.DropoffLocation.Address = "" // invalid

	_, err := svc.CreateOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid dropoff")
	}
}

// TestCreateOrder_EmptyUserID tests creating a ride order with empty user ID.
func TestCreateOrder_EmptyUserID(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	req := validCreateRequest()
	req.UserID = ""

	_, err := svc.CreateOrder(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

// ========================================
// Test Cases: GetOrder
// ========================================

// TestGetOrder_Success tests retrieving an existing ride order.
func TestGetOrder_Success(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expected := sampleOrder()

	mockRepo.EXPECT().
		GetByID(ctx, expected.ID).
		Return(expected, nil)

	order, err := svc.GetOrder(ctx, expected.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if order.ID != expected.ID {
		t.Errorf("expected order ID %s, got: %s", expected.ID, order.ID)
	}
}

// TestGetOrder_NotFound tests retrieving a non-existent order.
func TestGetOrder_NotFound(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockRepo.EXPECT().
		GetByID(ctx, "non-existent").
		Return(nil, repository.ErrOrderNotFound)

	_, err := svc.GetOrder(ctx, "non-existent")
	if err != service.ErrOrderNotFound {
		t.Fatalf("expected ErrOrderNotFound, got: %v", err)
	}
}

// TestGetOrder_EmptyID tests retrieving an order with empty ID.
func TestGetOrder_EmptyID(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.GetOrder(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

// ========================================
// Test Cases: AssignDriver (PENDING → ASSIGNED)
// ========================================

// TestAssignDriver_Success tests assigning a driver to a PENDING order.
// Expected: status ASSIGNED, payment AUTHORIZED, ride.assigned event published.
func TestAssignDriver_Success(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusPending

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	mockRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(nil)

	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRideAssigned, gomock.Any()).
		Return(nil)

	result, err := svc.AssignDriver(ctx, order.ID, "driver-456")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.RideStatusAssigned {
		t.Errorf("expected status ASSIGNED, got: %s", result.Status)
	}
	if result.DriverID != "driver-456" {
		t.Errorf("expected driver ID driver-456, got: %s", result.DriverID)
	}
	if result.PaymentStatus != model.PaymentStatusAuthorized {
		t.Errorf("expected payment AUTHORIZED, got: %s", result.PaymentStatus)
	}
}

// TestAssignDriver_InvalidStatus tests assigning a driver to a COMPLETED order.
func TestAssignDriver_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusCompleted

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.AssignDriver(ctx, order.ID, "driver-456")
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// TestAssignDriver_AlreadyAssigned tests assigning when driver already assigned.
func TestAssignDriver_AlreadyAssigned(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusPending
	order.DriverID = "existing-driver"

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.AssignDriver(ctx, order.ID, "new-driver")
	if err != service.ErrDriverAlreadyAssigned {
		t.Fatalf("expected ErrDriverAlreadyAssigned, got: %v", err)
	}
}

// ========================================
// Test Cases: PickingUp (ASSIGNED → PICKING_UP)
// ========================================

// TestPickingUp_Success tests transitioning to PICKING_UP status.
// Flowchart: driver is heading to pickup location.
func TestPickingUp_Success(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusAssigned
	order.DriverID = "driver-456"

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	mockRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(nil)

	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRidePickingUp, gomock.Any()).
		Return(nil)

	result, err := svc.PickingUp(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.RideStatusPickingUp {
		t.Errorf("expected status PICKING_UP, got: %s", result.Status)
	}
}

// TestPickingUp_InvalidStatus tests picking up from PENDING (should fail).
func TestPickingUp_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusPending // cannot picking_up from PENDING

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.PickingUp(ctx, order.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// Test Cases: OnTheWay (PICKING_UP → ON_THE_WAY)
// ========================================

// TestOnTheWay_Success tests transitioning to ON_THE_WAY status.
// Flowchart: passenger picked up, ride in progress.
func TestOnTheWay_Success(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusPickingUp
	order.DriverID = "driver-456"

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	mockRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(nil)

	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRideOnTheWay, gomock.Any()).
		Return(nil)

	result, err := svc.OnTheWay(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.RideStatusOnTheWay {
		t.Errorf("expected status ON_THE_WAY, got: %s", result.Status)
	}
}

// TestOnTheWay_InvalidStatus tests on the way from ASSIGNED (should fail, must PICKING_UP first).
func TestOnTheWay_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusAssigned // cannot ON_THE_WAY from ASSIGNED

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.OnTheWay(ctx, order.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// Test Cases: CompleteRide (ON_THE_WAY → COMPLETED)
// ========================================

// TestCompleteRide_Success tests completing an ON_THE_WAY ride.
// Expected: status COMPLETED, payment CAPTURED, ride.completed + payment.captured events.
func TestCompleteRide_Success(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusOnTheWay
	order.DriverID = "driver-456"

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	mockRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(nil)

	// Expect ride.completed event
	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRideCompleted, gomock.Any()).
		Return(nil)

	// Expect payment.captured event
	mockPublisher.EXPECT().
		Publish(ctx, event.TopicPaymentCaptured, gomock.Any()).
		Return(nil)

	result, err := svc.CompleteRide(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.RideStatusCompleted {
		t.Errorf("expected status COMPLETED, got: %s", result.Status)
	}
	if result.PaymentStatus != model.PaymentStatusCaptured {
		t.Errorf("expected payment CAPTURED, got: %s", result.PaymentStatus)
	}
}

// TestCompleteRide_InvalidStatus tests completing from PICKING_UP (should fail).
func TestCompleteRide_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusPickingUp // cannot complete from PICKING_UP

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.CompleteRide(ctx, order.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// TestCompleteRide_FromPending tests completing from PENDING (should fail).
func TestCompleteRide_FromPending(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusPending

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.CompleteRide(ctx, order.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// Test Cases: CancelRide (User Cancel)
// ========================================

// TestCancelRide_FromPending tests user cancelling a PENDING ride.
// Expected: status CANCELLED, payment REFUNDED, ride.cancelled + wallet.unlock events.
func TestCancelRide_FromPending(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusPending

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	mockRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(nil)

	// Expect ride.cancelled event
	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRideCancelled, gomock.Any()).
		Return(nil)

	// Expect wallet.unlock event
	mockPublisher.EXPECT().
		Publish(ctx, event.TopicWalletUnlock, gomock.Any()).
		Return(nil)

	cancelReq := &model.CancelRideRequest{CancelledBy: "user", CancelReason: "changed my mind"}
	result, err := svc.CancelRide(ctx, order.ID, cancelReq)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.RideStatusCancelled {
		t.Errorf("expected status CANCELLED, got: %s", result.Status)
	}
	if result.PaymentStatus != model.PaymentStatusRefunded {
		t.Errorf("expected payment REFUNDED, got: %s", result.PaymentStatus)
	}
	if result.CancelledBy != "user" {
		t.Errorf("expected cancelled_by user, got: %s", result.CancelledBy)
	}
}

// TestCancelRide_FromAssigned tests user cancelling an ASSIGNED ride.
func TestCancelRide_FromAssigned(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusAssigned
	order.DriverID = "driver-456"

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	mockRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(nil)

	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRideCancelled, gomock.Any()).
		Return(nil)

	mockPublisher.EXPECT().
		Publish(ctx, event.TopicWalletUnlock, gomock.Any()).
		Return(nil)

	result, err := svc.CancelRide(ctx, order.ID, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.RideStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", result.Status)
	}
}

// TestCancelRide_FromPickingUp tests user cancelling while driver is heading to pickup.
func TestCancelRide_FromPickingUp(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusPickingUp
	order.DriverID = "driver-456"

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	mockRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(nil)

	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRideCancelled, gomock.Any()).
		Return(nil)

	mockPublisher.EXPECT().
		Publish(ctx, event.TopicWalletUnlock, gomock.Any()).
		Return(nil)

	result, err := svc.CancelRide(ctx, order.ID, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.RideStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", result.Status)
	}
}

// TestCancelRide_AlreadyCompleted tests cancelling a COMPLETED ride (should fail).
func TestCancelRide_AlreadyCompleted(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusCompleted // cannot cancel

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.CancelRide(ctx, order.ID, nil)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// TestCancelRide_OnTheWay tests cancelling while ON_THE_WAY (cannot cancel mid-ride).
func TestCancelRide_OnTheWay(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusOnTheWay // cannot cancel mid-ride

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.CancelRide(ctx, order.ID, nil)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// Test Cases: DriverCancelRide (Re-Match)
// ========================================

// TestDriverCancel_Success tests driver cancellation with re-matching.
// Flowchart: Driver Cancel → Re-Match Driver → back to matching-service
func TestDriverCancel_Success(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusAssigned
	order.DriverID = "driver-456"

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	mockRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(nil)

	// Expect ride.driver_cancelled event → triggers re-matching
	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRideDriverCancelled, gomock.Any()).
		Return(nil)

	result, err := svc.DriverCancelRide(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// Order goes back to PENDING for re-matching
	if result.Status != model.RideStatusPending {
		t.Errorf("expected status PENDING (for re-match), got: %s", result.Status)
	}
	// Driver removed
	if result.DriverID != "" {
		t.Errorf("expected empty driver ID after driver cancel, got: %s", result.DriverID)
	}
	// Wallet stays locked (don't unlock until user cancels or ride completes)
	if result.PaymentStatus != model.PaymentStatusAuthorized {
		t.Errorf("expected payment still AUTHORIZED, got: %s", result.PaymentStatus)
	}
}

// TestDriverCancel_FromPickingUp tests driver cancel during PICKING_UP.
func TestDriverCancel_FromPickingUp(t *testing.T) {
	svc, mockRepo, mockPublisher, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusPickingUp
	order.DriverID = "driver-456"

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	mockRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(nil)

	mockPublisher.EXPECT().
		Publish(ctx, event.TopicRideDriverCancelled, gomock.Any()).
		Return(nil)

	result, err := svc.DriverCancelRide(ctx, order.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.RideStatusPending {
		t.Errorf("expected PENDING for re-match, got: %s", result.Status)
	}
}

// TestDriverCancel_InvalidStatus tests driver cancel from ON_THE_WAY (should fail).
func TestDriverCancel_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusOnTheWay // cannot driver cancel mid-ride
	order.DriverID = "driver-456"

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.DriverCancelRide(ctx, order.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// TestDriverCancel_NoDriver tests driver cancel when no driver assigned.
func TestDriverCancel_NoDriver(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	order := sampleOrder()
	order.Status = model.RideStatusAssigned
	order.DriverID = "" // no driver

	mockRepo.EXPECT().
		GetByID(ctx, order.ID).
		Return(order, nil)

	_, err := svc.DriverCancelRide(ctx, order.ID)
	if err != service.ErrNoDriverAssigned {
		t.Fatalf("expected ErrNoDriverAssigned, got: %v", err)
	}
}

// ========================================
// Test Cases: Full Flow (State Machine)
// ========================================

// TestFullFlow_HappyPath tests the complete ride state machine.
// Validates: PENDING → ASSIGNED → PICKING_UP → ON_THE_WAY → COMPLETED
func TestFullFlow_HappyPath(t *testing.T) {
	// Test that the state machine transitions are correct
	transitions := []struct {
		from model.RideStatus
		to   model.RideStatus
		ok   bool
	}{
		{model.RideStatusPending, model.RideStatusAssigned, true},
		{model.RideStatusAssigned, model.RideStatusPickingUp, true},
		{model.RideStatusPickingUp, model.RideStatusOnTheWay, true},
		{model.RideStatusOnTheWay, model.RideStatusCompleted, true},
		// Invalid transitions
		{model.RideStatusPending, model.RideStatusPickingUp, false},
		{model.RideStatusPending, model.RideStatusOnTheWay, false},
		{model.RideStatusPending, model.RideStatusCompleted, false},
		{model.RideStatusAssigned, model.RideStatusOnTheWay, false},
		{model.RideStatusAssigned, model.RideStatusCompleted, false},
		{model.RideStatusPickingUp, model.RideStatusCompleted, false},
		{model.RideStatusOnTheWay, model.RideStatusCancelled, false},
		{model.RideStatusCompleted, model.RideStatusCancelled, false},
	}

	for _, tc := range transitions {
		result := tc.from.CanTransitionTo(tc.to)
		if result != tc.ok {
			t.Errorf("%s → %s: expected %v, got %v", tc.from, tc.to, tc.ok, result)
		}
	}
}

// ========================================
// Test Cases: GetUserOrders
// ========================================

// TestGetUserOrders_Success tests retrieving orders for a valid user.
func TestGetUserOrders_Success(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expectedOrders := []*model.RideOrder{sampleOrder(), sampleOrder()}

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-123", 10, 0).
		Return(expectedOrders, nil)

	mockRepo.EXPECT().
		CountByUserID(ctx, "user-123").
		Return(2, nil)

	orders, total, err := svc.GetUserOrders(ctx, "user-123", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("expected 2 orders, got: %d", len(orders))
	}
	if total != 2 {
		t.Errorf("expected total 2, got: %d", total)
	}
}

// TestGetUserOrders_Empty tests retrieving orders for a user with no orders.
func TestGetUserOrders_Empty(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockRepo.EXPECT().
		GetByUserID(ctx, "user-empty", 10, 0).
		Return([]*model.RideOrder{}, nil)

	mockRepo.EXPECT().
		CountByUserID(ctx, "user-empty").
		Return(0, nil)

	orders, total, err := svc.GetUserOrders(ctx, "user-empty", 10, 0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("expected 0 orders, got: %d", len(orders))
	}
	if total != 0 {
		t.Errorf("expected total 0, got: %d", total)
	}
}
