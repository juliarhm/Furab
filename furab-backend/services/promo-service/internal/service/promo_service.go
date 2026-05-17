// Package service implements the business logic for promo-service.
package service

import (
	"context"
	"fmt"
	"time"

	"furab-backend/services/promo-service/internal/client"
	"furab-backend/services/promo-service/internal/model"
	"furab-backend/services/promo-service/internal/repository"
)

var (
	ErrPromoCodeRequired = fmt.Errorf("promo code is required")
)

// PromoService defines the interface for promo validation and application.
type PromoService interface {
	ValidatePromo(ctx context.Context, promoCode, userID, orderID string, totalAmount float64) (*model.PromoValidationResponse, error)
}

// promoServiceImpl is the concrete implementation of PromoService.
type promoServiceImpl struct {
	repo        repository.PromoRepository
	orderClient client.OrderClient
	userClient  client.UserClient
}

// NewPromoService creates a new PromoService.
func NewPromoService(repo repository.PromoRepository, orderClient client.OrderClient, userClient client.UserClient) PromoService {
	return &promoServiceImpl{
		repo:        repo,
		orderClient: orderClient,
		userClient:  userClient,
	}
}

func (s *promoServiceImpl) ValidatePromo(ctx context.Context, promoCode, userID, orderID string, totalAmount float64) (*model.PromoValidationResponse, error) {
	response := &model.PromoValidationResponse{
		IsValid:        false,
		DiscountAmount: 0,
		FinalAmount:    totalAmount,
	}

	if promoCode == "" {
		response.ErrorMessage = "promo code is required"
		return response, nil
	}

	promo, err := s.repo.GetPromoByCode(ctx, promoCode)
	if err != nil {
		if err == repository.ErrPromoNotFound {
			response.ErrorMessage = "promo not found or invalid"
			return response, nil
		}
		return nil, fmt.Errorf("failed to get promo: %w", err)
	}

	if time.Now().After(promo.ExpiryDate) {
		response.ErrorMessage = "promo has expired"
		return response, nil
	}

	if promo.UsageLimit > 0 && promo.UsageCount >= promo.UsageLimit {
		response.ErrorMessage = "promo usage limit exceeded"
		return response, nil
	}

	if totalAmount < promo.MinPurchase {
		response.ErrorMessage = fmt.Sprintf("minimum purchase not met. minimum: %.2f", promo.MinPurchase)
		return response, nil
	}

	orderValid, err := s.orderClient.ValidateOrderPromo(ctx, orderID, promoCode)
	if err != nil || !orderValid {
		response.ErrorMessage = "order does not meet promo conditions"
		return response, nil
	}

	userValid, err := s.userClient.ValidateUserPromo(ctx, userID, promoCode)
	if err != nil || !userValid {
		response.ErrorMessage = "user is not eligible for this promo"
		return response, nil
	}

	discountAmount := calculateDiscount(totalAmount, promo)
	finalAmount := totalAmount - discountAmount
	if finalAmount < 0 {
		finalAmount = 0
	}

	if err := s.repo.IncrementUsage(ctx, promo.PromoID); err != nil {
		if err == repository.ErrPromoUsageExceeded {
			response.ErrorMessage = "promo usage limit exceeded"
			return response, nil
		}
		return nil, fmt.Errorf("failed to increment promo usage: %w", err)
	}

	response.IsValid = true
	response.DiscountAmount = discountAmount
	response.FinalAmount = finalAmount
	return response, nil
}

func calculateDiscount(totalAmount float64, promo *model.Promo) float64 {
	switch promo.DiscountType {
	case "percentage":
		discount := totalAmount * promo.DiscountValue
		if promo.MaxDiscount > 0 && discount > promo.MaxDiscount {
			discount = promo.MaxDiscount
		}
		return discount
	case "fixed":
		return promo.DiscountValue
	default:
		return 0
	}
}
