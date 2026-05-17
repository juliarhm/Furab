// Package mock provides mock implementations for testing.
// Mock for repository.WalletRepository interface using gomock.
package mock

import (
	"context"

	"furab-backend/services/wallet-service/internal/model"

	"go.uber.org/mock/gomock"
)

// MockWalletRepository is a mock implementation of repository.WalletRepository.
type MockWalletRepository struct {
	ctrl     *gomock.Controller
	recorder *MockWalletRepositoryMockRecorder
}

// MockWalletRepositoryMockRecorder is the mock recorder for MockWalletRepository.
type MockWalletRepositoryMockRecorder struct {
	mock *MockWalletRepository
}

// NewMockWalletRepository creates a new mock instance.
func NewMockWalletRepository(ctrl *gomock.Controller) *MockWalletRepository {
	mock := &MockWalletRepository{ctrl: ctrl}
	mock.recorder = &MockWalletRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWalletRepository) EXPECT() *MockWalletRepositoryMockRecorder {
	return m.recorder
}

// GetByUserID mocks the GetByUserID method.
func (m *MockWalletRepository) GetByUserID(ctx context.Context, userID string) (*model.Wallet, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByUserID", ctx, userID)
	ret0, _ := ret[0].(*model.Wallet)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByUserID indicates an expected call of GetByUserID.
func (mr *MockWalletRepositoryMockRecorder) GetByUserID(ctx, userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetByUserID", ctx, userID)
}

// UpdateBalance mocks the UpdateBalance method.
func (m *MockWalletRepository) UpdateBalance(ctx context.Context, walletID string, newBalance float64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBalance", ctx, walletID, newBalance)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateBalance indicates an expected call of UpdateBalance.
func (mr *MockWalletRepositoryMockRecorder) UpdateBalance(ctx, walletID, newBalance interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "UpdateBalance", ctx, walletID, newBalance)
}

// CreateTransaction mocks the CreateTransaction method.
func (m *MockWalletRepository) CreateTransaction(ctx context.Context, tx *model.Transaction) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTransaction", ctx, tx)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateTransaction indicates an expected call of CreateTransaction.
func (mr *MockWalletRepositoryMockRecorder) CreateTransaction(ctx, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "CreateTransaction", ctx, tx)
}

// GetTransactionByReference mocks the GetTransactionByReference method.
func (m *MockWalletRepository) GetTransactionByReference(ctx context.Context, referenceID string, typ model.TransactionType) (*model.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransactionByReference", ctx, referenceID, typ)
	ret0, _ := ret[0].(*model.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTransactionByReference indicates an expected call of GetTransactionByReference.
func (mr *MockWalletRepositoryMockRecorder) GetTransactionByReference(ctx, referenceID, typ interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetTransactionByReference", ctx, referenceID, typ)
}
