// Package repository provides data access layer for driver-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/driver-service/internal/model"
)

// Common repository errors.
var (
	ErrDriverNotFound = errors.New("driver not found")
)

// DriverRepository defines the interface for driver-service data access.
type DriverRepository interface {
	// Save persists a new driver.
	Save(ctx context.Context, driver *model.Driver) error

	// FindByID retrieves a driver by its ID.
	FindByID(ctx context.Context, driverID string) (*model.Driver, error)

	// Update modifies an existing driver record.
	Update(ctx context.Context, driver *model.Driver) error

	// UpdateStatus changes the availability status of a driver.
	UpdateStatus(ctx context.Context, driverID, status string) error

	// UpdateLocation updates the GPS coordinates of a driver.
	UpdateLocation(ctx context.Context, driverID string, lat, long float64) error
}

// postgresDriverRepository implements DriverRepository using PostgreSQL.
type postgresDriverRepository struct {
	db *sql.DB
}

// NewPostgresDriverRepository creates a new PostgreSQL-based repository.
func NewPostgresDriverRepository(db *sql.DB) DriverRepository {
	return &postgresDriverRepository{db: db}
}

// Save persists a new driver.
func (r *postgresDriverRepository) Save(ctx context.Context, driver *model.Driver) error {
	query := `
		INSERT INTO drivers (
			driver_id, name, phone, vehicle_type, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		driver.DriverID, driver.Name, driver.Phone, driver.VehicleType, driver.Status,
		driver.CreatedAt, driver.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save driver: %w", err)
	}
	return nil
}

// FindByID retrieves a driver by ID.
func (r *postgresDriverRepository) FindByID(ctx context.Context, driverID string) (*model.Driver, error) {
	query := `
		SELECT driver_id, name, phone, vehicle_type, status,
			created_at, updated_at
		FROM drivers
		WHERE driver_id = $1
	`
	driver := &model.Driver{}
	err := r.db.QueryRowContext(ctx, query, driverID).Scan(
		&driver.DriverID, &driver.Name, &driver.Phone, &driver.VehicleType, &driver.Status,
		&driver.CreatedAt, &driver.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDriverNotFound
		}
		return nil, fmt.Errorf("failed to find driver: %w", err)
	}
	return driver, nil
}

// Update modifies an existing driver record.
func (r *postgresDriverRepository) Update(ctx context.Context, driver *model.Driver) error {
	query := `
		UPDATE drivers SET
			name = $2, phone = $3, vehicle_type = $4,
			status = $5, updated_at = $6
		WHERE driver_id = $1
	`
	driver.UpdatedAt = time.Now().UTC()
	result, err := r.db.ExecContext(ctx, query,
		driver.DriverID, driver.Name, driver.Phone, driver.VehicleType,
		driver.Status, driver.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update driver: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrDriverNotFound
	}
	return nil
}

// UpdateStatus changes the availability status of a driver.
func (r *postgresDriverRepository) UpdateStatus(ctx context.Context, driverID, status string) error {
	query := `UPDATE drivers SET status = $2, updated_at = $3 WHERE driver_id = $1`
	result, err := r.db.ExecContext(ctx, query, driverID, status, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update driver status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrDriverNotFound
	}
	return nil
}

// UpdateLocation updates the GPS coordinates of a driver.
func (r *postgresDriverRepository) UpdateLocation(ctx context.Context, driverID string, lat, long float64) error {
	query := `
		INSERT INTO driver_locations (driver_id, latitude, longitude, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (driver_id) DO UPDATE SET
			latitude = $2, longitude = $3, updated_at = $4
	`
	_, err := r.db.ExecContext(ctx, query, driverID, lat, long, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}
	return nil
}
