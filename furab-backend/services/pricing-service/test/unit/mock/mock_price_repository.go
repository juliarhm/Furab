// Package mock provides mock implementations for pricing-service testing.
package mock

import (
	"context"

	"furab-backend/services/pricing-service/internal/model"

	"go.uber.org/mock/gomock"
)

// MockPriceRepository is a mock implementation of repository.PriceRepository.
type MockPriceRepository struct {
	ctrl     *gomock.Controller
	recorder *MockPriceRepositoryMockRecorder
}

// MockPriceRepositoryMockRecorder is the mock recorder for MockPriceRepository.
type MockPriceRepositoryMockRecorder struct {
	mock *MockPriceRepository
}

// NewMockPriceRepository creates a new mock instance.
func NewMockPriceRepository(ctrl *gomock.Controller) *MockPriceRepository {
	mock := &MockPriceRepository{ctrl: ctrl}
	mock.recorder = &MockPriceRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPriceRepository) EXPECT() *MockPriceRepositoryMockRecorder {
	return m.recorder
}

// GetPricingRules mocks the GetPricingRules method.
func (m *MockPriceRepository) GetPricingRules(ctx context.Context) ([]model.PriceRule, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPricingRules", ctx)
	ret0, _ := ret[0].([]model.PriceRule)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPricingRules indicates an expected call of GetPricingRules.
func (mr *MockPriceRepositoryMockRecorder) GetPricingRules(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetPricingRules", ctx)
}

// GetPricingRuleByType mocks the GetPricingRuleByType method.
func (m *MockPriceRepository) GetPricingRuleByType(ctx context.Context, ruleType string) (*model.PriceRule, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPricingRuleByType", ctx, ruleType)
	ret0, _ := ret[0].(*model.PriceRule)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPricingRuleByType indicates an expected call of GetPricingRuleByType.
func (mr *MockPriceRepositoryMockRecorder) GetPricingRuleByType(ctx, ruleType interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCall(mr.mock, "GetPricingRuleByType", ctx, ruleType)
}
