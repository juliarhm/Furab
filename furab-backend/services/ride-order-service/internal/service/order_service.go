// Package service implements the business logic for ride order management.
package service

import (
	"context"
	"errors"
	"math"
	"time"

	"furab-backend/services/ride-order-service/internal/model"
	"furab-backend/services/ride-order-service/internal/repository"
	"furab-backend/shared/event"

	"github.com/google/uuid"
)

// Common service errors.
var (
	ErrInvalidRequest        = errors.New("invalid request")
	ErrOrderNotFound         = errors.New("order not found")
	ErrInvalidTransition     = errors.New("invalid status transition")
	ErrDriverAlreadyAssigned = errors.New("driver already assigned")
	ErrNoDriverAssigned      = errors.New("no driver assigned to this order")
)

// OrderService defines the interface for ride order business logic.
// This interface is used for dependency injection in handlers and can be mocked in tests.
type OrderService interface {
	// CreateOrder creates a new ride order, estimating fare and publishing ride.created event.
	// Also publishes wallet.lock event to authorize (lock) user's wallet balance.
	CreateOrder(ctx context.Context, req *model.CreateRideOrderRequest) (*model.RideOrder, error)

	// GetOrder retrieves a ride order by its ID.
	GetOrder(ctx context.Context, id string) (*model.RideOrder, error)

	// AssignDriver assigns a driver to a pending ride order and publishes ride.assigned event.
	// Flow: PENDING → ASSIGNED (driver accepts the offer from matching-service)
	AssignDriver(ctx context.Context, orderID, driverID string) (*model.RideOrder, error)

	// PickingUp sets status to PICKING_UP when driver is heading to pickup location.
	// Flow: ASSIGNED → PICKING_UP
	PickingUp(ctx context.Context, orderID string) (*model.RideOrder, error)

	// OnTheWay sets status to ON_THE_WAY when passenger is picked up and ride starts.
	// Flow: PICKING_UP → ON_THE_WAY
	OnTheWay(ctx context.Context, orderID string) (*model.RideOrder, error)

	// CompleteRide transitions an ON_THE_WAY ride to completed status.
	// Also publishes payment.capture and ride.completed events.
	// Flow: ON_THE_WAY → COMPLETED
	CompleteRide(ctx context.Context, orderID string) (*model.RideOrder, error)

	// CancelRide cancels a ride order (by user).
	// Publishes ride.cancelled and wallet.unlock events.
	CancelRide(ctx context.Context, orderID string, req *model.CancelRideRequest) (*model.RideOrder, error)

	// DriverCancelRide handles driver cancellation - resets order to PENDING for re-matching.
	// Publishes ride.driver_cancelled event so matching-service can find a new driver.
	DriverCancelRide(ctx context.Context, orderID string) (*model.RideOrder, error)

	// GetUserOrders retrieves all ride orders for a specific user with pagination.
	GetUserOrders(ctx context.Context, userID string, limit, offset int) ([]*model.RideOrder, int, error)
}

// orderServiceImpl is the concrete implementation of OrderService.
type orderServiceImpl struct {
	repo      repository.OrderRepository
	publisher event.Publisher
}

// NewOrderService creates a new OrderService with the given dependencies.
func NewOrderService(repo repository.OrderRepository, publisher event.Publisher) OrderService {
	return &orderServiceImpl{
		repo:      repo,
		publisher: publisher,
	}
}

