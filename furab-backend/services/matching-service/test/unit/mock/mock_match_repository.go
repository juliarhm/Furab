package mock

import (
	"context"

	"furab-backend/services/matching-service/internal/model"

	"go.uber.org/mock/gomock"
)

type MockMatchRepository struct {
	ctrl     *gomock.Controller
	recorder *MockMatchRepositoryMockRecorder
}

type MockMatchRepositoryMockRecorder struct {
	mock *MockMatchRepository
}

func NewMockMatchRepository(ctrl *gomock.Controller) *MockMatchRepository {
	mock := &MockMatchRepository{ctrl: ctrl}
	mock.recorder = &MockMatchRepositoryMockRecorder{mock}
	return mock
}

func (m *MockMatchRepository) EXPECT() *MockMatchRepositoryMockRecorder {
	return m.recorder
}

func (m *MockMatchRepository) Create(ctx context.Context, match *model.MatchRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, match)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockMatchRepositoryMockRecorder) Create(ctx, match interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Create", ctx, match)
}

func (m *MockMatchRepository) GetByID(ctx context.Context, id string) (*model.MatchRequest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*model.MatchRequest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockMatchRepositoryMockRecorder) GetByID(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetByID", ctx, id)
}

func (m *MockMatchRepository) Update(ctx context.Context, match *model.MatchRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, match)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockMatchRepositoryMockRecorder) Update(ctx, match interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Update", ctx, match)
}

func (m *MockMatchRepository) GetByOrderID(ctx context.Context, orderID string) (*model.MatchRequest, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByOrderID", ctx, orderID)
	ret0, _ := ret[0].(*model.MatchRequest)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockMatchRepositoryMockRecorder) GetByOrderID(ctx, orderID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetByOrderID", ctx, orderID)
}
