//go:build functional
// +build functional

// Package payment contains cross-service end-to-end functional tests for payment flow.
package payment

import (
	"testing"
)

// TestE2E_PaymentCapture tests the full payment flow:
// 1. Order completed → triggers payment.authorized
// 2. Payment service captures the payment
// 3. Wallet service credits driver
// 4. Settlement service records the settlement
func TestE2E_PaymentCapture(t *testing.T) {
	t.Skip("TODO: Implement cross-service payment capture test")
}

// TestE2E_PaymentFailure tests payment failure handling:
// 1. Order completed → triggers payment
// 2. Payment fails (insufficient balance)
// 3. Order service receives payment.failed event
// 4. User notified
func TestE2E_PaymentFailure(t *testing.T) {
	t.Skip("TODO: Implement cross-service payment failure test")
}

// TestE2E_WalletTopUpAndPay tests wallet top-up then pay flow.
func TestE2E_WalletTopUpAndPay(t *testing.T) {
	t.Skip("TODO: Implement wallet payment e2e test")
}
