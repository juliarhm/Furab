// Package model defines the domain models for payment-service.
package model

import "time"

type PaymentStatus string

const (
	StatusPending    PaymentStatus = "pending"
	StatusAuthorized PaymentStatus = "authorized"
	StatusCaptured   PaymentStatus = "captured"
	StatusRefunded   PaymentStatus = "refunded"
	StatusFailed     PaymentStatus = "failed"
	StatusCancelled  PaymentStatus = "cancelled"
)

// Payment represents the Payment model in payment-service.
type Payment struct {
	ID                   string        `json:"payment_id"`
	OrderID              string        `json:"order_id"`
	UserID               string        `json:"user_id"`
	Amount               float64       `json:"amount"`
	FinalAmount          float64       `json:"final_amount"`
	MethodID             string        `json:"payment_method"`
	PaymentDetail        string        `json:"payment_detail"`
	PaymentStatus        PaymentStatus `json:"payment_status"`
	TransactionReference string        `json:"transaction_reference"`
	IdempotencyKey       string        `json:"idempotency_key"`
	TransactionTime      time.Time     `json:"transaction_time"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
}

// PaymentMethod represents the PaymentMethod model in payment-service.
type PaymentMethod struct {
	ID         string    `json:"method_id"`
	MethodName string    `json:"method_name"`
	Provider   string    `json:"provider"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type PaymentLog struct {
	ID        string        `json:"log_id"`
	PaymentID string        `json:"payment_id"`
	Status    PaymentStatus `json:"status"`
	Timestamp time.Time     `json:"timestamp"`
}

type InitiatePaymentRequest struct {
	OrderID        string  `json:"order_id"`
	UserID         string  `json:"user_id"`
	PaymentMethod  string  `json:"payment_method"`
	PaymentDetail  string  `json:"payment_detail"`
	PromoCode      string  `json:"promo_code"`
	Amount         float64 `json:"amount"`
	IdempotencyKey string  `json:"idempotency_key"`
}
