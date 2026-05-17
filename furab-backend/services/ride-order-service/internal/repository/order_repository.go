// Package repository provides data access layer for ride orders.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/ride-order-service/internal/model"
)

// Common repository errors.
var (
	ErrOrderNotFound  = errors.New("order not found")
	ErrDuplicateOrder = errors.New("duplicate order")
)

// OrderRepository defines the interface for ride order data access.
type OrderRepository interface {
	Create(ctx context.Context, order *model.RideOrder) error
	GetByID(ctx context.Context, id string) (*model.RideOrder, error)
	Update(ctx context.Context, order *model.RideOrder) error
	UpdateStatus(ctx context.Context, id string, status model.RideStatus) error
	AssignDriver(ctx context.Context, orderID, driverID string) error
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.RideOrder, error)
	GetByDriverID(ctx context.Context, driverID string, limit, offset int) ([]*model.RideOrder, error)
	CountByUserID(ctx context.Context, userID string) (int, error)
	Delete(ctx context.Context, id string) error
}

// postgresOrderRepository implements OrderRepository using PostgreSQL.
type postgresOrderRepository struct {
	db *sql.DB
}

// NewPostgresOrderRepository creates a new PostgreSQL-based OrderRepository.
func NewPostgresOrderRepository(db *sql.DB) OrderRepository {
	return &postgresOrderRepository{db: db}
}

