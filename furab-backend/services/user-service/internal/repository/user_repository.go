// Package repository provides data access layer for user-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/user-service/internal/model"
)

// Common repository errors.
var (
	ErrUserNotFound  = errors.New("user not found")
	ErrDuplicateUser = errors.New("duplicate user")
)

// UserRepository defines the interface for user data access.
// This interface is used for dependency injection and can be mocked in unit tests.
type UserRepository interface {
	// Create inserts a new user into the database.
	Create(ctx context.Context, user *model.User) error

	// GetByID retrieves a user by their ID.
	GetByID(ctx context.Context, id string) (*model.User, error)

	// Update updates an existing user.
	Update(ctx context.Context, user *model.User) error
}

// postgresUserRepository implements UserRepository using PostgreSQL.
type postgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a new PostgreSQL-based UserRepository.
func NewPostgresUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

// Create inserts a new user into PostgreSQL.
func (r *postgresUserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (
			user_id, name, phone, email, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.UserID, user.Name, user.Phone, user.Email, user.Status,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by their ID from PostgreSQL.
func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT user_id, name, phone, email, status,
			created_at, updated_at
		FROM users
		WHERE user_id = $1
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.UserID, &user.Name, &user.Phone, &user.Email, &user.Status,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// Update updates an existing user in PostgreSQL.
func (r *postgresUserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET
			name = $2,
			phone = $3,
			email = $4,
			status = $5,
			updated_at = $6
		WHERE user_id = $1
	`

	user.UpdatedAt = time.Now().UTC()
	result, err := r.db.ExecContext(ctx, query,
		user.UserID, user.Name, user.Phone, user.Email, user.Status,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
