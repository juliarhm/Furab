// Package service implements the business logic for pricing-service.
package service

import (
	"context"
	"errors"
	"fmt"

	"furab-backend/services/pricing-service/internal/client"
	"furab-backend/services/pricing-service/internal/model"
	"furab-backend/services/pricing-service/internal/repository"
)

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrMissingDependency = errors.New("missing dependency")
	ErrNoOrderItems   = errors.New("order contains no items")
)

// PriceService defines the interface for pricing calculation.
type PriceService interface {
	CalculatePrice(ctx context.Context, orderID string) (*model.PriceCalculationResponse, error)
}

// priceServiceImpl is the concrete implementation of PriceService.
type priceServiceImpl struct {
	repo           repository.PriceRepository
	orderClient    client.OrderClient
	locationClient client.LocationClient
}

// NewPriceService creates a new PriceService.
func NewPriceService(repo repository.PriceRepository, orderClient client.OrderClient, locationClient client.LocationClient) PriceService {
	return &priceServiceImpl{
		repo:           repo,
		orderClient:    orderClient,
		locationClient: locationClient,
	}
}

func (s *priceServiceImpl) CalculatePrice(ctx context.Context, orderID string) (*model.PriceCalculationResponse, error) {
	if orderID == "" {
		return nil, ErrInvalidRequest
	}
	if s.repo == nil || s.orderClient == nil || s.locationClient == nil {
		return nil, ErrMissingDependency
	}

	items, err := s.orderClient.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch order items: %w", err)
	}

	if len(items) == 0 {
		return nil, ErrNoOrderItems
	}

	deliveryDistance, err := s.locationClient.GetDeliveryDistance(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch delivery distance: %w", err)
	}

	itemPrice := 0.0
	for _, item := range items {
		itemPrice += float64(item.Quantity) * item.UnitPrice
	}

	deliveryRule, err := s.repo.GetPricingRuleByType(ctx, "delivery")
	if err != nil {
		return nil, fmt.Errorf("delivery rule missing: %w", err)
	}

	serviceRule, err := s.repo.GetPricingRuleByType(ctx, "service")
	if err != nil {
		return nil, fmt.Errorf("service rule missing: %w", err)
	}

	deliveryFee := deliveryDistance * deliveryRule.Value
	serviceFee := itemPrice * serviceRule.Value
	totalAmount := itemPrice + deliveryFee + serviceFee

	return &model.PriceCalculationResponse{
		OrderID:     orderID,
		TotalAmount: totalAmount,
		ItemPrice:   itemPrice,
		DeliveryFee: deliveryFee,
		ServiceFee:  serviceFee,
	}, nil
}