// Create inserts a new ride order into PostgreSQL.
func (r *postgresOrderRepository) Create(ctx context.Context, order *model.RideOrder) error {
	query := `
		INSERT INTO ride_orders (
			id, user_id, driver_id, 
			pickup_lat, pickup_lng, pickup_address,
			dropoff_lat, dropoff_lng, dropoff_address,
			status, payment_status, fare, distance, estimated_duration,
			cancelled_by, cancel_reason,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err := r.db.ExecContext(ctx, query,
		order.ID, order.UserID, nullString(order.DriverID),
		order.PickupLocation.Latitude, order.PickupLocation.Longitude, order.PickupLocation.Address,
		order.DropoffLocation.Latitude, order.DropoffLocation.Longitude, order.DropoffLocation.Address,
		order.Status, order.PaymentStatus, order.Fare, order.Distance, order.EstimatedDuration,
		nullString(order.CancelledBy), nullString(order.CancelReason),
		order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}
	return nil
}

// GetByID retrieves a ride order by its ID from PostgreSQL.
func (r *postgresOrderRepository) GetByID(ctx context.Context, id string) (*model.RideOrder, error) {
	query := `
		SELECT id, user_id, driver_id,
			pickup_lat, pickup_lng, pickup_address,
			dropoff_lat, dropoff_lng, dropoff_address,
			status, payment_status, fare, distance, estimated_duration,
			cancelled_by, cancel_reason,
			created_at, updated_at
		FROM ride_orders
		WHERE id = $1
	`

	order := &model.RideOrder{}
	var driverID, cancelledBy, cancelReason sql.NullString
	var paymentStatus sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID, &order.UserID, &driverID,
		&order.PickupLocation.Latitude, &order.PickupLocation.Longitude, &order.PickupLocation.Address,
		&order.DropoffLocation.Latitude, &order.DropoffLocation.Longitude, &order.DropoffLocation.Address,
		&order.Status, &paymentStatus, &order.Fare, &order.Distance, &order.EstimatedDuration,
		&cancelledBy, &cancelReason,
		&order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if driverID.Valid {
		order.DriverID = driverID.String
	}
	if paymentStatus.Valid {
		order.PaymentStatus = model.PaymentStatus(paymentStatus.String)
	}
	if cancelledBy.Valid {
		order.CancelledBy = cancelledBy.String
	}
	if cancelReason.Valid {
		order.CancelReason = cancelReason.String
	}

	return order, nil
}

// Update updates an existing ride order in PostgreSQL.
func (r *postgresOrderRepository) Update(ctx context.Context, order *model.RideOrder) error {
	query := `
		UPDATE ride_orders SET
			driver_id = $2,
			status = $3,
			payment_status = $4,
			fare = $5,
			distance = $6,
			estimated_duration = $7,
			cancelled_by = $8,
			cancel_reason = $9,
			updated_at = $10
		WHERE id = $1
	`

	order.UpdatedAt = time.Now().UTC()
	result, err := r.db.ExecContext(ctx, query,
		order.ID, nullString(order.DriverID), order.Status,
		order.PaymentStatus, order.Fare, order.Distance, order.EstimatedDuration,
		nullString(order.CancelledBy), nullString(order.CancelReason),
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
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

// UpdateStatus updates only the status of a ride order.
func (r *postgresOrderRepository) UpdateStatus(ctx context.Context, id string, status model.RideStatus) error {
	query := `UPDATE ride_orders SET status = $2, updated_at = $3 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, status, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
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

// AssignDriver assigns a driver to a ride order.
func (r *postgresOrderRepository) AssignDriver(ctx context.Context, orderID, driverID string) error {
	query := `
		UPDATE ride_orders 
		SET driver_id = $2, status = $3, updated_at = $4 
		WHERE id = $1 AND status = $5
	`

	result, err := r.db.ExecContext(ctx, query,
		orderID, driverID, model.RideStatusAssigned, time.Now().UTC(), model.RideStatusPending,
	)
	if err != nil {
		return fmt.Errorf("failed to assign driver: %w", err)
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

// GetByUserID retrieves all ride orders for a specific user with pagination.
func (r *postgresOrderRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.RideOrder, error) {
	query := `
		SELECT id, user_id, driver_id,
			pickup_lat, pickup_lng, pickup_address,
			dropoff_lat, dropoff_lng, dropoff_address,
			status, payment_status, fare, distance, estimated_duration,
			cancelled_by, cancel_reason,
			created_at, updated_at
		FROM ride_orders
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

// GetByDriverID retrieves all ride orders for a specific driver with pagination.
func (r *postgresOrderRepository) GetByDriverID(ctx context.Context, driverID string, limit, offset int) ([]*model.RideOrder, error) {
	query := `
		SELECT id, user_id, driver_id,
			pickup_lat, pickup_lng, pickup_address,
			dropoff_lat, dropoff_lng, dropoff_address,
			status, payment_status, fare, distance, estimated_duration,
			cancelled_by, cancel_reason,
			created_at, updated_at
		FROM ride_orders
		WHERE driver_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, driverID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by driver: %w", err)
	}
	defer rows.Close()

	return r.scanOrders(rows)
}

// CountByUserID counts the total number of orders for a user.
func (r *postgresOrderRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM ride_orders WHERE user_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count orders: %w", err)
	}
	return count, nil
}

// Delete removes a ride order.
func (r *postgresOrderRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM ride_orders WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
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

// --- Helpers ---

// scanOrders scans multiple rows into RideOrder slices.
func (r *postgresOrderRepository) scanOrders(rows *sql.Rows) ([]*model.RideOrder, error) {
	var orders []*model.RideOrder
	for rows.Next() {
		order := &model.RideOrder{}
		var driverID, cancelledBy, cancelReason, paymentStatus sql.NullString

		err := rows.Scan(
			&order.ID, &order.UserID, &driverID,
			&order.PickupLocation.Latitude, &order.PickupLocation.Longitude, &order.PickupLocation.Address,
			&order.DropoffLocation.Latitude, &order.DropoffLocation.Longitude, &order.DropoffLocation.Address,
			&order.Status, &paymentStatus, &order.Fare, &order.Distance, &order.EstimatedDuration,
			&cancelledBy, &cancelReason,
			&order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		if driverID.Valid {
			order.DriverID = driverID.String
		}
		if paymentStatus.Valid {
			order.PaymentStatus = model.PaymentStatus(paymentStatus.String)
		}
		if cancelledBy.Valid {
			order.CancelledBy = cancelledBy.String
		}
		if cancelReason.Valid {
			order.CancelReason = cancelReason.String
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return orders, nil
}

// nullString converts a string to sql.NullString for nullable columns.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
