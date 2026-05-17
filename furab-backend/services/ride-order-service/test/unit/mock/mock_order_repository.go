// Package mock provides mock implementations for testing.
// Mock for repository.OrderRepository interface using gomock.
package mock

import (
	"context"

	"furab-backend/services/ride-order-service/internal/model"

	"go.uber.org/mock/gomock"
)

// MockOrderRepository is a mock implementation of repository.OrderRepository.
type MockOrderRepository struct {
	ctrl     *gomock.Controller
	recorder *MockOrderRepositoryMockRecorder
}

// MockOrderRepositoryMockRecorder is the mock recorder for MockOrderRepository.
type MockOrderRepositoryMockRecorder struct {
	mock *MockOrderRepository
}

// NewMockOrderRepository creates a new mock instance.
func NewMockOrderRepository(ctrl *gomock.Controller) *MockOrderRepository {
	mock := &MockOrderRepository{ctrl: ctrl}
	mock.recorder = &MockOrderRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOrderRepository) EXPECT() *MockOrderRepositoryMockRecorder {
	return m.recorder
}

// Create mocks the Create method.
func (m *MockOrderRepository) Create(ctx context.Context, order *model.RideOrder) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, order)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockOrderRepositoryMockRecorder) Create(ctx, order interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Create", ctx, order)
}

// GetByID mocks the GetByID method.
func (m *MockOrderRepository) GetByID(ctx context.Context, id string) (*model.RideOrder, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*model.RideOrder)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockOrderRepositoryMockRecorder) GetByID(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetByID", ctx, id)
}

// Update mocks the Update method.
func (m *MockOrderRepository) Update(ctx context.Context, order *model.RideOrder) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, order)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockOrderRepositoryMockRecorder) Update(ctx, order interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Update", ctx, order)
}

// UpdateStatus mocks the UpdateStatus method.
func (m *MockOrderRepository) UpdateStatus(ctx context.Context, id string, status model.RideStatus) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStatus", ctx, id, status)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateStatus indicates an expected call of UpdateStatus.
func (mr *MockOrderRepositoryMockRecorder) UpdateStatus(ctx, id, status interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "UpdateStatus", ctx, id, status)
}

// AssignDriver mocks the AssignDriver method.
func (m *MockOrderRepository) AssignDriver(ctx context.Context, orderID, driverID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AssignDriver", ctx, orderID, driverID)
	ret0, _ := ret[0].(error)
	return ret0
}

// AssignDriver indicates an expected call of AssignDriver.
func (mr *MockOrderRepositoryMockRecorder) AssignDriver(ctx, orderID, driverID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "AssignDriver", ctx, orderID, driverID)
}

// GetByUserID mocks the GetByUserID method.
func (m *MockOrderRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.RideOrder, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByUserID", ctx, userID, limit, offset)
	ret0, _ := ret[0].([]*model.RideOrder)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByUserID indicates an expected call of GetByUserID.
func (mr *MockOrderRepositoryMockRecorder) GetByUserID(ctx, userID, limit, offset interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetByUserID", ctx, userID, limit, offset)
}

// GetByDriverID mocks the GetByDriverID method.
func (m *MockOrderRepository) GetByDriverID(ctx context.Context, driverID string, limit, offset int) ([]*model.RideOrder, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByDriverID", ctx, driverID, limit, offset)
	ret0, _ := ret[0].([]*model.RideOrder)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByDriverID indicates an expected call of GetByDriverID.
func (mr *MockOrderRepositoryMockRecorder) GetByDriverID(ctx, driverID, limit, offset interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetByDriverID", ctx, driverID, limit, offset)
}

// CountByUserID mocks the CountByUserID method.
func (m *MockOrderRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CountByUserID", ctx, userID)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CountByUserID indicates an expected call of CountByUserID.
func (mr *MockOrderRepositoryMockRecorder) CountByUserID(ctx, userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "CountByUserID", ctx, userID)
}

// Delete mocks the Delete method.
func (m *MockOrderRepository) Delete(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockOrderRepositoryMockRecorder) Delete(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Delete", ctx, id)
}
