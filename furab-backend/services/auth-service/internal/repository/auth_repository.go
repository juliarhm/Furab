// Package repository provides data access layer for auth-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"furab-backend/services/auth-service/internal/model"
)

// Common repository errors.
var (
	ErrSessionNotFound = errors.New("session not found")
)

// AuthRepository defines the interface for auth-service data access.
type AuthRepository interface {
	SaveSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, token string) (*model.Session, error)
}

// postgresAuthRepository implements AuthRepository using PostgreSQL.
type postgresAuthRepository struct {
	db *sql.DB
}

// NewPostgresAuthRepository creates a new PostgreSQL-based repository.
func NewPostgresAuthRepository(db *sql.DB) AuthRepository {
	return &postgresAuthRepository{db: db}
}

// SaveSession persists a new session.
func (r *postgresAuthRepository) SaveSession(ctx context.Context, session *model.Session) error {
	query := `
		INSERT INTO sessions (session_id, user_id, token, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		session.SessionID, session.UserID, session.Token,
		session.CreatedAt, session.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}
	return nil
}

// GetSession retrieves a session by token.
func (r *postgresAuthRepository) GetSession(ctx context.Context, token string) (*model.Session, error) {
	query := `
		SELECT session_id, user_id, token, created_at, expires_at
		FROM sessions
		WHERE token = $1
	`
	session := &model.Session{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&session.SessionID, &session.UserID, &session.Token,
		&session.CreatedAt, &session.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return session, nil
}
