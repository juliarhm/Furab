// Package repository provides data access layer for OTP service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"furab-backend/services/otp-service/internal/model"
)

// Common repository errors.
var (
	ErrOTPNotFound = errors.New("otp not found")
)

// OTPRepository defines the interface for OTP data access.
// This interface is used for dependency injection and can be mocked in unit tests.
type OTPRepository interface {
	// Save inserts a new OTP record into the database.
	Save(ctx context.Context, otp *model.OTP) error

	// GetByTarget retrieves the latest OTP for a given target (phone/email).
	GetByTarget(ctx context.Context, target string) (*model.OTP, error)

	// Delete removes an OTP record by its ID (after successful verification).
	Delete(ctx context.Context, otpID string) error
}

// postgresOTPRepository implements OTPRepository using PostgreSQL.
type postgresOTPRepository struct {
	db *sql.DB
}

// NewPostgresOTPRepository creates a new PostgreSQL-based OTPRepository.
func NewPostgresOTPRepository(db *sql.DB) OTPRepository {
	return &postgresOTPRepository{db: db}
}

// Save inserts a new OTP record into PostgreSQL.
func (r *postgresOTPRepository) Save(ctx context.Context, otp *model.OTP) error {
	query := `
		INSERT INTO otps (
			otp_id, target, otp_code, expired_at, created_at
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		otp.OTPID, otp.Target, otp.OTPCode, otp.ExpiredAt, otp.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save otp: %w", err)
	}
	return nil
}

// GetByTarget retrieves the latest OTP for a given target from PostgreSQL.
func (r *postgresOTPRepository) GetByTarget(ctx context.Context, target string) (*model.OTP, error) {
	query := `
		SELECT otp_id, target, otp_code, expired_at, created_at
		FROM otps
		WHERE target = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	otp := &model.OTP{}
	err := r.db.QueryRowContext(ctx, query, target).Scan(
		&otp.OTPID, &otp.Target, &otp.OTPCode, &otp.ExpiredAt, &otp.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOTPNotFound
		}
		return nil, fmt.Errorf("failed to get otp: %w", err)
	}

	return otp, nil
}

// Delete removes an OTP record by its ID from PostgreSQL.
func (r *postgresOTPRepository) Delete(ctx context.Context, otpID string) error {
	query := `DELETE FROM otps WHERE otp_id = $1`
	result, err := r.db.ExecContext(ctx, query, otpID)
	if err != nil {
		return fmt.Errorf("failed to delete otp: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrOTPNotFound
	}

	return nil
}
