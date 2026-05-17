// Package service implements the business logic for settlement-service.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/settlement-service/internal/model"
	"furab-backend/services/settlement-service/internal/repository"

	"github.com/google/uuid"
)

// Common service errors.
var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrMissingDependency  = errors.New("missing dependency")
	ErrRecipientNotActive = errors.New("settlement recipient is not active")
)

type WalletClient interface {
	CreditBalance(ctx context.Context, walletID string, amount float64, referenceID string) error
}

type DriverClient interface {
	GetDriverWalletIDByOrderID(ctx context.Context, orderID string) (string, error)
}

type MerchantClient interface {
	GetMerchantWalletIDByOrderID(ctx context.Context, orderID string) (string, error)
}

// SettlementService defines settlement orchestration.
type SettlementService interface {
	ProcessSettlement(ctx context.Context, req *model.ProcessSettlementRequest) (*model.ProcessSettlementResponse, error)
}

type settlementServiceImpl struct {
	repo        repository.SettlementRepository
	walletCli   WalletClient
	driverCli   DriverClient
	merchantCli MerchantClient
}

func NewSettlementService(
	repo repository.SettlementRepository,
	walletCli WalletClient,
	driverCli DriverClient,
	merchantCli MerchantClient,
) SettlementService {
	return &settlementServiceImpl{
		repo:        repo,
		walletCli:   walletCli,
		driverCli:   driverCli,
		merchantCli: merchantCli,
	}
}

func (s *settlementServiceImpl) ProcessSettlement(ctx context.Context, req *model.ProcessSettlementRequest) (*model.ProcessSettlementResponse, error) {
	if req == nil || req.PaymentID == "" || req.OrderID == "" || req.TotalAmount <= 0 {
		return nil, ErrInvalidRequest
	}
	if s.repo == nil || s.walletCli == nil || s.driverCli == nil || s.merchantCli == nil {
		return nil, ErrMissingDependency
	}

	existing, err := s.repo.GetSettlementByPaymentID(ctx, req.PaymentID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == model.StatusSuccess {
		return &model.ProcessSettlementResponse{
			Status:         mapStatus(existing.Status),
			DriverAmount:   existing.DriverAmount,
			MerchantAmount: existing.MerchantAmount,
			PlatformFee:    existing.PlatformFee,
		}, nil
	}

	var settlement *model.Settlement
	if existing != nil {
		settlement = existing
	} else {
		driverAmount := req.TotalAmount * 0.80
		merchantAmount := req.TotalAmount * 0.15
		platformFee := req.TotalAmount - driverAmount - merchantAmount

		now := time.Now().UTC()
		settlement = &model.Settlement{
			ID:             uuid.New().String(),
			PaymentID:      req.PaymentID,
			OrderID:        req.OrderID,
			TotalAmount:    req.TotalAmount,
			DriverAmount:   driverAmount,
			MerchantAmount: merchantAmount,
			PlatformFee:    platformFee,
			Status:         model.StatusPending,
			IdempotencyKey: req.PaymentID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.repo.CreateSettlement(ctx, settlement); err != nil {
			return nil, err
		}
	}
	driverAmount := settlement.DriverAmount
	merchantAmount := settlement.MerchantAmount
	platformFee := settlement.PlatformFee

	driverWalletID, err := s.driverCli.GetDriverWalletIDByOrderID(ctx, req.OrderID)
	if err != nil {
		_ = s.repo.UpdateSettlementStatus(ctx, settlement.ID, model.StatusFailed)
		return failedSettlementResponse(driverAmount, merchantAmount, platformFee), err
	}
	if driverWalletID == "" {
		_ = s.repo.UpdateSettlementStatus(ctx, settlement.ID, model.StatusFailed)
		return failedSettlementResponse(driverAmount, merchantAmount, platformFee), ErrRecipientNotActive
	}

	merchantWalletID, err := s.merchantCli.GetMerchantWalletIDByOrderID(ctx, req.OrderID)
	if err != nil {
		_ = s.repo.UpdateSettlementStatus(ctx, settlement.ID, model.StatusFailed)
		return failedSettlementResponse(driverAmount, merchantAmount, platformFee), err
	}
	if merchantWalletID == "" {
		_ = s.repo.UpdateSettlementStatus(ctx, settlement.ID, model.StatusFailed)
		return failedSettlementResponse(driverAmount, merchantAmount, platformFee), ErrRecipientNotActive
	}

	if err := s.walletCli.CreditBalance(ctx, driverWalletID, driverAmount, fmt.Sprintf("SETTLE-DRV-%s", req.PaymentID)); err != nil {
		_ = s.repo.UpdateSettlementStatus(ctx, settlement.ID, model.StatusFailed)
		return failedSettlementResponse(driverAmount, merchantAmount, platformFee), err
	}
	if err := s.walletCli.CreditBalance(ctx, merchantWalletID, merchantAmount, fmt.Sprintf("SETTLE-MER-%s", req.PaymentID)); err != nil {
		_ = s.repo.UpdateSettlementStatus(ctx, settlement.ID, model.StatusFailed)
		return failedSettlementResponse(driverAmount, merchantAmount, platformFee), err
	}

	if err := s.repo.UpdateSettlementStatus(ctx, settlement.ID, model.StatusSuccess); err != nil {
		return nil, err
	}

	return &model.ProcessSettlementResponse{
		Status:         "SUCCESS",
		DriverAmount:   driverAmount,
		MerchantAmount: merchantAmount,
		PlatformFee:    platformFee,
	}, nil
}

func mapStatus(s model.SettlementStatus) string {
	switch s {
	case model.StatusSuccess:
		return "SUCCESS"
	case model.StatusFailed:
		return "FAILED"
	default:
		return "FAILED"
	}
}

func failedSettlementResponse(driverAmount, merchantAmount, platformFee float64) *model.ProcessSettlementResponse {
	return &model.ProcessSettlementResponse{
		Status:         "FAILED",
		DriverAmount:   driverAmount,
		MerchantAmount: merchantAmount,
		PlatformFee:    platformFee,
	}
}