// CreateOrder creates a new ride order.
// Step in flowchart: User konfirmasi order → Ride Order Service → Payment Service (wallet lock)
func (s *orderServiceImpl) CreateOrder(ctx context.Context, req *model.CreateRideOrderRequest) (*model.RideOrder, error) {
	// Validate request
	if req == nil {
		return nil, ErrInvalidRequest
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Calculate estimated fare and distance
	distance := calculateDistance(
		req.PickupLocation.Latitude, req.PickupLocation.Longitude,
		req.DropoffLocation.Latitude, req.DropoffLocation.Longitude,
	)
	fare := estimateFare(distance)
	duration := estimateDuration(distance)

	// Create order
	now := time.Now().UTC()
	order := &model.RideOrder{
		ID:                uuid.New().String(),
		UserID:            req.UserID,
		PickupLocation:    req.PickupLocation,
		DropoffLocation:   req.DropoffLocation,
		Status:            model.RideStatusPending,
		PaymentStatus:     model.PaymentStatusNone,
		Fare:              fare,
		Distance:          distance,
		EstimatedDuration: duration,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Save to database
	if err := s.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	// Publish ride.created event → triggers matching-service to find driver
	evt, err := event.NewEvent(event.TopicRideCreated, "ride-order-service", model.RideCreatedEvent{
		OrderID:         order.ID,
		UserID:          order.UserID,
		PickupLocation:  order.PickupLocation,
		DropoffLocation: order.DropoffLocation,
		EstimatedFare:   order.Fare,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicRideCreated, evt)
	}

	// Publish wallet.lock event → payment-service locks user's wallet balance
	walletEvt, err := event.NewEvent(event.TopicWalletLock, "ride-order-service", map[string]interface{}{
		"order_id": order.ID,
		"user_id":  order.UserID,
		"amount":   order.Fare,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicWalletLock, walletEvt)
	}

	return order, nil
}

// GetOrder retrieves a ride order by its ID.
func (s *orderServiceImpl) GetOrder(ctx context.Context, id string) (*model.RideOrder, error) {
	if id == "" {
		return nil, ErrInvalidRequest
	}

	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	return order, nil
}

// AssignDriver assigns a driver to a pending ride order.
// Flowchart: Driver Accept → status ASSIGNED
func (s *orderServiceImpl) AssignDriver(ctx context.Context, orderID, driverID string) (*model.RideOrder, error) {
	if orderID == "" || driverID == "" {
		return nil, ErrInvalidRequest
	}

	// Get current order
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	// Validate status transition: only PENDING -> ASSIGNED
	if !order.Status.CanTransitionTo(model.RideStatusAssigned) {
		return nil, ErrInvalidTransition
	}

	// Check if already assigned
	if order.DriverID != "" {
		return nil, ErrDriverAlreadyAssigned
	}

	// Update order
	order.DriverID = driverID
	order.Status = model.RideStatusAssigned
	order.PaymentStatus = model.PaymentStatusAuthorized // wallet locked
	order.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	// Publish ride.assigned event
	evt, err := event.NewEvent(event.TopicRideAssigned, "ride-order-service", model.RideAssignedEvent{
		OrderID:  order.ID,
		DriverID: driverID,
		UserID:   order.UserID,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicRideAssigned, evt)
	}

	return order, nil
}

// PickingUp sets status to PICKING_UP when driver heads to pickup.
// Flowchart: ASSIGNED → PICKING_UP (driver menuju lokasi jemput)
func (s *orderServiceImpl) PickingUp(ctx context.Context, orderID string) (*model.RideOrder, error) {
	if orderID == "" {
		return nil, ErrInvalidRequest
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	// Validate transition: ASSIGNED → PICKING_UP
	if !order.Status.CanTransitionTo(model.RideStatusPickingUp) {
		return nil, ErrInvalidTransition
	}

	order.Status = model.RideStatusPickingUp
	order.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	// Publish ride.picking_up event → location-service starts tracking
	evt, err := event.NewEvent(event.TopicRidePickingUp, "ride-order-service", model.RidePickingUpEvent{
		OrderID:  order.ID,
		DriverID: order.DriverID,
		UserID:   order.UserID,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicRidePickingUp, evt)
	}

	return order, nil
}

// OnTheWay sets status to ON_THE_WAY when passenger is picked up.
// Flowchart: PICKING_UP → ON_THE_WAY (penumpang sudah dijemput, perjalanan dimulai)
func (s *orderServiceImpl) OnTheWay(ctx context.Context, orderID string) (*model.RideOrder, error) {
	if orderID == "" {
		return nil, ErrInvalidRequest
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	// Validate transition: PICKING_UP → ON_THE_WAY
	if !order.Status.CanTransitionTo(model.RideStatusOnTheWay) {
		return nil, ErrInvalidTransition
	}

	order.Status = model.RideStatusOnTheWay
	order.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	// Publish ride.on_the_way event → location-service continues tracking
	evt, err := event.NewEvent(event.TopicRideOnTheWay, "ride-order-service", model.RideOnTheWayEvent{
		OrderID:  order.ID,
		DriverID: order.DriverID,
		UserID:   order.UserID,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicRideOnTheWay, evt)
	}

	return order, nil
}

// CompleteRide transitions an ON_THE_WAY ride to completed status.
// Flowchart: ON_THE_WAY → COMPLETED → Payment Capture → Wallet Deduct → Settlement
func (s *orderServiceImpl) CompleteRide(ctx context.Context, orderID string) (*model.RideOrder, error) {
	if orderID == "" {
		return nil, ErrInvalidRequest
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	// Validate transition: only ON_THE_WAY -> COMPLETED
	if !order.Status.CanTransitionTo(model.RideStatusCompleted) {
		return nil, ErrInvalidTransition
	}

	order.Status = model.RideStatusCompleted
	order.PaymentStatus = model.PaymentStatusCaptured
	order.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	// Publish ride.completed event → triggers rating-service, review-service, audit-log
	evt, err := event.NewEvent(event.TopicRideCompleted, "ride-order-service", model.RideCompletedEvent{
		OrderID:  order.ID,
		DriverID: order.DriverID,
		UserID:   order.UserID,
		Fare:     order.Fare,
		Distance: order.Distance,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicRideCompleted, evt)
	}

	// Publish payment.captured event → payment-service captures payment
	captureEvt, err := event.NewEvent(event.TopicPaymentCaptured, "ride-order-service", map[string]interface{}{
		"order_id":  order.ID,
		"user_id":   order.UserID,
		"driver_id": order.DriverID,
		"amount":    order.Fare,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicPaymentCaptured, captureEvt)
	}

	return order, nil
}

// CancelRide cancels a ride order (user cancel).
// Flowchart: User Cancel → Cancel Order → Wallet Unlock
func (s *orderServiceImpl) CancelRide(ctx context.Context, orderID string, req *model.CancelRideRequest) (*model.RideOrder, error) {
	if orderID == "" {
		return nil, ErrInvalidRequest
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	// Validate transition: PENDING/ASSIGNED/PICKING_UP → CANCELLED
	if !order.Status.CanTransitionTo(model.RideStatusCancelled) {
		return nil, ErrInvalidTransition
	}

	order.Status = model.RideStatusCancelled
	order.PaymentStatus = model.PaymentStatusRefunded
	if req != nil {
		order.CancelledBy = req.CancelledBy
		order.CancelReason = req.CancelReason
	} else {
		order.CancelledBy = "user"
	}
	order.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	// Publish ride.cancelled event
	evt, err := event.NewEvent(event.TopicRideCancelled, "ride-order-service", model.RideCancelledEvent{
		OrderID:     order.ID,
		UserID:      order.UserID,
		DriverID:    order.DriverID,
		CancelledBy: order.CancelledBy,
		Reason:      order.CancelReason,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicRideCancelled, evt)
	}

	// Publish wallet.unlock event → refund locked balance
	unlockEvt, err := event.NewEvent(event.TopicWalletUnlock, "ride-order-service", map[string]interface{}{
		"order_id": order.ID,
		"user_id":  order.UserID,
		"amount":   order.Fare,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicWalletUnlock, unlockEvt)
	}

	return order, nil
}

// DriverCancelRide handles driver cancellation with re-matching.
// Flowchart: Driver Cancel → Re-Match Driver → back to Matching Service
func (s *orderServiceImpl) DriverCancelRide(ctx context.Context, orderID string) (*model.RideOrder, error) {
	if orderID == "" {
		return nil, ErrInvalidRequest
	}

	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}

	// Driver can only cancel if ASSIGNED or PICKING_UP
	if order.Status != model.RideStatusAssigned && order.Status != model.RideStatusPickingUp {
		return nil, ErrInvalidTransition
	}

	if order.DriverID == "" {
		return nil, ErrNoDriverAssigned
	}

	// Save previous driver for the event
	previousDriverID := order.DriverID

	// Reset to PENDING for re-matching (remove driver, keep wallet locked)
	order.Status = model.RideStatusPending
	order.DriverID = ""
	order.PaymentStatus = model.PaymentStatusAuthorized // keep wallet locked
	order.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	// Publish ride.driver_cancelled → matching-service will find new driver
	evt, err := event.NewEvent(event.TopicRideDriverCancelled, "ride-order-service", model.RideDriverCancelledEvent{
		OrderID:          order.ID,
		UserID:           order.UserID,
		PreviousDriverID: previousDriverID,
		PickupLocation:   order.PickupLocation,
		DropoffLocation:  order.DropoffLocation,
		EstimatedFare:    order.Fare,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicRideDriverCancelled, evt)
	}

	return order, nil
}

// GetUserOrders retrieves all ride orders for a user with pagination.
func (s *orderServiceImpl) GetUserOrders(ctx context.Context, userID string, limit, offset int) ([]*model.RideOrder, int, error) {
	if userID == "" {
		return nil, 0, ErrInvalidRequest
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	orders, err := s.repo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// --- Helper Functions ---

// calculateDistance calculates the distance between two coordinates using the Haversine formula.
// Returns distance in kilometers.
func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// degreesToRadians converts degrees to radians.
func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// estimateFare calculates the estimated fare based on distance.
// Base fare: Rp 8,000 + Rp 2,500/km
func estimateFare(distanceKm float64) float64 {
	baseFare := 8000.0
	perKm := 2500.0
	return baseFare + (distanceKm * perKm)
}

// estimateDuration estimates ride duration in minutes based on distance.
// Assumes average speed of 30 km/h in city traffic.
func estimateDuration(distanceKm float64) int {
	avgSpeedKmH := 30.0
	minutes := (distanceKm / avgSpeedKmH) * 60
	return int(math.Ceil(minutes))
}
