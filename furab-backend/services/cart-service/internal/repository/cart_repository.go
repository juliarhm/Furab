// Package repository provides data access layer for cart.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/cart-service/internal/model"
)

// Common repository errors.
var (
	ErrCartNotFound = errors.New("cart not found")
	ErrItemNotFound = errors.New("item not found in cart")
)

// CartRepository defines the interface for cart data access.
type CartRepository interface {
	// GetByUserID retrieves a cart by user ID.
	GetByUserID(ctx context.Context, userID string) (*model.Cart, error)

	// Save creates or updates a cart (upsert).
	Save(ctx context.Context, cart *model.Cart) error

	// Delete removes a cart by user ID.
	Delete(ctx context.Context, userID string) error
}

// postgresCartRepository implements CartRepository using PostgreSQL.
// Uses a single table with JSONB for items to keep it simple.
type postgresCartRepository struct {
	db *sql.DB
}

// NewPostgresCartRepository creates a new PostgreSQL-based CartRepository.
func NewPostgresCartRepository(db *sql.DB) CartRepository {
	return &postgresCartRepository{db: db}
}

// GetByUserID retrieves a cart by user ID.
func (r *postgresCartRepository) GetByUserID(ctx context.Context, userID string) (*model.Cart, error) {
	query := `
		SELECT id, user_id, merchant_id, items, total_price, item_count, created_at, updated_at
		FROM carts
		WHERE user_id = $1
	`

	cart := &model.Cart{}
	var itemsJSON []byte
	var merchantID sql.NullString

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&cart.ID, &cart.UserID, &merchantID,
		&itemsJSON, &cart.TotalPrice, &cart.ItemCount,
		&cart.CreatedAt, &cart.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCartNotFound
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if merchantID.Valid {
		cart.MerchantID = merchantID.String
	}

	// Unmarshal items from JSONB
	if len(itemsJSON) > 0 {
		if err := json.Unmarshal(itemsJSON, &cart.Items); err != nil {
			return nil, fmt.Errorf("failed to unmarshal items: %w", err)
		}
	} else {
		cart.Items = []model.CartItem{}
	}

	return cart, nil
}

// Save creates or updates a cart using PostgreSQL upsert (ON CONFLICT).
func (r *postgresCartRepository) Save(ctx context.Context, cart *model.Cart) error {
	// Marshal items to JSON
	itemsJSON, err := json.Marshal(cart.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	cart.UpdatedAt = time.Now().UTC()

	query := `
		INSERT INTO carts (id, user_id, merchant_id, items, total_price, item_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			merchant_id = EXCLUDED.merchant_id,
			items = EXCLUDED.items,
			total_price = EXCLUDED.total_price,
			item_count = EXCLUDED.item_count,
			updated_at = EXCLUDED.updated_at
	`

	_, err = r.db.ExecContext(ctx, query,
		cart.ID, cart.UserID, nullString(cart.MerchantID),
		itemsJSON, cart.TotalPrice, cart.ItemCount,
		cart.CreatedAt, cart.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save cart: %w", err)
	}

	return nil
}

// Delete removes a cart by user ID.
func (r *postgresCartRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM carts WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}
	return nil
}

// nullString converts a string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
