// Package unit contains unit tests for payment-service.
// Unit tests do NOT access any database or external service.
package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/payment-service/internal/model"
	"furab-backend/services/payment-service/internal/service"
	"furab-backend/services/payment-service/test/unit/mock"

	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (
	service.PaymentService,
	*mock.MockPaymentRepository,
	*mock.MockPricingClient,
	*mock.MockPromoClient,
	*mock.MockWalletClient,
	*mock.MockSettlementClient,
	*gomock.Controller,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	repo := mock.NewMockPaymentRepository(ctrl)
	pricing := mock.NewMockPricingClient(ctrl)
	promo := mock.NewMockPromoClient(ctrl)
	wallet := mock.NewMockWalletClient(ctrl)
	settlement := mock.NewMockSettlementClient(ctrl)

	svc := service.NewPaymentService(repo, pricing, promo, wallet, settlement)
	return svc, repo, pricing, promo, wallet, settlement, ctrl
}

func validInitiateRequest() *model.InitiatePaymentRequest {
	return &model.InitiatePaymentRequest{
		OrderID:        "ORD-100",
		UserID:         "USR-1",
		PaymentMethod:  "wallet",
		PaymentDetail:  "wallet-topup",
		PromoCode:      "HEMAT20",
		IdempotencyKey: "idem-init-1",
	}
}

func TestInitiatePayment_Success(t *testing.T) {
	svc, repo, pricing, promo, wallet, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	ctx := context.Background()
	req := validInitiateRequest()

	repo.EXPECT().GetPaymentByIdempotencyKey(ctx, req.IdempotencyKey).Return(nil, nil)
	pricing.EXPECT().GetTotalAmount(ctx, req.OrderID).Return(100000.0, nil)
	promo.EXPECT().ApplyPromo(ctx, req.PromoCode, 100000.0).Return(80000.0, 20000.0, nil)

	repo.EXPECT().CreatePayment(ctx, gomock.Any()).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusPending).Return(nil)
	wallet.EXPECT().LockBalance(ctx, req.UserID, 80000.0, gomock.Any()).Return(nil)
	repo.EXPECT().UpdatePaymentStatus(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)

	p, err := svc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == "" || p.TransactionReference == "" || p.TransactionTime.IsZero() {
		t.Fatalf("required output fields should be set")
	}
	if p.PaymentStatus != model.StatusAuthorized {
		t.Fatalf("expected authorized, got %s", p.PaymentStatus)
	}
	if p.MethodID != "wallet" || p.PaymentDetail != "wallet-topup" {
		t.Fatalf("expected payment method and detail to be preserved, got %s / %s", p.MethodID, p.PaymentDetail)
	}
	if p.TransactionReference != "TXN-ORD-100" {
		t.Fatalf("expected transaction reference TXN-ORD-100, got %s", p.TransactionReference)
	}
}

func TestInitiatePayment_InvalidRequest(t *testing.T) {
	svc, _, _, _, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.InitiatePayment(context.Background(), &model.InitiatePaymentRequest{})
	if err == nil {
		t.Fatal("expected error for invalid request")
	}
}

func TestInitiatePayment_MissingPaymentMethod(t *testing.T) {
	svc, _, _, _, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.InitiatePayment(context.Background(), &model.InitiatePaymentRequest{OrderID: "ORD-ERR", UserID: "USR-ERR", PaymentDetail: "detail-only"})
	if err == nil {
		t.Fatal("expected error for missing payment method")
	}
}

func TestInitiatePayment_MissingPaymentDetail(t *testing.T) {
	svc, _, _, _, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	_, err := svc.InitiatePayment(context.Background(), &model.InitiatePaymentRequest{OrderID: "ORD-ERR", UserID: "USR-ERR", PaymentMethod: "wallet"})
	if err == nil {
		t.Fatal("expected error for missing payment detail")
	}
}

