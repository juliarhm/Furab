// Package unit contains unit tests for wallet-service.
// Unit tests do NOT access any database or external service.
// All dependencies are mocked using gomock.
package unit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"furab-backend/services/wallet-service/internal/model"
	"furab-backend/services/wallet-service/internal/service"
	"furab-backend/services/wallet-service/test/unit/mock"

	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (service.WalletService, *mock.MockWalletRepository, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo)
	return svc, mockRepo, ctrl
}

func sampleWallet(balance float64) *model.Wallet {
	return &model.Wallet{
		ID:      "wallet-123",
		UserID:  "user-123",
		Balance: balance,
	}
}

func TestHoldBalance_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	wallet := sampleWallet(100000)

	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-hold-1", model.TypeHold).
		Return(nil, nil)
	mockRepo.EXPECT().
		GetByUserID(ctx, wallet.UserID).
		Return(wallet, nil)
	mockRepo.EXPECT().
		UpdateBalance(ctx, wallet.ID, 70000.0).
		Return(nil)
	mockRepo.EXPECT().
		CreateTransaction(ctx, gomock.Any()).
		Return(nil)

	res, err := svc.HoldBalance(ctx, wallet.UserID, 30000, "ref-hold-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != model.StatusSuccess {
		t.Fatalf("expected SUCCESS status, got: %v", res.Status)
	}
	if res.CurrentBalance != 70000 {
		t.Fatalf("expected current balance 70000, got: %v", res.CurrentBalance)
	}
	if res.TransactionID == "" {
		t.Fatal("expected non-empty transaction ID")
	}
}

func TestHoldBalance_InsufficientBalance(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	wallet := sampleWallet(10000)

	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-hold-2", model.TypeHold).
		Return(nil, nil)
	mockRepo.EXPECT().
		GetByUserID(ctx, wallet.UserID).
		Return(wallet, nil)

	res, err := svc.HoldBalance(ctx, wallet.UserID, 20000, "ref-hold-2")
	if err == nil {
		t.Fatal("expected insufficient balance error")
	}
	if res == nil {
		t.Fatal("expected failed result payload")
	}
	if res.Status != model.StatusFailed {
		t.Fatalf("expected FAILED status, got: %v", res.Status)
	}
	if res.CurrentBalance != 10000 {
		t.Fatalf("expected unchanged balance 10000, got: %v", res.CurrentBalance)
	}
}

func TestReleaseBalance_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	wallet := sampleWallet(70000)

	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-release-1", model.TypeHold).
		Return(&model.Transaction{
			ID:     "tx-hold-1",
			Status: model.StatusSuccess,
		}, nil)
	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-release-1", model.TypeRelease).
		Return(nil, nil)
	mockRepo.EXPECT().
		GetByUserID(ctx, wallet.UserID).
		Return(wallet, nil)
	mockRepo.EXPECT().
		UpdateBalance(ctx, wallet.ID, 100000.0).
		Return(nil)
	mockRepo.EXPECT().
		CreateTransaction(ctx, gomock.Any()).
		Return(nil)

	res, err := svc.ReleaseBalance(ctx, wallet.UserID, 30000, "ref-release-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != model.StatusSuccess || res.CurrentBalance != 100000 {
		t.Fatalf("unexpected release result: %+v", res)
	}
}

func TestDebitBalance_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	wallet := sampleWallet(100000)

	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-debit-1", model.TypeDebit).
		Return(nil, nil)
	mockRepo.EXPECT().
		GetByUserID(ctx, wallet.UserID).
		Return(wallet, nil)
	mockRepo.EXPECT().
		UpdateBalance(ctx, wallet.ID, 80000.0).
		Return(nil)
	mockRepo.EXPECT().
		CreateTransaction(ctx, gomock.Any()).
		Return(nil)

	res, err := svc.DebitBalance(ctx, wallet.UserID, 20000, "ref-debit-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != model.StatusSuccess || res.CurrentBalance != 80000 {
		t.Fatalf("unexpected debit result: %+v", res)
	}
}

func TestCreditBalance_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	wallet := sampleWallet(80000)

	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-credit-1", model.TypeCredit).
		Return(nil, nil)
	mockRepo.EXPECT().
		GetByUserID(ctx, wallet.UserID).
		Return(wallet, nil)
	mockRepo.EXPECT().
		UpdateBalance(ctx, wallet.ID, 100000.0).
		Return(nil)
	mockRepo.EXPECT().
		CreateTransaction(ctx, gomock.Any()).
		Return(nil)

	res, err := svc.CreditBalance(ctx, wallet.UserID, 20000, "ref-credit-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != model.StatusSuccess || res.CurrentBalance != 100000 {
		t.Fatalf("unexpected credit result: %+v", res)
	}
}

func TestRefund_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	wallet := sampleWallet(50000)

	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-refund-1", model.TypeDebit).
		Return(&model.Transaction{
			ID:     "tx-debit-1",
			Status: model.StatusSuccess,
		}, nil)
	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-refund-1", model.TypeRefund).
		Return(nil, nil)
	mockRepo.EXPECT().
		GetByUserID(ctx, wallet.UserID).
		Return(wallet, nil)
	mockRepo.EXPECT().
		UpdateBalance(ctx, wallet.ID, 60000.0).
		Return(nil)
	mockRepo.EXPECT().
		CreateTransaction(ctx, gomock.Any()).
		Return(nil)

	res, err := svc.Refund(ctx, wallet.UserID, 10000, "ref-refund-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != model.StatusSuccess || res.CurrentBalance != 60000 {
		t.Fatalf("unexpected refund result: %+v", res)
	}
}

