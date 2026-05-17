// Package repository provides data access layer for payment-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/payment-service/internal/model"
)

// Common repository errors.
var (
	ErrPaymentNotFound = errors.New("payment not found")
)

// PaymentRepository defines the interface for payment-service data access.
// This interface is used for dependency injection and unit test mocking.
type PaymentRepository interface {
	CreatePayment(ctx context.Context, p *model.Payment) error
	GetPaymentByID(ctx context.Context, paymentID string) (*model.Payment, error)
	GetPaymentByIdempotencyKey(ctx context.Context, key string) (*model.Payment, error)
	UpdatePaymentStatus(ctx context.Context, paymentID string, status model.PaymentStatus) error
	CreatePaymentLog(ctx context.Context, paymentID string, status model.PaymentStatus) error
}

// postgresPaymentRepository implements PaymentRepository using PostgreSQL.
type postgresPaymentRepository struct {
	db *sql.DB
}

// NewPostgresPaymentRepository creates a new PostgreSQL-based repository.
func NewPostgresPaymentRepository(db *sql.DB) PaymentRepository {
	return &postgresPaymentRepository{db: db}
}

func (r *postgresPaymentRepository) CreatePayment(ctx context.Context, p *model.Payment) error {
	query := `
		INSERT INTO payments (
			id, order_id, user_id, amount, final_amount,
			method_id, payment_detail, payment_status, transaction_reference,
			idempotency_key, transaction_time, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`

	_, err := r.db.ExecContext(ctx, query,
		p.ID,
		p.OrderID,
		p.UserID,
		p.Amount,
		p.FinalAmount,
		p.MethodID,
		p.PaymentDetail,
		p.PaymentStatus,
		p.TransactionReference,
		p.IdempotencyKey,
		p.TransactionTime,
		p.CreatedAt,
		p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

func (r *postgresPaymentRepository) GetPaymentByID(ctx context.Context, paymentID string) (*model.Payment, error) {
	query := `
		SELECT id, order_id, user_id, amount, final_amount,
			method_id, payment_detail, payment_status, transaction_reference,
			idempotency_key, transaction_time, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	p := &model.Payment{}
	row := r.db.QueryRowContext(ctx, query, paymentID)
	if err := row.Scan(
		&p.ID,
		&p.OrderID,
		&p.UserID,
		&p.Amount,
		&p.FinalAmount,
		&p.MethodID,
		&p.PaymentDetail,
		&p.PaymentStatus,
		&p.TransactionReference,
		&p.IdempotencyKey,
		&p.TransactionTime,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by id: %w", err)
	}
	return p, nil
}

func (r *postgresPaymentRepository) GetPaymentByIdempotencyKey(ctx context.Context, key string) (*model.Payment, error) {
	query := `
		SELECT id, order_id, user_id, amount, final_amount,
			method_id, payment_detail, payment_status, transaction_reference,
			idempotency_key, transaction_time, created_at, updated_at
		FROM payments
		WHERE idempotency_key = $1
	`

	p := &model.Payment{}
	row := r.db.QueryRowContext(ctx, query, key)
	if err := row.Scan(
		&p.ID,
		&p.OrderID,
		&p.UserID,
		&p.Amount,
		&p.FinalAmount,
		&p.MethodID,
		&p.PaymentDetail,
		&p.PaymentStatus,
		&p.TransactionReference,
		&p.IdempotencyKey,
		&p.TransactionTime,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get payment by idempotency key: %w", err)
	}
	return p, nil
}

func (r *postgresPaymentRepository) UpdatePaymentStatus(ctx context.Context, paymentID string, status model.PaymentStatus) error {
	query := `
		UPDATE payments
		SET payment_status = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, paymentID, status, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to determine rows affected: %w", err)
	}
	if affected == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *postgresPaymentRepository) CreatePaymentLog(ctx context.Context, paymentID string, status model.PaymentStatus) error {
	query := `
		INSERT INTO payment_logs (payment_id, status, timestamp)
		VALUES ($1,$2,$3)
	`

	_, err := r.db.ExecContext(ctx, query, paymentID, status, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to create payment log: %w", err)
	}
	return nil
}
