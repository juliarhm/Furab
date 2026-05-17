// Package model defines the domain models for wallet-service.
package model

import "time"

type TransactionType string
type TransactionStatus string

const (
	TypeHold    TransactionType = "hold"
	TypeRelease TransactionType = "release"
	TypeDebit   TransactionType = "debit"
	TypeCredit  TransactionType = "credit"
	TypeRefund  TransactionType = "refund"
)

const (
	StatusPending   TransactionStatus = "pending"
	StatusSuccess   TransactionStatus = "success"
	StatusFailed    TransactionStatus = "failed"
	StatusCancelled TransactionStatus = "cancelled"
)

// Wallet represents user wallet with balance.
type Wallet struct {
	ID        string    `json:"wallet_id"`
	UserID    string    `json:"user_id"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WalletTransaction represents a wallet transaction entry (audit trail).
type WalletTransaction struct {
	ID          string            `json:"transaction_id"`
	WalletID    string            `json:"wallet_id"`
	ReferenceID string            `json:"reference_id"`
	Amount      float64           `json:"amount"`
	Type        TransactionType   `json:"type"`
	Status      TransactionStatus `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Transaction is the internal transaction model used by service/repository layers.
type Transaction struct {
	ID             string
	WalletID       string
	ReferenceID    string
	Amount         float64
	Type           TransactionType
	Status         TransactionStatus
	CurrentBalance float64
	CreatedAt      time.Time
}

// WalletResult is a compact service response for balance-changing operations.
type WalletResult struct {
	Status         TransactionStatus
	CurrentBalance float64
	TransactionID  string
}

// LockBalanceRequest represents request to hold saldo (pre-authorization).
type LockBalanceRequest struct {
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

// UnlockBalanceRequest represents request to release held saldo.
type UnlockBalanceRequest struct {
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

// DeductBalanceRequest represents request to debit saldo (capture).
type DeductBalanceRequest struct {
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

// CreditBalanceRequest represents request to kredit saldo.
type CreditBalanceRequest struct {
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

// RefundBalanceRequest represents request to refund saldo.
type RefundBalanceRequest struct {
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Reference string  `json:"reference"`
}

// WalletTransactionResponse represents response for wallet operations.
type WalletTransactionResponse struct {
	TransactionID  string            `json:"transaction_id"`
	Status         TransactionStatus `json:"status"`
	CurrentBalance float64           `json:"current_balance"`
	Amount         float64           `json:"amount"`
	Type           TransactionType   `json:"type"`
	CreatedAt      time.Time         `json:"created_at"`
}

// GetWalletResponse represents response for get wallet details.
type GetWalletResponse struct {
	WalletID  string    `json:"wallet_id"`
	UserID    string    `json:"user_id"`
	Balance   float64   `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

