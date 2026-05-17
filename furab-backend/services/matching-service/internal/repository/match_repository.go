// Package repository provides data access layer for match requests.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/matching-service/internal/model"
)

var (
	ErrMatchNotFound = errors.New("match request not found")
)

// MatchRepository defines the interface for match request data access.
type MatchRepository interface {
	Create(ctx context.Context, match *model.MatchRequest) error
	GetByID(ctx context.Context, id string) (*model.MatchRequest, error)
	Update(ctx context.Context, match *model.MatchRequest) error
	GetByOrderID(ctx context.Context, orderID string) (*model.MatchRequest, error)
}

// postgresMatchRepository implements MatchRepository using PostgreSQL.
type postgresMatchRepository struct {
	db *sql.DB
}

// NewPostgresMatchRepository creates a new PostgreSQL-based MatchRepository.
func NewPostgresMatchRepository(db *sql.DB) MatchRepository {
	return &postgresMatchRepository{db: db}
}

func (r *postgresMatchRepository) Create(ctx context.Context, match *model.MatchRequest) error {
	query := `
		INSERT INTO match_requests (
			id, order_id, order_type, user_id,
			pickup_lat, pickup_lng, pickup_address,
			dropoff_lat, dropoff_lng, dropoff_address,
			status, driver_id, attempt_count, max_attempts, radius,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
	`

	_, err := r.db.ExecContext(ctx, query,
		match.ID, match.OrderID, match.OrderType, match.UserID,
		match.PickupLocation.Latitude, match.PickupLocation.Longitude, match.PickupLocation.Address,
		match.DropoffLocation.Latitude, match.DropoffLocation.Longitude, match.DropoffLocation.Address,
		match.Status, nullStr(match.DriverID), match.AttemptCount, match.MaxAttempts, match.Radius,
		match.CreatedAt, match.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}
	return nil
}

func (r *postgresMatchRepository) GetByID(ctx context.Context, id string) (*model.MatchRequest, error) {
	query := `
		SELECT id, order_id, order_type, user_id,
			pickup_lat, pickup_lng, pickup_address,
			dropoff_lat, dropoff_lng, dropoff_address,
			status, driver_id, attempt_count, max_attempts, radius,
			created_at, updated_at
		FROM match_requests
		WHERE id = $1
	`

	match := &model.MatchRequest{}
	var driverID sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&match.ID, &match.OrderID, &match.OrderType, &match.UserID,
		&match.PickupLocation.Latitude, &match.PickupLocation.Longitude, &match.PickupLocation.Address,
		&match.DropoffLocation.Latitude, &match.DropoffLocation.Longitude, &match.DropoffLocation.Address,
		&match.Status, &driverID, &match.AttemptCount, &match.MaxAttempts, &match.Radius,
		&match.CreatedAt, &match.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, fmt.Errorf("failed to get match: %w", err)
	}

	if driverID.Valid {
		match.DriverID = driverID.String
	}

	return match, nil
}

func (r *postgresMatchRepository) Update(ctx context.Context, match *model.MatchRequest) error {
	match.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE match_requests SET
			status = $2, driver_id = $3,
			attempt_count = $4, radius = $5,
			updated_at = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		match.ID, match.Status, nullStr(match.DriverID),
		match.AttemptCount, match.Radius,
		match.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update match: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows: %w", err)
	}
	if rows == 0 {
		return ErrMatchNotFound
	}

	return nil
}

func (r *postgresMatchRepository) GetByOrderID(ctx context.Context, orderID string) (*model.MatchRequest, error) {
	query := `
		SELECT id, order_id, order_type, user_id,
			pickup_lat, pickup_lng, pickup_address,
			dropoff_lat, dropoff_lng, dropoff_address,
			status, driver_id, attempt_count, max_attempts, radius,
			created_at, updated_at
		FROM match_requests
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	match := &model.MatchRequest{}
	var driverID sql.NullString

	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&match.ID, &match.OrderID, &match.OrderType, &match.UserID,
		&match.PickupLocation.Latitude, &match.PickupLocation.Longitude, &match.PickupLocation.Address,
		&match.DropoffLocation.Latitude, &match.DropoffLocation.Longitude, &match.DropoffLocation.Address,
		&match.Status, &driverID, &match.AttemptCount, &match.MaxAttempts, &match.Radius,
		&match.CreatedAt, &match.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, fmt.Errorf("failed to get match by order: %w", err)
	}

	if driverID.Valid {
		match.DriverID = driverID.String
	}

	return match, nil
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
