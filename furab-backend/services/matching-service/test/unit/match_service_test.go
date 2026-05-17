// Package unit contains unit tests for the matching service.
// Tests cover: SEARCHING → OFFERED → MATCHED, reject+retry, cancel, max attempts fail.
// Matching service digunakan BERSAMA oleh ride-order-service dan food-order-service.
package unit

import (
	"context"
	"testing"
	"time"

	"furab-backend/services/matching-service/internal/model"
	"furab-backend/services/matching-service/internal/repository"
	"furab-backend/services/matching-service/internal/service"
	"furab-backend/services/matching-service/test/unit/mock"

	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (service.MatchService, *mock.MockMatchRepository, *mock.MockEventPublisher, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockMatchRepository(ctrl)
	mockPub := mock.NewMockEventPublisher(ctrl)
	svc := service.NewMatchService(mockRepo, mockPub)
	return svc, mockRepo, mockPub, ctrl
}

func validFindRequest(orderType string) *model.FindDriverRequest {
	return &model.FindDriverRequest{
		OrderID:   "order-001",
		OrderType: orderType,
		UserID:    "user-123",
		PickupLocation: model.Location{
			Latitude: -6.2088, Longitude: 106.8456, Address: "Monas",
		},
		DropoffLocation: model.Location{
			Latitude: -6.1751, Longitude: 106.8650, Address: "Ancol",
		},
	}
}

func sampleMatch() *model.MatchRequest {
	return &model.MatchRequest{
		ID:          "match-001",
		OrderID:     "order-001",
		OrderType:   "ride",
		UserID:      "user-123",
		Status:      model.MatchStatusSearching,
		AttemptCount: 0,
		MaxAttempts: 3,
		Radius:      5.0,
		PickupLocation: model.Location{Latitude: -6.2088, Longitude: 106.8456, Address: "Monas"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

// ========================================
// FindDriver
// ========================================

func TestFindDriver_RideSuccess(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()

	mockRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, "match.searching", gomock.Any()).Return(nil)

	match, err := svc.FindDriver(ctx, validFindRequest("ride"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if match.Status != model.MatchStatusSearching {
		t.Errorf("expected SEARCHING, got: %s", match.Status)
	}
	if match.OrderType != "ride" {
		t.Errorf("expected ride, got: %s", match.OrderType)
	}
}

func TestFindDriver_FoodSuccess(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()

	mockRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, "match.searching", gomock.Any()).Return(nil)

	match, err := svc.FindDriver(ctx, validFindRequest("food"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if match.OrderType != "food" {
		t.Errorf("expected food, got: %s", match.OrderType)
	}
}

func TestFindDriver_NilRequest(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	_, err := svc.FindDriver(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFindDriver_InvalidOrderType(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	req := validFindRequest("invalid")
	_, err := svc.FindDriver(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid order type")
	}
}

func TestFindDriver_EmptyOrderID(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	req := validFindRequest("ride")
	req.OrderID = ""
	_, err := svc.FindDriver(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty order ID")
	}
}

// ========================================
// OfferToDriver
// ========================================

func TestOfferToDriver_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, "match.offered", gomock.Any()).Return(nil)

	result, err := svc.OfferToDriver(ctx, match.ID, "driver-456")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.MatchStatusOffered {
		t.Errorf("expected OFFERED, got: %s", result.Status)
	}
	if result.DriverID != "driver-456" {
		t.Errorf("expected driver-456, got: %s", result.DriverID)
	}
	if result.AttemptCount != 1 {
		t.Errorf("expected attempt 1, got: %d", result.AttemptCount)
	}
}

func TestOfferToDriver_AlreadyOffered(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.DriverID = "existing-driver"

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	_, err := svc.OfferToDriver(ctx, match.ID, "new-driver")
	if err != service.ErrAlreadyOffered {
		t.Fatalf("expected ErrAlreadyOffered, got: %v", err)
	}
}

func TestOfferToDriver_EmptyDriverID(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	_, err := svc.OfferToDriver(context.Background(), "match-001", "")
	if err == nil {
		t.Fatal("expected error for empty driver ID")
	}
}

// ========================================
// DriverAccept
// ========================================

func TestDriverAccept_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.Status = model.MatchStatusOffered
	match.DriverID = "driver-456"

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, "match.accepted", gomock.Any()).Return(nil)

	result, err := svc.DriverAccept(ctx, match.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.MatchStatusMatched {
		t.Errorf("expected MATCHED, got: %s", result.Status)
	}
}

func TestDriverAccept_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.Status = model.MatchStatusSearching

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	_, err := svc.DriverAccept(ctx, match.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// DriverReject (retry)
// ========================================

func TestDriverReject_RetrySuccess(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.Status = model.MatchStatusOffered
	match.DriverID = "driver-456"
	match.AttemptCount = 1

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

	result, err := svc.DriverReject(ctx, match.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.MatchStatusSearching {
		t.Errorf("expected SEARCHING (retry), got: %s", result.Status)
	}
	if result.DriverID != "" {
		t.Errorf("expected empty driver after reject, got: %s", result.DriverID)
	}
}

func TestDriverReject_MaxAttempts(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.Status = model.MatchStatusOffered
	match.DriverID = "driver-456"
	match.AttemptCount = 3 // max reached
	match.MaxAttempts = 3

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, "match.failed", gomock.Any()).Return(nil)

	result, err := svc.DriverReject(ctx, match.ID)
	if err != service.ErrMaxAttempts {
		t.Fatalf("expected ErrMaxAttempts, got: %v", err)
	}
	if result.Status != model.MatchStatusFailed {
		t.Errorf("expected FAILED, got: %s", result.Status)
	}
}

func TestDriverReject_InvalidStatus(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.Status = model.MatchStatusSearching

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	_, err := svc.DriverReject(ctx, match.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// CancelMatch
// ========================================

func TestCancelMatch_Success(t *testing.T) {
	svc, mockRepo, mockPub, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.Status = model.MatchStatusSearching

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	mockPub.EXPECT().Publish(ctx, "match.cancelled", gomock.Any()).Return(nil)

	result, err := svc.CancelMatch(ctx, match.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Status != model.MatchStatusCancelled {
		t.Errorf("expected CANCELLED, got: %s", result.Status)
	}
}

func TestCancelMatch_AlreadyMatched(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.Status = model.MatchStatusMatched

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	_, err := svc.CancelMatch(ctx, match.ID)
	if err != service.ErrInvalidTransition {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// ========================================
// GetMatchStatus
// ========================================

func TestGetMatchStatus_Success(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)

	result, err := svc.GetMatchStatus(ctx, match.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.ID != match.ID {
		t.Errorf("expected ID %s, got: %s", match.ID, result.ID)
	}
}

func TestGetMatchStatus_NotFound(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()

	mockRepo.EXPECT().GetByID(ctx, "nope").Return(nil, repository.ErrMatchNotFound)
	_, err := svc.GetMatchStatus(ctx, "nope")
	if err != service.ErrMatchNotFound {
		t.Fatalf("expected ErrMatchNotFound, got: %v", err)
	}
}

// ========================================
// RetryMatch
// ========================================

func TestRetryMatch_Success(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.Status = model.MatchStatusSearching
	match.AttemptCount = 1

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	mockRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

	result, err := svc.RetryMatch(ctx, match.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Radius != 7.0 { // 5.0 + 2.0
		t.Errorf("expected radius 7.0, got: %v", result.Radius)
	}
}

func TestRetryMatch_MaxAttempts(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()
	ctx := context.Background()
	match := sampleMatch()
	match.Status = model.MatchStatusSearching
	match.AttemptCount = 3
	match.MaxAttempts = 3

	mockRepo.EXPECT().GetByID(ctx, match.ID).Return(match, nil)
	_, err := svc.RetryMatch(ctx, match.ID)
	if err != service.ErrMaxAttempts {
		t.Fatalf("expected ErrMaxAttempts, got: %v", err)
	}
}

// ========================================
// State Machine
// ========================================

func TestMatchStateMachine(t *testing.T) {
	transitions := []struct {
		from model.MatchStatus
		to   model.MatchStatus
		ok   bool
	}{
		{model.MatchStatusSearching, model.MatchStatusOffered, true},
		{model.MatchStatusSearching, model.MatchStatusFailed, true},
		{model.MatchStatusSearching, model.MatchStatusCancelled, true},
		{model.MatchStatusOffered, model.MatchStatusMatched, true},
		{model.MatchStatusOffered, model.MatchStatusSearching, true},
		{model.MatchStatusOffered, model.MatchStatusCancelled, true},
		// Invalid
		{model.MatchStatusMatched, model.MatchStatusCancelled, false},
		{model.MatchStatusFailed, model.MatchStatusSearching, false},
		{model.MatchStatusCancelled, model.MatchStatusSearching, false},
		{model.MatchStatusSearching, model.MatchStatusMatched, false},
	}

	for _, tc := range transitions {
		result := tc.from.CanTransitionTo(tc.to)
		if result != tc.ok {
			t.Errorf("%s → %s: expected %v, got %v", tc.from, tc.to, tc.ok, result)
		}
	}
}
