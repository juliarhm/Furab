// Package mock provides mock implementations for testing.
// Mock for event.Publisher interface.
package mock

import (
	"context"

	"furab-backend/shared/event"

	"go.uber.org/mock/gomock"
)

// MockEventPublisher is a mock implementation of event.Publisher.
type MockEventPublisher struct {
	ctrl     *gomock.Controller
	recorder *MockEventPublisherMockRecorder
}

// MockEventPublisherMockRecorder is the mock recorder for MockEventPublisher.
type MockEventPublisherMockRecorder struct {
	mock *MockEventPublisher
}

// NewMockEventPublisher creates a new mock instance.
func NewMockEventPublisher(ctrl *gomock.Controller) *MockEventPublisher {
	mock := &MockEventPublisher{ctrl: ctrl}
	mock.recorder = &MockEventPublisherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventPublisher) EXPECT() *MockEventPublisherMockRecorder {
	return m.recorder
}

// Publish mocks the Publish method.
func (m *MockEventPublisher) Publish(ctx context.Context, topic string, evt *event.Event) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Publish", ctx, topic, evt)
	ret0, _ := ret[0].(error)
	return ret0
}

// Publish indicates an expected call of Publish.
func (mr *MockEventPublisherMockRecorder) Publish(ctx, topic, evt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Publish", ctx, topic, evt)
}

// Close mocks the Close method.
func (m *MockEventPublisher) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockEventPublisherMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "Close")
}
