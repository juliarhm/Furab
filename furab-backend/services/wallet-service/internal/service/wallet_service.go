// Package service implements the business logic for wallet-service.
package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"furab-backend/services/wallet-service/internal/model"
	"furab-backend/services/wallet-service/internal/repository"

	"github.com/google/uuid"
)

// Common service errors.
var (
	ErrInvalidRequest      = errors.New("invalid request")
	ErrMissingRepository   = errors.New("missing repository")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrReferenceNotFound   = errors.New("reference transaction not found")
	ErrInvalidRefund       = errors.New("invalid refund reference")
)

// WalletService defines the interface for wallet-service business logic.
type WalletService interface {
	// HoldBalance places a temporary hold on available balance.
	HoldBalance(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error)
	// ReleaseBalance releases previously held balance.
	ReleaseBalance(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error)
	// DebitBalance deducts user's balance (capture).
	DebitBalance(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error)
	// CreditBalance adds balance to user's wallet.
	CreditBalance(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error)
	// Refund adds balance back as compensation/refund.
	Refund(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error)
}

// walletServiceImpl is the concrete implementation of WalletService.
type walletServiceImpl struct {
	repo      repository.WalletRepository
	userLocks sync.Map
}

// NewWalletService creates a new WalletService.
func NewWalletService(repo repository.WalletRepository) WalletService {
	return &walletServiceImpl{repo: repo}
}

// HoldBalance reduces available balance as pre-authorization hold.
func (s *walletServiceImpl) HoldBalance(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error) {
	return s.changeBalance(ctx, userID, -amount, amount, referenceID, model.TypeHold)
}

// ReleaseBalance returns held amount to available balance.
func (s *walletServiceImpl) ReleaseBalance(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error) {
	return s.changeBalance(ctx, userID, amount, amount, referenceID, model.TypeRelease)
}

// DebitBalance deducts balance permanently.
func (s *walletServiceImpl) DebitBalance(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error) {
	return s.changeBalance(ctx, userID, -amount, amount, referenceID, model.TypeDebit)
}

// CreditBalance adds amount to wallet.
func (s *walletServiceImpl) CreditBalance(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error) {
	return s.changeBalance(ctx, userID, amount, amount, referenceID, model.TypeCredit)
}

// Refund adds refunded amount to wallet.
func (s *walletServiceImpl) Refund(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error) {
	return s.changeBalance(ctx, userID, amount, amount, referenceID, model.TypeRefund)
}

func (s *walletServiceImpl) changeBalance(
	ctx context.Context,
	userID string,
	delta float64,
	amount float64,
	referenceID string,
	typ model.TransactionType,
) (*model.WalletResult, error) {
	if userID == "" || amount <= 0 {
		return nil, ErrInvalidRequest
	}
	if s.repo == nil {
		return nil, ErrMissingRepository
	}
	if referenceID != "" {
		if err := s.validateReferenceForType(ctx, referenceID, typ); err != nil {
			return nil, err
		}

		existing, err := s.repo.GetTransactionByReference(ctx, referenceID, typ)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return &model.WalletResult{
				Status:         existing.Status,
				CurrentBalance: existing.CurrentBalance,
				TransactionID:  existing.ID,
			}, nil
		}
	}

	lock := s.userLock(userID)
	lock.Lock()
	defer lock.Unlock()

	w, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	newBal := w.Balance + delta
	if newBal < 0 {
		return &model.WalletResult{
			Status:         model.StatusFailed,
			CurrentBalance: w.Balance,
			TransactionID:  "",
		}, ErrInsufficientBalance
	}

	if err := s.repo.UpdateBalance(ctx, w.ID, newBal); err != nil {
		return nil, err
	}

	tx := &model.Transaction{
		ID:             uuid.New().String(),
		WalletID:       w.ID,
		ReferenceID:    referenceID,
		Type:           typ,
		Amount:         amount,
		Status:         model.StatusSuccess,
		CurrentBalance: newBal,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.repo.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}

	return &model.WalletResult{
		Status:         model.StatusSuccess,
		CurrentBalance: newBal,
		TransactionID:  tx.ID,
	}, nil
}

func (s *walletServiceImpl) validateReferenceForType(ctx context.Context, referenceID string, typ model.TransactionType) error {
	switch typ {
	case model.TypeRelease:
		holdTx, err := s.repo.GetTransactionByReference(ctx, referenceID, model.TypeHold)
		if err != nil {
			return err
		}
		if holdTx == nil || holdTx.Status != model.StatusSuccess {
			return ErrReferenceNotFound
		}
	case model.TypeRefund:
		debitTx, err := s.repo.GetTransactionByReference(ctx, referenceID, model.TypeDebit)
		if err != nil {
			return err
		}
		if debitTx == nil || debitTx.Status != model.StatusSuccess {
			return ErrInvalidRefund
		}
	}
	return nil
}

func (s *walletServiceImpl) userLock(userID string) *sync.Mutex {
	mu, _ := s.userLocks.LoadOrStore(userID, &sync.Mutex{})
	return mu.(*sync.Mutex)
}