func TestInitiatePayment_Idempotency(t *testing.T) {
	ctx := context.Background()
	svc, repo, pricing, _, wallet, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	req := &model.InitiatePaymentRequest{
		OrderID:        "ORD-IDEM",
		UserID:         "USR-IDEM",
		PaymentMethod:  "wallet",
		PaymentDetail:  "idem-detail",
		IdempotencyKey: "unique-idem-key-123",
	}

	// First call expects full chain
	repo.EXPECT().GetPaymentByIdempotencyKey(ctx, req.IdempotencyKey).Return(nil, nil)
	pricing.EXPECT().GetTotalAmount(ctx, req.OrderID).Return(100000.0, nil)

	repo.EXPECT().CreatePayment(ctx, gomock.Any()).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusPending).Return(nil)
	wallet.EXPECT().LockBalance(ctx, req.UserID, 100000.0, gomock.Any()).Return(nil)
	repo.EXPECT().UpdatePaymentStatus(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)

	first, err := svc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("first initiate failed: %v", err)
	}

	// Second call expects just the idempotency key check
	repo.EXPECT().GetPaymentByIdempotencyKey(ctx, req.IdempotencyKey).Return(first, nil)

	second, err := svc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("second initiate failed: %v", err)
	}

	// Verify both calls return same payment
	if first.ID != second.ID {
		t.Fatalf("idempotency key should return same payment: first=%s, second=%s", first.ID, second.ID)
	}

	if second.PaymentStatus != model.StatusAuthorized || second.FinalAmount == 0 {
		t.Fatalf("expected authorized payment with valid amount, got status=%s amount=%f", second.PaymentStatus, second.FinalAmount)
	}

	if second.TransactionReference != first.TransactionReference {
		t.Fatalf("transaction reference should be same for idempotent calls: first=%s, second=%s", first.TransactionReference, second.TransactionReference)
	}
}

func TestCapturePayment_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, pricing, _, wallet, settlement, ctrl := newTestService(t)
	defer ctrl.Finish()

	req := &model.InitiatePaymentRequest{OrderID: "ORD-CAP", UserID: "USR-CAP", PaymentMethod: "wallet", PaymentDetail: "cap-detail", IdempotencyKey: "idem-cap"}

	// Initiate Phase Expectations
	repo.EXPECT().GetPaymentByIdempotencyKey(ctx, req.IdempotencyKey).Return(nil, nil)
	pricing.EXPECT().GetTotalAmount(ctx, req.OrderID).Return(100000.0, nil)
	repo.EXPECT().CreatePayment(ctx, gomock.Any()).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusPending).Return(nil)
	wallet.EXPECT().LockBalance(ctx, req.UserID, 100000.0, gomock.Any()).Return(nil)
	repo.EXPECT().UpdatePaymentStatus(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)

	p, err := svc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("unexpected initiate error: %v", err)
	}

	// Capture Phase Expectations
	repo.EXPECT().GetPaymentByID(ctx, p.ID).Return(p, nil)
	wallet.EXPECT().DeductBalance(ctx, p.UserID, p.FinalAmount, gomock.Any()).Return(nil)
	repo.EXPECT().UpdatePaymentStatus(ctx, p.ID, model.StatusCaptured).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, p.ID, model.StatusCaptured).Return(nil)
	settlement.EXPECT().TriggerSettlement(ctx, p.ID, p.OrderID, p.FinalAmount).Return(nil)

	got, err := svc.CapturePayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected capture error: %v", err)
	}
	if got.PaymentStatus != model.StatusCaptured {
		t.Fatalf("expected captured status, got %s", got.PaymentStatus)
	}
}

func TestCapturePayment_InvalidState(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	repo.EXPECT().GetPaymentByID(ctx, "PAY-1").Return(&model.Payment{ID: "PAY-1", PaymentStatus: model.StatusPending}, nil)

	_, err := svc.CapturePayment(ctx, "PAY-1")
	if err == nil {
		t.Fatal("expected error for non-authorized payment")
	}
}

func TestCancelPayment_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, pricing, _, wallet, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	req := &model.InitiatePaymentRequest{OrderID: "ORD-CAN", UserID: "USR-CAN", PaymentMethod: "wallet", PaymentDetail: "cancel-detail", IdempotencyKey: "idem-can"}

	// Initiate Phase
	repo.EXPECT().GetPaymentByIdempotencyKey(ctx, req.IdempotencyKey).Return(nil, nil)
	pricing.EXPECT().GetTotalAmount(ctx, req.OrderID).Return(100000.0, nil)
	repo.EXPECT().CreatePayment(ctx, gomock.Any()).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusPending).Return(nil)
	wallet.EXPECT().LockBalance(ctx, req.UserID, 100000.0, gomock.Any()).Return(nil)
	repo.EXPECT().UpdatePaymentStatus(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)

	p, err := svc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("unexpected initiate error: %v", err)
	}

	// Cancel Phase
	repo.EXPECT().GetPaymentByID(ctx, p.ID).Return(p, nil)
	wallet.EXPECT().UnlockBalance(ctx, p.UserID, p.FinalAmount, gomock.Any()).Return(nil)
	repo.EXPECT().UpdatePaymentStatus(ctx, p.ID, model.StatusCancelled).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, p.ID, model.StatusCancelled).Return(nil)

	got, err := svc.CancelPayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected cancel error: %v", err)
	}
	if got.PaymentStatus != model.StatusCancelled {
		t.Fatalf("expected cancelled status, got %s", got.PaymentStatus)
	}
}

