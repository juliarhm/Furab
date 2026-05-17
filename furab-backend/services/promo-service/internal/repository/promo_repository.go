// Package repository provides data access layer for promo-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/promo-service/internal/model"
)

var (
	ErrPromoNotFound      = errors.New("promo not found")
	ErrPromoUsageExceeded = errors.New("promo usage limit exceeded")
)

// PromoRepository defines the interface for promo-service promo data access.
type PromoRepository interface {
	GetPromoByCode(ctx context.Context, promoCode string) (*model.Promo, error)
	IncrementUsage(ctx context.Context, promoID string) error
}

type inMemoryPromoRepository struct {
	promos []*model.Promo
}

// NewInMemoryPromoRepository creates a fixed promo repository for testing and development.
func NewInMemoryPromoRepository() PromoRepository {
	return &inMemoryPromoRepository{
		promos: []*model.Promo{
			{
				PromoID:       "promo-001",
				Code:          "DISKONHEMAT",
				DiscountType:  "percentage",
				DiscountValue: 0.1,
				MinPurchase:   50000,
				MaxDiscount:   20000,
				ExpiryDate:    time.Now().AddDate(0, 1, 0),
				UsageLimit:    100,
				UsageCount:    0,
			},
			{
				PromoID:       "promo-002",
				Code:          "FIXED50",
				DiscountType:  "fixed",
				DiscountValue: 50000,
				MinPurchase:   100000,
				MaxDiscount:   50000,
				ExpiryDate:    time.Now().AddDate(0, 2, 0),
				UsageLimit:    50,
				UsageCount:    0,
			},
			{
				PromoID:       "promo-003",
				Code:          "EXPIRED",
				DiscountType:  "fixed",
				DiscountValue: 10000,
				MinPurchase:   0,
				MaxDiscount:   0,
				ExpiryDate:    time.Now().AddDate(0, -1, 0),
				UsageLimit:    100,
				UsageCount:    0,
			},
			{
				PromoID:       "promo-004",
				Code:          "FULL",
				DiscountType:  "fixed",
				DiscountValue: 10000,
				MinPurchase:   0,
				MaxDiscount:   0,
				ExpiryDate:    time.Now().AddDate(0, 1, 0),
				UsageLimit:    10,
				UsageCount:    10,
			},
			{
				PromoID:       "promo-005",
				Code:          "BIGPERCENT",
				DiscountType:  "percentage",
				DiscountValue: 0.5,
				MinPurchase:   0,
				MaxDiscount:   10000,
				ExpiryDate:    time.Now().AddDate(0, 1, 0),
				UsageLimit:    100,
				UsageCount:    0,
			},
		},
	}
}

func (r *inMemoryPromoRepository) GetPromoByCode(ctx context.Context, promoCode string) (*model.Promo, error) {
	for _, promo := range r.promos {
		if promo.Code == promoCode {
			return promo, nil
		}
	}

	return nil, ErrPromoNotFound
}

func (r *inMemoryPromoRepository) IncrementUsage(ctx context.Context, promoID string) error {
	for _, promo := range r.promos {
		if promo.PromoID == promoID {
			if promo.UsageLimit > 0 && promo.UsageCount >= promo.UsageLimit {
				return ErrPromoUsageExceeded
			}
			promo.UsageCount++
			return nil
		}
	}

	return ErrPromoNotFound
}

type postgresPromoRepository struct {
	db *sql.DB
}

// NewPostgresPromoRepository creates a new PostgreSQL-based repository.
func NewPostgresPromoRepository(db *sql.DB) PromoRepository {
	return &postgresPromoRepository{db: db}
}

func (r *postgresPromoRepository) GetPromoByCode(ctx context.Context, promoCode string) (*model.Promo, error) {
	query := `
		SELECT promo_id, code, discount_type, discount_value, min_purchase, max_discount, expiry_date, usage_limit, usage_count
		FROM promos
		WHERE code = $1
	`
	promo := &model.Promo{}
	err := r.db.QueryRowContext(ctx, query, promoCode).Scan(
		&promo.PromoID, &promo.Code, &promo.DiscountType, &promo.DiscountValue,
		&promo.MinPurchase, &promo.MaxDiscount, &promo.ExpiryDate,
		&promo.UsageLimit, &promo.UsageCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPromoNotFound
		}
		return nil, fmt.Errorf("failed to get promo by code: %w", err)
	}

	return promo, nil
}

func (r *postgresPromoRepository) IncrementUsage(ctx context.Context, promoID string) error {
	// First check usage limit with FOR UPDATE to prevent race conditions
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var usageLimit, usageCount int
	queryCheck := `SELECT usage_limit, usage_count FROM promos WHERE promo_id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, queryCheck, promoID).Scan(&usageLimit, &usageCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPromoNotFound
		}
		return fmt.Errorf("failed to check promo usage: %w", err)
	}

	if usageLimit > 0 && usageCount >= usageLimit {
		return ErrPromoUsageExceeded
	}

	queryUpdate := `UPDATE promos SET usage_count = usage_count + 1 WHERE promo_id = $1`
	_, err = tx.ExecContext(ctx, queryUpdate, promoID)
	if err != nil {
		return fmt.Errorf("failed to increment promo usage: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
