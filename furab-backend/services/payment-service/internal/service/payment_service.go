// Package service implements the business logic for payment-service.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/payment-service/internal/model"
	"furab-backend/services/payment-service/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrInvalidRequest  = errors.New("invalid request")
	ErrPaymentNotFound = errors.New("payment not found")
)

type PricingClient interface {
	GetTotalAmount(ctx context.Context, orderID string) (float64, error)
}

type PromoClient interface {
	ApplyPromo(ctx context.Context, promoCode string, totalAmount float64) (finalAmount float64, discountAmount float64, err error)
}

type WalletClient interface {
	LockBalance(ctx context.Context, userID string, amount float64, reference string) error
	UnlockBalance(ctx context.Context, userID string, amount float64, reference string) error
	DeductBalance(ctx context.Context, userID string, amount float64, reference string) error
	CreditBalance(ctx context.Context, userID string, amount float64, reference string) error
}

type SettlementClient interface {
	TriggerSettlement(ctx context.Context, paymentID, orderID string, finalAmount float64) error
}

// PaymentService defines the interface for payment-service business logic.
type PaymentService interface {
	InitiatePayment(ctx context.Context, req *model.InitiatePaymentRequest) (*model.Payment, error)
	CapturePayment(ctx context.Context, paymentID string) (*model.Payment, error)
	CancelPayment(ctx context.Context, paymentID string) (*model.Payment, error)
	RefundPayment(ctx context.Context, paymentID string) (*model.Payment, error)
	GetPayment(ctx context.Context, paymentID string) (*model.Payment, error)
}

