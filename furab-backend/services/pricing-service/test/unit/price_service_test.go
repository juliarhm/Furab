// Package unit contains unit tests for pricing-service.
// Unit tests do NOT access any database or external service.
package unit

import (
	"context"
	"errors"
	"strings"
	"testing"

	"furab-backend/services/pricing-service/internal/model"
	"furab-backend/services/pricing-service/internal/repository"
	"furab-backend/services/pricing-service/internal/service"
	"furab-backend/services/pricing-service/test/unit/mock"

	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (service.PriceService, *mock.MockPriceRepository, *mock.MockOrderClient, *mock.MockLocationClient, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockPriceRepository(ctrl)
	mockOrder := mock.NewMockOrderClient(ctrl)
	mockLocation := mock.NewMockLocationClient(ctrl)

	svc := service.NewPriceService(mockRepo, mockOrder, mockLocation)
	return svc, mockRepo, mockOrder, mockLocation, ctrl
}

func TestCalculatePrice_Success(t *testing.T) {
	svc, mockRepo, mockOrder, mockLocation, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	
	mockOrder.EXPECT().GetOrderItems(ctx, "order-123").Return([]model.OrderItem{
		{ProductID: "item-1", Quantity: 2, UnitPrice: 10000},
		{ProductID: "item-2", Quantity: 1, UnitPrice: 15000},
	}, nil).AnyTimes()

	mockLocation.EXPECT().GetDeliveryDistance(ctx, "order-123").Return(5.0, nil).AnyTimes()

	mockRepo.EXPECT().GetPricingRuleByType(ctx, "delivery").Return(&model.PriceRule{
		RuleID: "delivery-per-km",
		Type:   "delivery",
		Value:  5000,
	}, nil)
	mockRepo.EXPECT().GetPricingRuleByType(ctx, "service").Return(&model.PriceRule{
		RuleID: "service-percent",
		Type:   "service",
		Value:  0.05,
	}, nil)

	result, err := svc.CalculatePrice(ctx, "order-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OrderID != "order-123" {
		t.Fatalf("expected order id order-123, got %s", result.OrderID)
	}

	if result.ItemPrice != 35000 {
		t.Fatalf("expected item price 35000, got %v", result.ItemPrice)
	}

	if result.DeliveryFee != 25000 {
		t.Fatalf("expected delivery fee 25000, got %v", result.DeliveryFee)
	}

	if result.ServiceFee != 1750 {
		t.Fatalf("expected service fee 1750, got %v", result.ServiceFee)
	}

	if result.TotalAmount != 61750 {
		t.Fatalf("expected total amount 61750, got %v", result.TotalAmount)
	}
}

func TestCalculatePrice_InvalidRequest_EmptyOrderID(t *testing.T) {
	svc, _, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.CalculatePrice(context.Background(), "")
	if err == nil {
		t.Fatal("expected an error when order ID is missing")
	}

	if !errors.Is(err, service.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestCalculatePrice_NoOrderItems(t *testing.T) {
	svc, _, mockOrder, mockLocation, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockOrder.EXPECT().GetOrderItems(ctx, "order-empty").Return([]model.OrderItem{}, nil).AnyTimes()
	mockLocation.EXPECT().GetDeliveryDistance(ctx, "order-empty").Return(2.0, nil).AnyTimes()

	_, err := svc.CalculatePrice(ctx, "order-empty")
	if !errors.Is(err, service.ErrNoOrderItems) {
		t.Fatalf("expected ErrNoOrderItems, got %v", err)
	}
}

func TestCalculatePrice_MissingPricingRule(t *testing.T) {
	svc, mockRepo, mockOrder, mockLocation, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	
	mockOrder.EXPECT().GetOrderItems(ctx, "order-123").Return([]model.OrderItem{
		{ProductID: "item-1", Quantity: 2, UnitPrice: 10000},
		{ProductID: "item-2", Quantity: 1, UnitPrice: 15000},
	}, nil).AnyTimes()
	
	mockLocation.EXPECT().GetDeliveryDistance(ctx, "order-123").Return(5.0, nil).AnyTimes()

	mockRepo.EXPECT().GetPricingRuleByType(ctx, "delivery").Return(nil, repository.ErrPriceRuleNotFound)

	_, err := svc.CalculatePrice(ctx, "order-123")
	if err == nil {
		t.Fatal("expected error when delivery pricing rule is missing")
	}
}

func TestCalculatePrice_DistanceLogic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo := mock.NewMockPriceRepository(ctrl)
	mockOrderNear := mock.NewMockOrderClient(ctrl)
	mockLocationNear := mock.NewMockLocationClient(ctrl)
	mockOrderFar := mock.NewMockOrderClient(ctrl)
	mockLocationFar := mock.NewMockLocationClient(ctrl)

	items := []model.OrderItem{
		{ProductID: "item-1", Quantity: 1, UnitPrice: 10000},
	}
	
	mockOrderNear.EXPECT().GetOrderItems(ctx, "order-near").Return(items, nil).AnyTimes()
	mockLocationNear.EXPECT().GetDeliveryDistance(ctx, "order-near").Return(1.0, nil).AnyTimes()
	
	mockOrderFar.EXPECT().GetOrderItems(ctx, "order-far").Return(items, nil).AnyTimes()
	mockLocationFar.EXPECT().GetDeliveryDistance(ctx, "order-far").Return(10.0, nil).AnyTimes()

	svcNear := service.NewPriceService(mockRepo, mockOrderNear, mockLocationNear)
	svcFar := service.NewPriceService(mockRepo, mockOrderFar, mockLocationFar)

	mockRepo.EXPECT().GetPricingRuleByType(gomock.Any(), "delivery").Return(&model.PriceRule{Type: "delivery", Value: 5000}, nil).Times(2)
	mockRepo.EXPECT().GetPricingRuleByType(gomock.Any(), "service").Return(&model.PriceRule{Type: "service", Value: 0.05}, nil).Times(2)

	nearRes, err := svcNear.CalculatePrice(ctx, "order-near")
	if err != nil {
		t.Fatalf("unexpected error for near distance: %v", err)
	}
	farRes, err := svcFar.CalculatePrice(ctx, "order-far")
	if err != nil {
		t.Fatalf("unexpected error for far distance: %v", err)
	}

	if nearRes.DeliveryFee != 5000 {
		t.Fatalf("expected near delivery fee 5000, got %v", nearRes.DeliveryFee)
	}
	if farRes.DeliveryFee != 50000 {
		t.Fatalf("expected far delivery fee 50000, got %v", farRes.DeliveryFee)
	}
	if farRes.TotalAmount <= nearRes.TotalAmount {
		t.Fatalf("expected far total (%v) > near total (%v)", farRes.TotalAmount, nearRes.TotalAmount)
	}
}

func TestCalculatePrice_TotalSumValidation(t *testing.T) {
	svc, mockRepo, mockOrder, mockLocation, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	
	mockOrder.EXPECT().GetOrderItems(ctx, "order-sum-check").Return([]model.OrderItem{
		{ProductID: "item-1", Quantity: 2, UnitPrice: 10000},
		{ProductID: "item-2", Quantity: 1, UnitPrice: 15000},
	}, nil).AnyTimes()
	mockLocation.EXPECT().GetDeliveryDistance(ctx, "order-sum-check").Return(5.0, nil).AnyTimes()

	mockRepo.EXPECT().GetPricingRuleByType(ctx, "delivery").Return(&model.PriceRule{Type: "delivery", Value: 5000}, nil)
	mockRepo.EXPECT().GetPricingRuleByType(ctx, "service").Return(&model.PriceRule{Type: "service", Value: 0.05}, nil)

	res, err := svc.CalculatePrice(ctx, "order-sum-check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedTotal := res.ItemPrice + res.DeliveryFee + res.ServiceFee
	if res.TotalAmount != expectedTotal {
		t.Fatalf("total mismatch: total=%v item=%v delivery=%v service=%v",
			res.TotalAmount, res.ItemPrice, res.DeliveryFee, res.ServiceFee)
	}
}

func TestCalculatePrice_LocationServiceTimeout(t *testing.T) {
	svc, _, mockOrder, mockLocation, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	
	mockOrder.EXPECT().GetOrderItems(ctx, "order-timeout").Return([]model.OrderItem{
		{ProductID: "item-1", Quantity: 1, UnitPrice: 10000},
	}, nil).AnyTimes()
	
	mockLocation.EXPECT().GetDeliveryDistance(ctx, "order-timeout").Return(0.0, errors.New("location timeout")).AnyTimes()

	_, err := svc.CalculatePrice(ctx, "order-timeout")
	if err == nil {
		t.Fatal("expected error when location service times out")
	}
	if !strings.Contains(err.Error(), "failed to fetch delivery distance") {
		t.Fatalf("expected delivery distance error wrapping, got: %v", err)
	}
}

func TestCalculatePrice_OrderServiceDown(t *testing.T) {
	svc, _, mockOrder, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()

	mockOrder.EXPECT().GetOrderItems(ctx, "order-down").Return(nil, errors.New("order service unavailable")).AnyTimes()
	// Location client is not expected to be called if Order client fails early.

	_, err := svc.CalculatePrice(ctx, "order-down")
	if err == nil {
		t.Fatal("expected error when order service is down")
	}
	if !strings.Contains(err.Error(), "failed to fetch order items") {
		t.Fatalf("expected order items error wrapping, got: %v", err)
	}
}
