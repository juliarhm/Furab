// Package mock provides mock implementations for cart-service dependencies.
package mock

import (
	"context"

	"furab-backend/services/cart-service/internal/model"

	"go.uber.org/mock/gomock"
)

// MockCartRepository is a mock of CartRepository interface.
type MockCartRepository struct {
	ctrl     *gomock.Controller
	recorder *MockCartRepositoryMockRecorder
}

// MockCartRepositoryMockRecorder is the mock recorder for MockCartRepository.
type MockCartRepositoryMockRecorder struct {
	mock *MockCartRepository
}

// NewMockCartRepository creates a new mock instance.
func NewMockCartRepository(ctrl *gomock.Controller) *MockCartRepository {
	mock := &MockCartRepository{ctrl: ctrl}
	mock.recorder = &MockCartRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCartRepository) EXPECT() *MockCartRepositoryMockRecorder {
	return m.recorder
}

// GetByUserID mocks CartRepository.GetByUserID.
func (m *MockCartRepository) GetByUserID(ctx context.Context, userID string) (*model.Cart, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByUserID", ctx, userID)
	ret0, _ := ret[0].(*model.Cart)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByUserID indicates an expected call of GetByUserID.
func (mr *MockCartRepositoryMockRecorder) GetByUserID(ctx, userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetByUserID", ctx, userID)
}

// Save mocks CartRepository.Save.
func (m *MockCartRepository) Save(ctx context.Context, cart *model.Cart) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Save", ctx, cart)
	ret0, _ := ret[0].(error)
	return ret0
}

// Save indicates an expected call of Save.
func (mr *MockCartRepositoryMockRecorder) Save(ctx, cart interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Save", ctx, cart)
}

// Delete mocks CartRepository.Delete.
func (m *MockCartRepository) Delete(ctx context.Context, userID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, userID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockCartRepositoryMockRecorder) Delete(ctx, userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Delete", ctx, userID)
}