func TestCancelPayment_InvalidState(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	repo.EXPECT().GetPaymentByID(ctx, "PAY-2").Return(&model.Payment{ID: "PAY-2", PaymentStatus: model.StatusCaptured}, nil)

	_, err := svc.CancelPayment(ctx, "PAY-2")
	if err == nil {
		t.Fatal("expected error when cancelling captured payment")
	}
}

func TestRefundPayment_Success(t *testing.T) {
	ctx := context.Background()
	svc, repo, pricing, _, wallet, settlement, ctrl := newTestService(t)
	defer ctrl.Finish()

	req := &model.InitiatePaymentRequest{OrderID: "ORD-REF", UserID: "USR-REF", PaymentMethod: "wallet", PaymentDetail: "refund-detail", IdempotencyKey: "idem-ref"}

	// Initiate Phase
	repo.EXPECT().GetPaymentByIdempotencyKey(ctx, req.IdempotencyKey).Return(nil, nil)
	pricing.EXPECT().GetTotalAmount(ctx, req.OrderID).Return(100000.0, nil)
	repo.EXPECT().CreatePayment(ctx, gomock.Any()).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusPending).Return(nil)
	wallet.EXPECT().LockBalance(ctx, req.UserID, 100000.0, gomock.Any()).Return(nil)
	repo.EXPECT().UpdatePaymentStatus(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, gomock.Any(), model.StatusAuthorized).Return(nil)

	p, err := svc.InitiatePayment(ctx, req)
	if err != nil {
		t.Fatalf("unexpected initiate error: %v", err)
	}

	// Capture Phase
	repo.EXPECT().GetPaymentByID(ctx, p.ID).Return(p, nil)
	wallet.EXPECT().DeductBalance(ctx, p.UserID, p.FinalAmount, gomock.Any()).Return(nil)
	repo.EXPECT().UpdatePaymentStatus(ctx, p.ID, model.StatusCaptured).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, p.ID, model.StatusCaptured).Return(nil)
	settlement.EXPECT().TriggerSettlement(ctx, p.ID, p.OrderID, p.FinalAmount).Return(nil)

	_, err = svc.CapturePayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected capture error: %v", err)
	}

	// Refund Phase
	repo.EXPECT().GetPaymentByID(ctx, p.ID).Return(p, nil)
	wallet.EXPECT().CreditBalance(ctx, p.UserID, p.FinalAmount, gomock.Any()).Return(nil)
	repo.EXPECT().UpdatePaymentStatus(ctx, p.ID, model.StatusRefunded).Return(nil)
	repo.EXPECT().CreatePaymentLog(ctx, p.ID, model.StatusRefunded).Return(nil)

	refunded, err := svc.RefundPayment(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected refund error: %v", err)
	}
	if refunded.PaymentStatus != model.StatusRefunded {
		t.Fatalf("expected refunded, got %s", refunded.PaymentStatus)
	}
}

func TestRefundPayment_InvalidState(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, _, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	repo.EXPECT().GetPaymentByID(ctx, "PAY-REF-ERR").Return(&model.Payment{ID: "PAY-REF-ERR", PaymentStatus: model.StatusAuthorized}, nil)

	_, err := svc.RefundPayment(ctx, "PAY-REF-ERR")
	if err == nil {
		t.Fatal("expected error when refunding non-captured payment")
	}
}

func TestGetPayment_Success(t *testing.T) {
	svc, repo, _, _, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	repo.EXPECT().GetPaymentByID(context.Background(), "PAY-GET").Return(&model.Payment{ID: "PAY-GET", PaymentStatus: model.StatusAuthorized}, nil)

	payment, err := svc.GetPayment(context.Background(), "PAY-GET")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if payment == nil || payment.ID != "PAY-GET" {
		t.Fatalf("expected payment PAY-GET, got %v", payment)
	}
}

func TestGetPayment_NotFound(t *testing.T) {
	svc, repo, _, _, _, _, ctrl := newTestService(t)
	defer ctrl.Finish()

	repo.EXPECT().GetPaymentByID(context.Background(), "missing").Return(nil, nil)

	_, err := svc.GetPayment(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing payment")
	}
	if !errors.Is(err, service.ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got %v", err)
	}
}
