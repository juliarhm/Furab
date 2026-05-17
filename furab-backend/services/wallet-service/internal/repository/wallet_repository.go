// Package repository provides data access layer for wallet-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/wallet-service/internal/model"
)

// Common repository errors.
var (
	ErrWalletNotFound = errors.New("wallet not found")
)

// WalletRepository defines the interface for wallet-service data access.
type WalletRepository interface {
	// GetByUserID retrieves wallet by user identifier.
	GetByUserID(ctx context.Context, userID string) (*model.Wallet, error)
	// UpdateBalance updates wallet available balance atomically.
	UpdateBalance(ctx context.Context, walletID string, newBalance float64) error
	// CreateTransaction stores wallet transaction audit log.
	CreateTransaction(ctx context.Context, tx *model.Transaction) error
	// GetTransactionByReference gets transaction by reference and type for idempotency.
	GetTransactionByReference(ctx context.Context, referenceID string, typ model.TransactionType) (*model.Transaction, error)
}

// postgresWalletRepository implements WalletRepository using PostgreSQL.
type postgresWalletRepository struct {
	db *sql.DB
}

// NewPostgresWalletRepository creates a new PostgreSQL-based repository.
func NewPostgresWalletRepository(db *sql.DB) WalletRepository {
	return &postgresWalletRepository{db: db}
}

func (r *postgresWalletRepository) GetByUserID(ctx context.Context, userID string) (*model.Wallet, error) {
	query := `
		SELECT wallet_id, user_id, balance, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
	`

	wallet := &model.Wallet{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to get wallet by user id: %w", err)
	}

	return wallet, nil
}

func (r *postgresWalletRepository) UpdateBalance(ctx context.Context, walletID string, newBalance float64) error {
	query := `
		UPDATE wallets
		SET balance = $2, updated_at = $3
		WHERE wallet_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, walletID, newBalance, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrWalletNotFound
	}

	return nil
}

func (r *postgresWalletRepository) CreateTransaction(ctx context.Context, tx *model.Transaction) error {
	query := `
		INSERT INTO wallet_transactions (
			transaction_id, wallet_id, reference_id, amount, type, status, current_balance, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		tx.ID,
		tx.WalletID,
		tx.ReferenceID,
		tx.Amount,
		tx.Type,
		tx.Status,
		tx.CurrentBalance,
		tx.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create wallet transaction: %w", err)
	}

	return nil
}

func (r *postgresWalletRepository) GetTransactionByReference(ctx context.Context, referenceID string, typ model.TransactionType) (*model.Transaction, error) {
	query := `
		SELECT transaction_id, wallet_id, reference_id, amount, type, status, current_balance, created_at
		FROM wallet_transactions
		WHERE reference_id = $1 AND type = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	tx := &model.Transaction{}
	err := r.db.QueryRowContext(ctx, query, referenceID, typ).Scan(
		&tx.ID,
		&tx.WalletID,
		&tx.ReferenceID,
		&tx.Amount,
		&tx.Type,
		&tx.Status,
		&tx.CurrentBalance,
		&tx.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get transaction by reference: %w", err)
	}

	return tx, nil
}
