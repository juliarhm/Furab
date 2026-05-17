// Package unit contains unit tests for settlement-service.
// Unit tests do NOT access any database or external service.
// All dependencies are mocked or faked in memory.
package unit

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"furab-backend/services/settlement-service/internal/model"
	"furab-backend/services/settlement-service/internal/service"
	"furab-backend/services/settlement-service/test/unit/mock"

	"go.uber.org/mock/gomock"
)

type fakeWalletClient struct {
	fail       bool
	creditCall int
	failOnRef  map[string]int
	appliedRef map[string]bool
}

func (f *fakeWalletClient) CreditBalance(ctx context.Context, walletID string, amount float64, referenceID string) error {
	if f.appliedRef == nil {
		f.appliedRef = map[string]bool{}
	}

	if f.appliedRef[referenceID] {
		return nil
	}

	f.creditCall++
	if f.fail {
		return errors.New("wallet service error")
	}
	if f.failOnRef != nil {
		if remainingFail := f.failOnRef[referenceID]; remainingFail > 0 {
			f.failOnRef[referenceID] = remainingFail - 1
			return errors.New("wallet service partial failure")
		}
	}
	f.appliedRef[referenceID] = true
	return nil
}

type fakeDriverClient struct {
	walletID string
	err      error
}

func (f *fakeDriverClient) GetDriverWalletIDByOrderID(ctx context.Context, orderID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.walletID, nil
}

type fakeMerchantClient struct {
	walletID string
	err      error
}

func (f *fakeMerchantClient) GetMerchantWalletIDByOrderID(ctx context.Context, orderID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.walletID, nil
}

func newTestService(t *testing.T) (service.SettlementService, *mock.MockSettlementRepository, *fakeWalletClient, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	mockRepo := mock.NewMockSettlementRepository(ctrl)
	fakeWallet := &fakeWalletClient{}
	svc := service.NewSettlementService(
		mockRepo,
		fakeWallet,
		&fakeDriverClient{walletID: "wallet-driver-1"},
		&fakeMerchantClient{walletID: "wallet-merchant-1"},
	)
	return svc, mockRepo, fakeWallet, ctrl
}

func TestProcessSettlement_Success(t *testing.T) {
	svc, mockRepo, walletCli, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := &model.ProcessSettlementRequest{
		PaymentID:   "pay-101",
		OrderID:     "order-500",
		TotalAmount: 100000,
	}

	mockRepo.EXPECT().
		GetSettlementByPaymentID(ctx, req.PaymentID).
		Return(nil, nil)
	mockRepo.EXPECT().
		CreateSettlement(ctx, gomock.Any()).
		Return(nil)
	mockRepo.EXPECT().
		UpdateSettlementStatus(ctx, gomock.Any(), model.StatusSuccess).
		Return(nil)

	res, err := svc.ProcessSettlement(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got: %s", res.Status)
	}
	if res.DriverAmount != 80000 || res.MerchantAmount != 15000 || res.PlatformFee != 5000 {
		t.Fatalf("unexpected distribution split: %+v", res)
	}
	if walletCli.creditCall != 2 {
		t.Fatalf("expected 2 wallet credit calls, got %d", walletCli.creditCall)
	}
}

