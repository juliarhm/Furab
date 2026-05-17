package mock

import (
	"context"

	"furab-backend/shared/event"

	"go.uber.org/mock/gomock"
)

type MockEventPublisher struct {
	ctrl     *gomock.Controller
	recorder *MockEventPublisherMockRecorder
}

type MockEventPublisherMockRecorder struct {
	mock *MockEventPublisher
}

func NewMockEventPublisher(ctrl *gomock.Controller) *MockEventPublisher {
	mock := &MockEventPublisher{ctrl: ctrl}
	mock.recorder = &MockEventPublisherMockRecorder{mock}
	return mock
}

func (m *MockEventPublisher) EXPECT() *MockEventPublisherMockRecorder {
	return m.recorder
}

func (m *MockEventPublisher) Publish(ctx context.Context, topic string, evt *event.Event) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Publish", ctx, topic, evt)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockEventPublisherMockRecorder) Publish(ctx, topic, evt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Publish", ctx, topic, evt)
}

func (m *MockEventPublisher) Close() error {
	return nil
}
