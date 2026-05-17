// Package repository provides data access layer for settlement-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/settlement-service/internal/model"
)

var (
	ErrSettlementNotFound = errors.New("settlement not found")
)

// SettlementRepository defines the interface for settlement-service data access.
type SettlementRepository interface {
	CreateSettlement(ctx context.Context, s *model.Settlement) error
	GetSettlementByPaymentID(ctx context.Context, paymentID string) (*model.Settlement, error)
	UpdateSettlementStatus(ctx context.Context, settlementID string, status model.SettlementStatus) error
}

// postgresSettlementRepository implements SettlementRepository using PostgreSQL.
type postgresSettlementRepository struct {
	db *sql.DB
}

// NewPostgresSettlementRepository creates a new PostgreSQL-based repository.
func NewPostgresSettlementRepository(db *sql.DB) SettlementRepository {
	return &postgresSettlementRepository{db: db}
}

func (r *postgresSettlementRepository) CreateSettlement(ctx context.Context, s *model.Settlement) error {
	query := `
		INSERT INTO settlements (
			settlement_id, payment_id, order_id,
			total_amount, driver_amount, merchant_amount, platform_fee,
			status, idempotency_key, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		s.ID,
		s.PaymentID,
		s.OrderID,
		s.TotalAmount,
		s.DriverAmount,
		s.MerchantAmount,
		s.PlatformFee,
		s.Status,
		s.IdempotencyKey,
		s.CreatedAt,
		s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create settlement: %w", err)
	}
	return nil
}

func (r *postgresSettlementRepository) GetSettlementByPaymentID(ctx context.Context, paymentID string) (*model.Settlement, error) {
	query := `
		SELECT settlement_id, payment_id, order_id,
			total_amount, driver_amount, merchant_amount, platform_fee,
			status, idempotency_key, created_at, updated_at
		FROM settlements
		WHERE payment_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	settlement := &model.Settlement{}
	err := r.db.QueryRowContext(ctx, query, paymentID).Scan(
		&settlement.ID,
		&settlement.PaymentID,
		&settlement.OrderID,
		&settlement.TotalAmount,
		&settlement.DriverAmount,
		&settlement.MerchantAmount,
		&settlement.PlatformFee,
		&settlement.Status,
		&settlement.IdempotencyKey,
		&settlement.CreatedAt,
		&settlement.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get settlement by payment id: %w", err)
	}
	return settlement, nil
}

func (r *postgresSettlementRepository) UpdateSettlementStatus(ctx context.Context, settlementID string, status model.SettlementStatus) error {
	query := `
		UPDATE settlements
		SET status = $2, updated_at = $3
		WHERE settlement_id = $1
	`
	result, err := r.db.ExecContext(ctx, query, settlementID, status, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update settlement status: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSettlementNotFound
	}
	return nil
}
