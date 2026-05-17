package mock

import (
	"context"

	"furab-backend/services/food-order-service/internal/model"

	"go.uber.org/mock/gomock"
)

type MockFoodOrderRepository struct {
	ctrl     *gomock.Controller
	recorder *MockFoodOrderRepositoryMockRecorder
}

type MockFoodOrderRepositoryMockRecorder struct {
	mock *MockFoodOrderRepository
}

func NewMockFoodOrderRepository(ctrl *gomock.Controller) *MockFoodOrderRepository {
	mock := &MockFoodOrderRepository{ctrl: ctrl}
	mock.recorder = &MockFoodOrderRepositoryMockRecorder{mock}
	return mock
}

func (m *MockFoodOrderRepository) EXPECT() *MockFoodOrderRepositoryMockRecorder {
	return m.recorder
}

func (m *MockFoodOrderRepository) Create(ctx context.Context, order *model.FoodOrder) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, order)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockFoodOrderRepositoryMockRecorder) Create(ctx, order interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Create", ctx, order)
}

func (m *MockFoodOrderRepository) GetByID(ctx context.Context, id string) (*model.FoodOrder, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*model.FoodOrder)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFoodOrderRepositoryMockRecorder) GetByID(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetByID", ctx, id)
}

func (m *MockFoodOrderRepository) Update(ctx context.Context, order *model.FoodOrder) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, order)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockFoodOrderRepositoryMockRecorder) Update(ctx, order interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Update", ctx, order)
}

func (m *MockFoodOrderRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.FoodOrder, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByUserID", ctx, userID, limit, offset)
	ret0, _ := ret[0].([]*model.FoodOrder)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFoodOrderRepositoryMockRecorder) GetByUserID(ctx, userID, limit, offset interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetByUserID", ctx, userID, limit, offset)
}

func (m *MockFoodOrderRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CountByUserID", ctx, userID)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockFoodOrderRepositoryMockRecorder) CountByUserID(ctx, userID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "CountByUserID", ctx, userID)
}