func TestDebitBalance_IdempotencyByReference_Success(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	existingTx := &model.Transaction{
		ID:             "tx-existing",
		Status:         model.StatusSuccess,
		CurrentBalance: 90000,
	}

	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-idempotent-1", model.TypeDebit).
		Return(existingTx, nil)

	res, err := svc.DebitBalance(ctx, "user-123", 10000, "ref-idempotent-1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.TransactionID != "tx-existing" {
		t.Fatalf("expected same transaction ID, got: %s", res.TransactionID)
	}
	if res.CurrentBalance != 90000 {
		t.Fatalf("expected current balance 90000, got: %v", res.CurrentBalance)
	}
}

func TestCreditBalance_InvalidRequest_EmptyUserID(t *testing.T) {
	svc, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.CreditBalance(context.Background(), "", 10000, "ref-invalid-1")
	if err == nil {
		t.Fatal("expected error for empty user ID")
	}
}

func TestCreditBalance_InvalidRequest_ZeroAmount(t *testing.T) {
	svc, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.CreditBalance(context.Background(), "user-123", 0, "ref-invalid-2")
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestHoldBalance_UpdateBalanceFailed(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	wallet := sampleWallet(100000)
	dbErr := errors.New("update failed")

	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-hold-3", model.TypeHold).
		Return(nil, nil)
	mockRepo.EXPECT().
		GetByUserID(ctx, wallet.UserID).
		Return(wallet, nil)
	mockRepo.EXPECT().
		UpdateBalance(ctx, wallet.ID, 50000.0).
		Return(dbErr)

	_, err := svc.HoldBalance(ctx, wallet.UserID, 50000, "ref-hold-3")
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected update error, got: %v", err)
	}
}

func TestReleaseBalance_TransactionNotFound(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-release-missing", model.TypeHold).
		Return(nil, nil)

	_, err := svc.ReleaseBalance(ctx, "user-123", 10000, "ref-release-missing")
	if !errors.Is(err, service.ErrReferenceNotFound) {
		t.Fatalf("expected ErrReferenceNotFound, got: %v", err)
	}
}

func TestRefund_InvalidTransaction(t *testing.T) {
	svc, mockRepo, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockRepo.EXPECT().
		GetTransactionByReference(ctx, "ref-refund-invalid", model.TypeDebit).
		Return(nil, nil)

	_, err := svc.Refund(ctx, "user-123", 10000, "ref-refund-invalid")
	if !errors.Is(err, service.ErrInvalidRefund) {
		t.Fatalf("expected ErrInvalidRefund, got: %v", err)
	}
}

func TestHoldBalance_NegativeAmount(t *testing.T) {
	svc, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.HoldBalance(context.Background(), "user-123", -1000, "ref-hold-neg")
	if !errors.Is(err, service.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got: %v", err)
	}
}

func TestDebitBalance_EmptyUserID(t *testing.T) {
	svc, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.DebitBalance(context.Background(), "", 1000, "ref-debit-empty-user")
	if !errors.Is(err, service.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got: %v", err)
	}
}

type concurrentRepo struct {
	mu           sync.Mutex
	walletByUser map[string]*model.Wallet
	txByRefType  map[string]*model.Transaction
}

func newConcurrentRepo() *concurrentRepo {
	return &concurrentRepo{
		walletByUser: map[string]*model.Wallet{},
		txByRefType:  map[string]*model.Transaction{},
	}
}

func (r *concurrentRepo) GetByUserID(ctx context.Context, userID string) (*model.Wallet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, ok := r.walletByUser[userID]
	if !ok {
		return nil, errors.New("wallet not found")
	}
	copy := *w
	return &copy, nil
}

func (r *concurrentRepo) UpdateBalance(ctx context.Context, walletID string, newBalance float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, w := range r.walletByUser {
		if w.ID == walletID {
			w.Balance = newBalance
			return nil
		}
	}
	return errors.New("wallet not found")
}

func (r *concurrentRepo) CreateTransaction(ctx context.Context, tx *model.Transaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tx.ReferenceID != "" {
		r.txByRefType[string(tx.Type)+"::"+tx.ReferenceID] = tx
	}
	return nil
}

func (r *concurrentRepo) GetTransactionByReference(ctx context.Context, referenceID string, typ model.TransactionType) (*model.Transaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, ok := r.txByRefType[string(typ)+"::"+referenceID]
	if !ok {
		return nil, nil
	}
	return tx, nil
}

func TestWallet_Concurrency(t *testing.T) {
	repo := newConcurrentRepo()
	repo.walletByUser["user-123"] = &model.Wallet{
		ID:      "wallet-123",
		UserID:  "user-123",
		Balance: 100,
	}
	svc := service.NewWalletService(repo)

	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var resultMu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ref := fmt.Sprintf("ref-conc-%d", i)
			_, err := svc.DebitBalance(context.Background(), "user-123", 1, ref)

			resultMu.Lock()
			defer resultMu.Unlock()
			if err == nil {
				successCount++
				return
			}
			failCount++
		}(i)
	}
	wg.Wait()

	if successCount != 100 {
		t.Fatalf("expected 100 successful debits, got %d success and %d fail", successCount, failCount)
	}

	finalWallet := repo.walletByUser["user-123"]
	if finalWallet.Balance != 0 {
		t.Fatalf("expected final balance 0, got %v", finalWallet.Balance)
	}
}
