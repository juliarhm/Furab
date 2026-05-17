// Package service implements the business logic for food-order-service.
package service

import (
	"context"
	"errors"
	"math"
	"time"

	"furab-backend/services/food-order-service/internal/model"
	"furab-backend/services/food-order-service/internal/repository"
	"furab-backend/shared/event"

	"github.com/google/uuid"
)

var (
	ErrInvalidRequest        = errors.New("invalid request")
	ErrOrderNotFound         = errors.New("order not found")
	ErrInvalidTransition     = errors.New("invalid status transition")
	ErrDriverAlreadyAssigned = errors.New("driver already assigned")
	ErrNoDriverAssigned      = errors.New("no driver assigned")
)

// FoodOrderService defines the interface for food order business logic.
type FoodOrderService interface {
	CreateOrder(ctx context.Context, req *model.CreateFoodOrderRequest) (*model.FoodOrder, error)
	GetOrder(ctx context.Context, id string) (*model.FoodOrder, error)
	MerchantConfirm(ctx context.Context, orderID string) (*model.FoodOrder, error)
	MerchantReject(ctx context.Context, orderID, reason string) (*model.FoodOrder, error)
	StartPreparing(ctx context.Context, orderID string) (*model.FoodOrder, error)
	MarkReady(ctx context.Context, orderID string) (*model.FoodOrder, error)
	AssignDriver(ctx context.Context, orderID, driverID string) (*model.FoodOrder, error)
	PickedUp(ctx context.Context, orderID string) (*model.FoodOrder, error)
	Delivering(ctx context.Context, orderID string) (*model.FoodOrder, error)
	CompleteOrder(ctx context.Context, orderID string) (*model.FoodOrder, error)
	CancelOrder(ctx context.Context, orderID string, req *model.CancelFoodOrderRequest) (*model.FoodOrder, error)
	DriverCancelOrder(ctx context.Context, orderID string) (*model.FoodOrder, error)
	GetUserOrders(ctx context.Context, userID string, limit, offset int) ([]*model.FoodOrder, int, error)
}

type foodOrderServiceImpl struct {
	repo      repository.FoodOrderRepository
	publisher event.Publisher
}

func NewFoodOrderService(repo repository.FoodOrderRepository, publisher event.Publisher) FoodOrderService {
	return &foodOrderServiceImpl{repo: repo, publisher: publisher}
}

