// Package model defines the domain models for settlement-service.
package model

import "time"

type SettlementStatus string

const (
	StatusPending SettlementStatus = "pending"
	StatusSuccess SettlementStatus = "success"
	StatusFailed  SettlementStatus = "failed"
)

// Settlement represents the Settlement model in settlement-service.
type Settlement struct {
	ID             string           `json:"settlement_id"`
	PaymentID      string           `json:"payment_id"`
	OrderID        string           `json:"order_id"`
	TotalAmount    float64          `json:"total_amount"`
	DriverAmount   float64          `json:"driver_amount"`
	MerchantAmount float64          `json:"merchant_amount"`
	PlatformFee    float64          `json:"platform_fee"`
	Status         SettlementStatus `json:"status"`
	IdempotencyKey string           `json:"idempotency_key"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type ProcessSettlementRequest struct {
	PaymentID   string  `json:"payment_id"`
	OrderID     string  `json:"order_id"`
	TotalAmount float64 `json:"total_amount"`
}

type ProcessSettlementResponse struct {
	Status         string  `json:"status"` // SUCCESS / FAILED
	DriverAmount   float64 `json:"driver_amount"`
	MerchantAmount float64 `json:"merchant_amount"`
	PlatformFee    float64 `json:"platform_fee"`
}
