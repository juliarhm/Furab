// Package mock provides mock implementations for testing.
package mock

import (
	"context"

	"furab-backend/services/settlement-service/internal/model"

	"go.uber.org/mock/gomock"
)

// MockSettlementRepository is a mock implementation of repository.SettlementRepository.
type MockSettlementRepository struct {
	ctrl     *gomock.Controller
	recorder *MockSettlementRepositoryMockRecorder
}

// MockSettlementRepositoryMockRecorder is the mock recorder for MockSettlementRepository.
type MockSettlementRepositoryMockRecorder struct {
	mock *MockSettlementRepository
}

// NewMockSettlementRepository creates a new mock instance.
func NewMockSettlementRepository(ctrl *gomock.Controller) *MockSettlementRepository {
	mock := &MockSettlementRepository{ctrl: ctrl}
	mock.recorder = &MockSettlementRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSettlementRepository) EXPECT() *MockSettlementRepositoryMockRecorder {
	return m.recorder
}

// CreateSettlement mocks the CreateSettlement method.
func (m *MockSettlementRepository) CreateSettlement(ctx context.Context, s *model.Settlement) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSettlement", ctx, s)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateSettlement indicates an expected call of CreateSettlement.
func (mr *MockSettlementRepositoryMockRecorder) CreateSettlement(ctx, s interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "CreateSettlement", ctx, s)
}

// GetSettlementByPaymentID mocks the GetSettlementByPaymentID method.
func (m *MockSettlementRepository) GetSettlementByPaymentID(ctx context.Context, paymentID string) (*model.Settlement, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSettlementByPaymentID", ctx, paymentID)
	ret0, _ := ret[0].(*model.Settlement)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSettlementByPaymentID indicates an expected call of GetSettlementByPaymentID.
func (mr *MockSettlementRepositoryMockRecorder) GetSettlementByPaymentID(ctx, paymentID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetSettlementByPaymentID", ctx, paymentID)
}

// UpdateSettlementStatus mocks the UpdateSettlementStatus method.
func (m *MockSettlementRepository) UpdateSettlementStatus(ctx context.Context, settlementID string, status model.SettlementStatus) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateSettlementStatus", ctx, settlementID, status)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateSettlementStatus indicates an expected call of UpdateSettlementStatus.
func (mr *MockSettlementRepositoryMockRecorder) UpdateSettlementStatus(ctx, settlementID, status interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "UpdateSettlementStatus", ctx, settlementID, status)
}
