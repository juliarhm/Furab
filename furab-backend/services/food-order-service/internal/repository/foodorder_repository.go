// Package repository provides data access layer for food orders.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/food-order-service/internal/model"
)

var (
	ErrOrderNotFound  = errors.New("order not found")
	ErrDuplicateOrder = errors.New("duplicate order")
)

// FoodOrderRepository defines the interface for food order data access.
type FoodOrderRepository interface {
	Create(ctx context.Context, order *model.FoodOrder) error
	GetByID(ctx context.Context, id string) (*model.FoodOrder, error)
	Update(ctx context.Context, order *model.FoodOrder) error
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.FoodOrder, error)
	CountByUserID(ctx context.Context, userID string) (int, error)
}

// postgresFoodOrderRepository implements FoodOrderRepository using PostgreSQL.
type postgresFoodOrderRepository struct {
	db *sql.DB
}

// NewPostgresFoodOrderRepository creates a new PostgreSQL-based FoodOrderRepository.
func NewPostgresFoodOrderRepository(db *sql.DB) FoodOrderRepository {
	return &postgresFoodOrderRepository{db: db}
}

func (r *postgresFoodOrderRepository) Create(ctx context.Context, order *model.FoodOrder) error {
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	query := `
		INSERT INTO food_orders (
			id, user_id, merchant_id, driver_id, items,
			status, payment_status, sub_total, delivery_fee, discount, total_amount,
			delivery_lat, delivery_lng, delivery_address,
			merchant_lat, merchant_lng, merchant_address,
			cancelled_by, cancel_reason, notes,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
	`

	_, err = r.db.ExecContext(ctx, query,
		order.ID, order.UserID, order.MerchantID, nullStr(order.DriverID), itemsJSON,
		order.Status, order.PaymentStatus, order.SubTotal, order.DeliveryFee, order.Discount, order.TotalAmount,
		order.DeliveryAddress.Latitude, order.DeliveryAddress.Longitude, order.DeliveryAddress.Address,
		order.MerchantAddress.Latitude, order.MerchantAddress.Longitude, order.MerchantAddress.Address,
		nullStr(order.CancelledBy), nullStr(order.CancelReason), nullStr(order.Notes),
		order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create food order: %w", err)
	}
	return nil
}

func (r *postgresFoodOrderRepository) GetByID(ctx context.Context, id string) (*model.FoodOrder, error) {
	query := `
		SELECT id, user_id, merchant_id, driver_id, items,
			status, payment_status, sub_total, delivery_fee, discount, total_amount,
			delivery_lat, delivery_lng, delivery_address,
			merchant_lat, merchant_lng, merchant_address,
			cancelled_by, cancel_reason, notes,
			created_at, updated_at
		FROM food_orders
		WHERE id = $1
	`

	order := &model.FoodOrder{}
	var driverID, cancelledBy, cancelReason, notes sql.NullString
	var itemsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID, &order.UserID, &order.MerchantID, &driverID, &itemsJSON,
		&order.Status, &order.PaymentStatus, &order.SubTotal, &order.DeliveryFee, &order.Discount, &order.TotalAmount,
		&order.DeliveryAddress.Latitude, &order.DeliveryAddress.Longitude, &order.DeliveryAddress.Address,
		&order.MerchantAddress.Latitude, &order.MerchantAddress.Longitude, &order.MerchantAddress.Address,
		&cancelledBy, &cancelReason, &notes,
		&order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get food order: %w", err)
	}

	if driverID.Valid {
		order.DriverID = driverID.String
	}
	if cancelledBy.Valid {
		order.CancelledBy = cancelledBy.String
	}
	if cancelReason.Valid {
		order.CancelReason = cancelReason.String
	}
	if notes.Valid {
		order.Notes = notes.String
	}

	if len(itemsJSON) > 0 {
		if err := json.Unmarshal(itemsJSON, &order.Items); err != nil {
			return nil, fmt.Errorf("failed to unmarshal items: %w", err)
		}
	}

	return order, nil
}

func (r *postgresFoodOrderRepository) Update(ctx context.Context, order *model.FoodOrder) error {
	itemsJSON, err := json.Marshal(order.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	order.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE food_orders SET
			driver_id = $2, items = $3,
			status = $4, payment_status = $5,
			sub_total = $6, delivery_fee = $7, discount = $8, total_amount = $9,
			cancelled_by = $10, cancel_reason = $11, notes = $12,
			updated_at = $13
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		order.ID, nullStr(order.DriverID), itemsJSON,
		order.Status, order.PaymentStatus,
		order.SubTotal, order.DeliveryFee, order.Discount, order.TotalAmount,
		nullStr(order.CancelledBy), nullStr(order.CancelReason), nullStr(order.Notes),
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update food order: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrOrderNotFound
	}

	return nil
}

func (r *postgresFoodOrderRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.FoodOrder, error) {
	query := `
		SELECT id, user_id, merchant_id, driver_id, items,
			status, payment_status, sub_total, delivery_fee, discount, total_amount,
			delivery_lat, delivery_lng, delivery_address,
			merchant_lat, merchant_lng, merchant_address,
			cancelled_by, cancel_reason, notes,
			created_at, updated_at
		FROM food_orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by user: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

func (r *postgresFoodOrderRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM food_orders WHERE user_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count orders: %w", err)
	}
	return count, nil
}

func (r *postgresFoodOrderRepository) scanOrders(rows *sql.Rows) ([]*model.FoodOrder, error) {
	var orders []*model.FoodOrder
	for rows.Next() {
		order := &model.FoodOrder{}
		var driverID, cancelledBy, cancelReason, notes sql.NullString
		var itemsJSON []byte

		err := rows.Scan(
			&order.ID, &order.UserID, &order.MerchantID, &driverID, &itemsJSON,
			&order.Status, &order.PaymentStatus, &order.SubTotal, &order.DeliveryFee, &order.Discount, &order.TotalAmount,
			&order.DeliveryAddress.Latitude, &order.DeliveryAddress.Longitude, &order.DeliveryAddress.Address,
			&order.MerchantAddress.Latitude, &order.MerchantAddress.Longitude, &order.MerchantAddress.Address,
			&cancelledBy, &cancelReason, &notes,
			&order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		if driverID.Valid {
			order.DriverID = driverID.String
		}
		if cancelledBy.Valid {
			order.CancelledBy = cancelledBy.String
		}
		if cancelReason.Valid {
			order.CancelReason = cancelReason.String
		}
		if notes.Valid {
			order.Notes = notes.String
		}
		if len(itemsJSON) > 0 {
			json.Unmarshal(itemsJSON, &order.Items)
		}

		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return orders, nil
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
