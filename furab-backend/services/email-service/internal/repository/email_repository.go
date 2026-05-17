// Package repository provides data access layer for email-service.
package repository

import (
	"context"

	"furab-backend/services/email-service/internal/model"
)

// EmailRepository defines the interface for email-service data access.
type EmailRepository interface {
	// SaveEmailLog stores email delivery result for monitoring/audit.
	SaveEmailLog(ctx context.Context, log model.EmailLog) error
	// GetTemplateByID fetches an email template by ID.
	GetTemplateByID(ctx context.Context, templateID string) (*model.EmailTemplate, error)
}

// postgresEmailRepository implements EmailRepository using PostgreSQL.
type postgresEmailRepository struct {
	// TODO: add *sql.DB field
}

// NewPostgresEmailRepository creates a new PostgreSQL-based repository.
func NewPostgresEmailRepository() EmailRepository {
	return &postgresEmailRepository{}
}

// SaveEmailLog stores email logs in database.
func (r *postgresEmailRepository) SaveEmailLog(ctx context.Context, log model.EmailLog) error {
	_ = ctx
	_ = log
	// TODO: implement database persistence.
	return nil
}

// GetTemplateByID fetches a template by ID from database.
func (r *postgresEmailRepository) GetTemplateByID(ctx context.Context, templateID string) (*model.EmailTemplate, error) {
	_ = ctx
	_ = templateID
	// TODO: implement database persistence.
	return nil, nil
}