func TestProcessSettlement_CalculationCheck(t *testing.T) {
	svc, mockRepo, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := &model.ProcessSettlementRequest{
		PaymentID:   "pay-calc-1",
		OrderID:     "order-calc-1",
		TotalAmount: 99999,
	}

	mockRepo.EXPECT().
		GetSettlementByPaymentID(ctx, req.PaymentID).
		Return(nil, nil)
	mockRepo.EXPECT().
		CreateSettlement(ctx, gomock.Any()).
		Return(nil)
	mockRepo.EXPECT().
		UpdateSettlementStatus(ctx, gomock.Any(), model.StatusSuccess).
		Return(nil)

	res, err := svc.ProcessSettlement(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.DriverAmount+res.MerchantAmount+res.PlatformFee != req.TotalAmount {
		t.Fatalf(
			"total mismatch, total=%v driver=%v merchant=%v platform=%v",
			req.TotalAmount,
			res.DriverAmount,
			res.MerchantAmount,
			res.PlatformFee,
		)
	}
}

func TestProcessSettlement_IdempotentReplay_Success(t *testing.T) {
	svc, mockRepo, walletCli, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := &model.ProcessSettlementRequest{
		PaymentID:   "pay-existing",
		OrderID:     "order-900",
		TotalAmount: 300000,
	}

	mockRepo.EXPECT().
		GetSettlementByPaymentID(ctx, req.PaymentID).
		Return(&model.Settlement{
			ID:             "set-existing",
			PaymentID:      req.PaymentID,
			OrderID:        req.OrderID,
			TotalAmount:    req.TotalAmount,
			DriverAmount:   240000,
			MerchantAmount: 45000,
			PlatformFee:    15000,
			Status:         model.StatusSuccess,
		}, nil)

	res, err := svc.ProcessSettlement(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got: %s", res.Status)
	}
	if walletCli.creditCall != 0 {
		t.Fatalf("expected no wallet credit call on idempotent replay, got %d", walletCli.creditCall)
	}
}

func TestProcessSettlement_WalletCreditFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSettlementRepository(ctrl)
	walletCli := &fakeWalletClient{fail: true}
	svc := service.NewSettlementService(
		mockRepo,
		walletCli,
		&fakeDriverClient{walletID: "wallet-driver-1"},
		&fakeMerchantClient{walletID: "wallet-merchant-1"},
	)

	ctx := context.Background()
	req := &model.ProcessSettlementRequest{
		PaymentID:   "pay-102",
		OrderID:     "order-501",
		TotalAmount: 50000,
	}

	mockRepo.EXPECT().
		GetSettlementByPaymentID(ctx, req.PaymentID).
		Return(nil, nil)
	mockRepo.EXPECT().
		CreateSettlement(ctx, gomock.Any()).
		Return(nil)
	mockRepo.EXPECT().
		UpdateSettlementStatus(ctx, gomock.Any(), model.StatusFailed).
		Return(nil)

	res, err := svc.ProcessSettlement(ctx, req)
	if err == nil {
		t.Fatal("expected error when wallet credit fails")
	}
	if res == nil || res.Status != "FAILED" {
		t.Fatalf("expected FAILED response, got %+v", res)
	}
}

func TestProcessSettlement_InvalidRequest(t *testing.T) {
	svc, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.ProcessSettlement(context.Background(), &model.ProcessSettlementRequest{
		PaymentID:   "",
		OrderID:     "order-1",
		TotalAmount: 1000,
	})
	if !errors.Is(err, service.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestProcessSettlement_DriverNotActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSettlementRepository(ctrl)
	walletCli := &fakeWalletClient{}
	svc := service.NewSettlementService(
		mockRepo,
		walletCli,
		&fakeDriverClient{walletID: ""},
		&fakeMerchantClient{walletID: "wallet-merchant-1"},
	)

	ctx := context.Background()
	req := &model.ProcessSettlementRequest{
		PaymentID:   "pay-300",
		OrderID:     "order-300",
		TotalAmount: 10000,
	}

	mockRepo.EXPECT().
		GetSettlementByPaymentID(ctx, req.PaymentID).
		Return(nil, nil)
	mockRepo.EXPECT().
		CreateSettlement(ctx, gomock.Any()).
		Return(nil)
	mockRepo.EXPECT().
		UpdateSettlementStatus(ctx, gomock.Any(), model.StatusFailed).
		Return(nil)

	_, err := svc.ProcessSettlement(ctx, req)
	if !errors.Is(err, service.ErrRecipientNotActive) {
		t.Fatalf("expected ErrRecipientNotActive, got %v", err)
	}
}

func TestProcessSettlement_MerchantNotActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSettlementRepository(ctrl)
	walletCli := &fakeWalletClient{}
	svc := service.NewSettlementService(
		mockRepo,
		walletCli,
		&fakeDriverClient{walletID: "wallet-driver-1"},
		&fakeMerchantClient{walletID: ""},
	)

	ctx := context.Background()
	req := &model.ProcessSettlementRequest{
		PaymentID:   "pay-merchant-inactive",
		OrderID:     "order-merchant-inactive",
		TotalAmount: 12000,
	}

	mockRepo.EXPECT().
		GetSettlementByPaymentID(ctx, req.PaymentID).
		Return(nil, nil)
	mockRepo.EXPECT().
		CreateSettlement(ctx, gomock.Any()).
		Return(nil)
	mockRepo.EXPECT().
		UpdateSettlementStatus(ctx, gomock.Any(), model.StatusFailed).
		Return(nil)

	_, err := svc.ProcessSettlement(ctx, req)
	if !errors.Is(err, service.ErrRecipientNotActive) {
		t.Fatalf("expected ErrRecipientNotActive, got %v", err)
	}
}