func (s *foodOrderServiceImpl) CreateOrder(ctx context.Context, req *model.CreateFoodOrderRequest) (*model.FoodOrder, error) {
	if req == nil {
		return nil, ErrInvalidRequest
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	order := &model.FoodOrder{
		ID:              uuid.New().String(),
		UserID:          req.UserID,
		MerchantID:      req.MerchantID,
		Items:           req.Items,
		Status:          model.FoodStatusPending,
		PaymentStatus:   model.PaymentStatusNone,
		DeliveryAddress: req.DeliveryAddress,
		MerchantAddress: req.MerchantAddress,
		Notes:           req.Notes,
		DeliveryFee:     calculateDeliveryFee(req.DeliveryAddress, req.MerchantAddress),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	order.CalculateTotal()

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, err
	}

	evt, err := event.NewEvent(event.TopicFoodCreated, "food-order-service", model.FoodCreatedEvent{
		OrderID: order.ID, UserID: order.UserID, MerchantID: order.MerchantID, TotalAmount: order.TotalAmount,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicFoodCreated, evt)
	}

	walletEvt, err := event.NewEvent(event.TopicWalletLock, "food-order-service", map[string]interface{}{
		"order_id": order.ID, "user_id": order.UserID, "amount": order.TotalAmount,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicWalletLock, walletEvt)
	}

	return order, nil
}

func (s *foodOrderServiceImpl) GetOrder(ctx context.Context, id string) (*model.FoodOrder, error) {
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

// MerchantConfirm: PENDING → CONFIRMED
func (s *foodOrderServiceImpl) MerchantConfirm(ctx context.Context, orderID string) (*model.FoodOrder, error) {
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
	if !order.Status.CanTransitionTo(model.FoodStatusConfirmed) {
		return nil, ErrInvalidTransition
	}

	order.Status = model.FoodStatusConfirmed
	order.PaymentStatus = model.PaymentStatusAuthorized
	order.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	evt, err := event.NewEvent(event.TopicFoodConfirmed, "food-order-service", model.FoodConfirmedEvent{
		OrderID: order.ID, MerchantID: order.MerchantID,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicFoodConfirmed, evt)
	}

	return order, nil
}

// MerchantReject: PENDING → CANCELLED
func (s *foodOrderServiceImpl) MerchantReject(ctx context.Context, orderID, reason string) (*model.FoodOrder, error) {
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
	if !order.Status.CanTransitionTo(model.FoodStatusCancelled) {
		return nil, ErrInvalidTransition
	}

	order.Status = model.FoodStatusCancelled
	order.PaymentStatus = model.PaymentStatusRefunded
	order.CancelledBy = "merchant"
	order.CancelReason = reason
	order.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	s.publishCancel(ctx, order)
	s.publishWalletUnlock(ctx, order)

	return order, nil
}

// StartPreparing: CONFIRMED → PREPARING
func (s *foodOrderServiceImpl) StartPreparing(ctx context.Context, orderID string) (*model.FoodOrder, error) {
	return s.transitionStatus(ctx, orderID, model.FoodStatusPreparing, event.TopicFoodPreparing)
}

// MarkReady: PREPARING → READY
func (s *foodOrderServiceImpl) MarkReady(ctx context.Context, orderID string) (*model.FoodOrder, error) {
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
	if !order.Status.CanTransitionTo(model.FoodStatusReady) {
		return nil, ErrInvalidTransition
	}

	order.Status = model.FoodStatusReady
	order.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	evt, err := event.NewEvent(event.TopicFoodReady, "food-order-service", model.FoodReadyEvent{
		OrderID: order.ID, MerchantID: order.MerchantID,
		MerchantAddress: order.MerchantAddress, DeliveryAddress: order.DeliveryAddress,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicFoodReady, evt)
	}

	return order, nil
}

// AssignDriver: READY → PICKED_UP needs driver first
func (s *foodOrderServiceImpl) AssignDriver(ctx context.Context, orderID, driverID string) (*model.FoodOrder, error) {
	if orderID == "" || driverID == "" {
		return nil, ErrInvalidRequest
	}
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	if order.DriverID != "" {
		return nil, ErrDriverAlreadyAssigned
	}

	order.DriverID = driverID
	order.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

// PickedUp: READY → PICKED_UP
func (s *foodOrderServiceImpl) PickedUp(ctx context.Context, orderID string) (*model.FoodOrder, error) {
	return s.transitionStatus(ctx, orderID, model.FoodStatusPickedUp, event.TopicFoodPickedUp)
}

// Delivering: PICKED_UP → DELIVERING
func (s *foodOrderServiceImpl) Delivering(ctx context.Context, orderID string) (*model.FoodOrder, error) {
	return s.transitionStatus(ctx, orderID, model.FoodStatusDelivering, "food.delivering")
}

// CompleteOrder: DELIVERING → COMPLETED
func (s *foodOrderServiceImpl) CompleteOrder(ctx context.Context, orderID string) (*model.FoodOrder, error) {
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
	if !order.Status.CanTransitionTo(model.FoodStatusCompleted) {
		return nil, ErrInvalidTransition
	}

	order.Status = model.FoodStatusCompleted
	order.PaymentStatus = model.PaymentStatusCaptured
	order.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	evt, err := event.NewEvent(event.TopicFoodCompleted, "food-order-service", model.FoodCompletedEvent{
		OrderID: order.ID, UserID: order.UserID, DriverID: order.DriverID,
		MerchantID: order.MerchantID, TotalAmount: order.TotalAmount,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicFoodCompleted, evt)
	}

	captureEvt, err := event.NewEvent(event.TopicPaymentCaptured, "food-order-service", map[string]interface{}{
		"order_id": order.ID, "user_id": order.UserID, "amount": order.TotalAmount,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicPaymentCaptured, captureEvt)
	}

	return order, nil
}

// CancelOrder: user/system cancel
func (s *foodOrderServiceImpl) CancelOrder(ctx context.Context, orderID string, req *model.CancelFoodOrderRequest) (*model.FoodOrder, error) {
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
	if !order.Status.CanTransitionTo(model.FoodStatusCancelled) {
		return nil, ErrInvalidTransition
	}

	order.Status = model.FoodStatusCancelled
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

	s.publishCancel(ctx, order)
	s.publishWalletUnlock(ctx, order)

	return order, nil
}

// DriverCancelOrder: driver cancel → remove driver, go back to READY for re-match
func (s *foodOrderServiceImpl) DriverCancelOrder(ctx context.Context, orderID string) (*model.FoodOrder, error) {
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

	// Driver can only cancel if food is READY or PICKED_UP (before delivering)
	if order.Status != model.FoodStatusReady && order.Status != model.FoodStatusPickedUp {
		return nil, ErrInvalidTransition
	}
	if order.DriverID == "" {
		return nil, ErrNoDriverAssigned
	}

	order.DriverID = ""
	order.Status = model.FoodStatusReady // back to READY for re-match
	order.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	evt, err := event.NewEvent(event.TopicRideDriverCancelled, "food-order-service", map[string]interface{}{
		"order_id": order.ID, "order_type": "food",
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicRideDriverCancelled, evt)
	}

	return order, nil
}

func (s *foodOrderServiceImpl) GetUserOrders(ctx context.Context, userID string, limit, offset int) ([]*model.FoodOrder, int, error) {
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

// --- Helpers ---

func (s *foodOrderServiceImpl) transitionStatus(ctx context.Context, orderID string, target model.FoodOrderStatus, topic string) (*model.FoodOrder, error) {
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
	if !order.Status.CanTransitionTo(target) {
		return nil, ErrInvalidTransition
	}

	order.Status = target
	order.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, order); err != nil {
		return nil, err
	}

	if s.publisher != nil && topic != "" {
		evt, err := event.NewEvent(topic, "food-order-service", map[string]interface{}{
			"order_id": order.ID, "merchant_id": order.MerchantID,
		})
		if err == nil {
			_ = s.publisher.Publish(ctx, topic, evt)
		}
	}

	return order, nil
}

func (s *foodOrderServiceImpl) publishCancel(ctx context.Context, order *model.FoodOrder) {
	evt, err := event.NewEvent(event.TopicRideCancelled, "food-order-service", model.FoodCancelledEvent{
		OrderID: order.ID, UserID: order.UserID, MerchantID: order.MerchantID,
		CancelledBy: order.CancelledBy, Reason: order.CancelReason,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicRideCancelled, evt)
	}
}

func (s *foodOrderServiceImpl) publishWalletUnlock(ctx context.Context, order *model.FoodOrder) {
	evt, err := event.NewEvent(event.TopicWalletUnlock, "food-order-service", map[string]interface{}{
		"order_id": order.ID, "user_id": order.UserID, "amount": order.TotalAmount,
	})
	if err == nil && s.publisher != nil {
		_ = s.publisher.Publish(ctx, event.TopicWalletUnlock, evt)
	}
}

func calculateDeliveryFee(delivery, merchant model.Location) float64 {
	const earthRadiusKm = 6371.0
	dLat := (merchant.Latitude - delivery.Latitude) * math.Pi / 180
	dLng := (merchant.Longitude - delivery.Longitude) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(delivery.Latitude*math.Pi/180)*math.Cos(merchant.Latitude*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	dist := earthRadiusKm * c
	baseFee := 5000.0
	perKm := 2000.0
	return baseFee + (dist * perKm)
}