// paymentServiceImpl is the concrete implementation of PaymentService.
type paymentServiceImpl struct {
	repo       repository.PaymentRepository
	pricingCli PricingClient
	promoCli   PromoClient
	walletCli  WalletClient
	settleCli  SettlementClient
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(
	repo repository.PaymentRepository,
	pricingCli PricingClient,
	promoCli PromoClient,
	walletCli WalletClient,
	settleCli SettlementClient,
) PaymentService {
	return &paymentServiceImpl{
		repo:       repo,
		pricingCli: pricingCli,
		promoCli:   promoCli,
		walletCli:  walletCli,
		settleCli:  settleCli,
	}
}

func (s *paymentServiceImpl) InitiatePayment(ctx context.Context, req *model.InitiatePaymentRequest) (*model.Payment, error) {
	if req == nil || req.OrderID == "" || req.UserID == "" || req.PaymentMethod == "" || req.PaymentDetail == "" {
		return nil, ErrInvalidRequest
	}

	if req.IdempotencyKey != "" {
		existing, err := s.repo.GetPaymentByIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}

	baseAmount := req.Amount
	if baseAmount <= 0 {
		if s.pricingCli == nil {
			return nil, errors.New("missing pricing client")
		}
		amount, err := s.pricingCli.GetTotalAmount(ctx, req.OrderID)
		if err != nil {
			return nil, err
		}
		baseAmount = amount
	}

	finalAmount := baseAmount
	if req.PromoCode != "" && s.promoCli != nil {
		discounted, _, err := s.promoCli.ApplyPromo(ctx, req.PromoCode, baseAmount)
		if err != nil {
			return nil, err
		}
		finalAmount = discounted
	}

	now := time.Now().UTC()
	p := &model.Payment{
		ID:                   uuid.New().String(),
		OrderID:              req.OrderID,
		UserID:               req.UserID,
		Amount:               baseAmount,
		FinalAmount:          finalAmount,
		MethodID:             req.PaymentMethod,
		PaymentDetail:        req.PaymentDetail,
		PaymentStatus:        model.StatusPending,
		TransactionReference: fmt.Sprintf("TXN-%s", req.OrderID),
		IdempotencyKey:       req.IdempotencyKey,
		TransactionTime:      now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.repo.CreatePayment(ctx, p); err != nil {
		return nil, err
	}
	_ = s.repo.CreatePaymentLog(ctx, p.ID, model.StatusPending)

	lockRef := fmt.Sprintf("AUTH-%s", p.TransactionReference)
	if err := s.walletCli.LockBalance(ctx, p.UserID, p.FinalAmount, lockRef); err != nil {
		_ = s.repo.UpdatePaymentStatus(ctx, p.ID, model.StatusFailed)
		_ = s.repo.CreatePaymentLog(ctx, p.ID, model.StatusFailed)
		return nil, err
	}

	if err := s.repo.UpdatePaymentStatus(ctx, p.ID, model.StatusAuthorized); err != nil {
		return nil, err
	}
	_ = s.repo.CreatePaymentLog(ctx, p.ID, model.StatusAuthorized)
	p.PaymentStatus = model.StatusAuthorized
	return p, nil
}

func (s *paymentServiceImpl) CapturePayment(ctx context.Context, paymentID string) (*model.Payment, error) {
	p, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPaymentNotFound
	}
	if p.PaymentStatus != model.StatusAuthorized {
		return nil, errors.New("payment is not in authorized state")
	}

	ref := fmt.Sprintf("CAPTURE-%s", p.TransactionReference)
	if err := s.walletCli.DeductBalance(ctx, p.UserID, p.FinalAmount, ref); err != nil {
		_ = s.repo.UpdatePaymentStatus(ctx, p.ID, model.StatusFailed)
		_ = s.repo.CreatePaymentLog(ctx, p.ID, model.StatusFailed)
		return nil, err
	}

	if err := s.repo.UpdatePaymentStatus(ctx, p.ID, model.StatusCaptured); err != nil {
		return nil, err
	}
	_ = s.repo.CreatePaymentLog(ctx, p.ID, model.StatusCaptured)
	p.PaymentStatus = model.StatusCaptured

	if s.settleCli != nil {
		_ = s.settleCli.TriggerSettlement(ctx, p.ID, p.OrderID, p.FinalAmount)
	}
	return p, nil
}

func (s *paymentServiceImpl) CancelPayment(ctx context.Context, paymentID string) (*model.Payment, error) {
	p, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPaymentNotFound
	}
	if p.PaymentStatus != model.StatusAuthorized && p.PaymentStatus != model.StatusPending {
		return nil, errors.New("payment cannot be cancelled in current state")
	}

	ref := fmt.Sprintf("CANCEL-%s", p.TransactionReference)
	if err := s.walletCli.UnlockBalance(ctx, p.UserID, p.FinalAmount, ref); err != nil {
		return nil, err
	}
	if err := s.repo.UpdatePaymentStatus(ctx, p.ID, model.StatusCancelled); err != nil {
		return nil, err
	}
	_ = s.repo.CreatePaymentLog(ctx, p.ID, model.StatusCancelled)
	p.PaymentStatus = model.StatusCancelled
	return p, nil
}

func (s *paymentServiceImpl) RefundPayment(ctx context.Context, paymentID string) (*model.Payment, error) {
	p, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPaymentNotFound
	}
	if p.PaymentStatus != model.StatusCaptured {
		return nil, errors.New("payment cannot be refunded in current state")
	}

	ref := fmt.Sprintf("REFUND-%s", p.TransactionReference)
	if err := s.walletCli.CreditBalance(ctx, p.UserID, p.FinalAmount, ref); err != nil {
		return nil, err
	}
	if err := s.repo.UpdatePaymentStatus(ctx, p.ID, model.StatusRefunded); err != nil {
		return nil, err
	}
	_ = s.repo.CreatePaymentLog(ctx, p.ID, model.StatusRefunded)
	p.PaymentStatus = model.StatusRefunded
	return p, nil
}

func (s *paymentServiceImpl) GetPayment(ctx context.Context, paymentID string) (*model.Payment, error) {
	p, err := s.repo.GetPaymentByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPaymentNotFound
	}
	return p, nil
}