func TestProcessSettlement_PartialWalletCreditSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock.NewMockSettlementRepository(ctrl)
	walletCli := &fakeWalletClient{
		failOnRef: map[string]int{
			"SETTLE-MER-pay-partial-1": 1,
		},
	}
	svc := service.NewSettlementService(
		mockRepo,
		walletCli,
		&fakeDriverClient{walletID: "wallet-driver-1"},
		&fakeMerchantClient{walletID: "wallet-merchant-1"},
	)

	ctx := context.Background()
	req := &model.ProcessSettlementRequest{
		PaymentID:   "pay-partial-1",
		OrderID:     "order-partial-1",
		TotalAmount: 100000,
	}
	failedSettlement := &model.Settlement{
		ID:             "set-partial-1",
		PaymentID:      req.PaymentID,
		OrderID:        req.OrderID,
		TotalAmount:    req.TotalAmount,
		DriverAmount:   80000,
		MerchantAmount: 15000,
		PlatformFee:    5000,
		Status:         model.StatusFailed,
		IdempotencyKey: req.PaymentID,
	}

	gomock.InOrder(
		mockRepo.EXPECT().GetSettlementByPaymentID(ctx, req.PaymentID).Return(nil, nil),
		mockRepo.EXPECT().CreateSettlement(ctx, gomock.Any()).Return(nil),
		mockRepo.EXPECT().UpdateSettlementStatus(ctx, gomock.Any(), model.StatusFailed).Return(nil),
		mockRepo.EXPECT().GetSettlementByPaymentID(ctx, req.PaymentID).Return(failedSettlement, nil),
		mockRepo.EXPECT().UpdateSettlementStatus(ctx, failedSettlement.ID, model.StatusSuccess).Return(nil),
	)

	_, err := svc.ProcessSettlement(ctx, req)
	if err == nil {
		t.Fatal("expected first attempt to fail on merchant credit")
	}

	res, err := svc.ProcessSettlement(ctx, req)
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if res.Status != "SUCCESS" {
		t.Fatalf("expected SUCCESS after retry, got %s", res.Status)
	}

	if !walletCli.appliedRef["SETTLE-DRV-"+req.PaymentID] {
		t.Fatal("expected driver credit to be applied")
	}
	if !walletCli.appliedRef["SETTLE-MER-"+req.PaymentID] {
		t.Fatal("expected merchant credit to be applied after retry")
	}
	driverApplyCount := 0
	for ref := range walletCli.appliedRef {
		if ref == fmt.Sprintf("SETTLE-DRV-%s", req.PaymentID) {
			driverApplyCount++
		}
	}
	if driverApplyCount != 1 {
		t.Fatalf("expected driver applied exactly once, got %d", driverApplyCount)
	}
}
