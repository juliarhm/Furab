// Package service implements the business logic for matching-service.
package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/matching-service/internal/model"
	"furab-backend/services/matching-service/internal/repository"
	"furab-backend/shared/event"

	"github.com/google/uuid"
)

var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrMatchNotFound     = errors.New("match not found")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrMaxAttempts       = errors.New("maximum match attempts reached")
	ErrNoDriver          = errors.New("no driver assigned")
	ErrAlreadyOffered    = errors.New("already offered to a driver")
)

const (
	DefaultMaxAttempts = 3
	DefaultRadius      = 5.0 // km
)

// MatchService defines the interface for matching business logic.
type MatchService interface {
	FindDriver(ctx context.Context, req *model.FindDriverRequest) (*model.MatchRequest, error)
	OfferToDriver(ctx context.Context, matchID, driverID string) (*model.MatchRequest, error)
	DriverAccept(ctx context.Context, matchID string) (*model.MatchRequest, error)
	DriverReject(ctx context.Context, matchID string) (*model.MatchRequest, error)
	CancelMatch(ctx context.Context, matchID string) (*model.MatchRequest, error)
	GetMatchStatus(ctx context.Context, matchID string) (*model.MatchRequest, error)
	RetryMatch(ctx context.Context, matchID string) (*model.MatchRequest, error)
}

type matchServiceImpl struct {
	repo      repository.MatchRepository
	publisher event.Publisher
}

func NewMatchService(repo repository.MatchRepository, publisher event.Publisher) MatchService {
	return &matchServiceImpl{repo: repo, publisher: publisher}
}

// FindDriver initiates a new match request.
// Flowchart: ride.created / food.ready → Matching Service → Location Service → Offer Driver
func (s *matchServiceImpl) FindDriver(ctx context.Context, req *model.FindDriverRequest) (*model.MatchRequest, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	match := &model.MatchRequest{
		ID:              uuid.New().String(),
		OrderID:         req.OrderID,
		OrderType:       req.OrderType,
		UserID:          req.UserID,
		PickupLocation:  req.PickupLocation,
		DropoffLocation: req.DropoffLocation,
		Status:          model.MatchStatusSearching,
		AttemptCount:    0,
		MaxAttempts:     DefaultMaxAttempts,
		Radius:          DefaultRadius,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, match); err != nil {
		return nil, err
	}

	evt, err := event.NewEvent("match.searching", "matching-service", model.MatchSearchingEvent{
		MatchID: match.ID, OrderID: match.OrderID, OrderType: match.OrderType,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, "match.searching", evt)
	}

	return match, nil
}

// OfferToDriver offers the match to a specific driver.
// Flowchart: Location Service finds driver → Offer Driver
func (s *matchServiceImpl) OfferToDriver(ctx context.Context, matchID, driverID string) (*model.MatchRequest, error) {
	if matchID == "" || driverID == "" {
		return nil, ErrInvalidRequest
	}

	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, repository.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	if !match.Status.CanTransitionTo(model.MatchStatusOffered) {
		return nil, ErrInvalidTransition
	}

	if match.DriverID != "" {
		return nil, ErrAlreadyOffered
	}

	match.Status = model.MatchStatusOffered
	match.DriverID = driverID
	match.AttemptCount++
	match.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, match); err != nil {
		return nil, err
	}

	evt, err := event.NewEvent("match.offered", "matching-service", model.MatchOfferedEvent{
		MatchID: match.ID, OrderID: match.OrderID, DriverID: driverID,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, "match.offered", evt)
	}

	return match, nil
}

// DriverAccept handles driver accepting the match.
// Flowchart: Driver Accept → rides/{orderID}/assign or foods/{orderID}/assign
func (s *matchServiceImpl) DriverAccept(ctx context.Context, matchID string) (*model.MatchRequest, error) {
	if matchID == "" {
		return nil, ErrInvalidRequest
	}

	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, repository.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	if !match.Status.CanTransitionTo(model.MatchStatusMatched) {
		return nil, ErrInvalidTransition
	}

	match.Status = model.MatchStatusMatched
	match.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, match); err != nil {
		return nil, err
	}

	// Publish match.accepted → triggers ride-order-service or food-order-service to assign driver
	evt, err := event.NewEvent("match.accepted", "matching-service", model.MatchAcceptedEvent{
		MatchID: match.ID, OrderID: match.OrderID, OrderType: match.OrderType,
		DriverID: match.DriverID, UserID: match.UserID,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, "match.accepted", evt)
	}

	return match, nil
}

// DriverReject handles driver rejecting → go back to SEARCHING for retry.
// Flowchart: No Driver / Reject → Retry Matching → back to Matching Service
func (s *matchServiceImpl) DriverReject(ctx context.Context, matchID string) (*model.MatchRequest, error) {
	if matchID == "" {
		return nil, ErrInvalidRequest
	}

	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, repository.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	if match.Status != model.MatchStatusOffered {
		return nil, ErrInvalidTransition
	}

	// Check if max attempts reached
	if match.AttemptCount >= match.MaxAttempts {
		match.Status = model.MatchStatusFailed
		match.DriverID = ""
		match.UpdatedAt = time.Now().UTC()

		if err := s.repo.Update(ctx, match); err != nil {
			return nil, err
		}

		evt, err := event.NewEvent("match.failed", "matching-service", model.MatchFailedEvent{
			MatchID: match.ID, OrderID: match.OrderID,
			AttemptCount: match.AttemptCount, Reason: "max attempts reached",
		})
		if err == nil && s.publisher != nil {
			_ = s.publisher.Publish(ctx, "match.failed", evt)
		}

		return match, ErrMaxAttempts
	}

	// Go back to SEARCHING
	match.Status = model.MatchStatusSearching
	match.DriverID = ""
	match.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, match); err != nil {
		return nil, err
	}

	return match, nil
}

// CancelMatch cancels the match (user cancelled).
// Flowchart: Timeout → Cancel Order + Unlock Wallet
func (s *matchServiceImpl) CancelMatch(ctx context.Context, matchID string) (*model.MatchRequest, error) {
	if matchID == "" {
		return nil, ErrInvalidRequest
	}

	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, repository.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	if !match.Status.CanTransitionTo(model.MatchStatusCancelled) {
		return nil, ErrInvalidTransition
	}

	match.Status = model.MatchStatusCancelled
	match.DriverID = ""
	match.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, match); err != nil {
		return nil, err
	}

	evt, err := event.NewEvent("match.cancelled", "matching-service", map[string]interface{}{
		"match_id": match.ID, "order_id": match.OrderID, "order_type": match.OrderType,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, "match.cancelled", evt)
	}

	return match, nil
}

// GetMatchStatus retrieves match request by ID.
func (s *matchServiceImpl) GetMatchStatus(ctx context.Context, matchID string) (*model.MatchRequest, error) {
	if matchID == "" {
		return nil, ErrInvalidRequest
	}

	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, repository.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return match, nil
}

// RetryMatch starts a new search attempt.
func (s *matchServiceImpl) RetryMatch(ctx context.Context, matchID string) (*model.MatchRequest, error) {
	if matchID == "" {
		return nil, ErrInvalidRequest
	}

	match, err := s.repo.GetByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, repository.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	if match.Status != model.MatchStatusSearching {
		return nil, ErrInvalidTransition
	}

	if match.AttemptCount >= match.MaxAttempts {
		return nil, ErrMaxAttempts
	}

	// Expand radius on retry
	match.Radius += 2.0
	match.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, match); err != nil {
		return nil, err
	}

	return match, nil
}
