// Package repository provides data access layer for notification-service.
package repository

import (
	"context"

	"furab-backend/services/notification-service/internal/model"
)

// NotificationRepository defines the interface for notification-service data access.
type NotificationRepository interface {
	SaveNotificationLog(ctx context.Context, log model.NotificationLog) error
	GetTemplateByEventType(ctx context.Context, eventType string) (*model.NotifTemplate, error)
}

// postgresNotificationRepository implements NotificationRepository using PostgreSQL.
type postgresNotificationRepository struct {
	// TODO: add *sql.DB field
}

// NewPostgresNotificationRepository creates a new PostgreSQL-based repository.
func NewPostgresNotificationRepository() NotificationRepository {
	return &postgresNotificationRepository{}
}

func (r *postgresNotificationRepository) SaveNotificationLog(ctx context.Context, log model.NotificationLog) error {
	return nil
}

func (r *postgresNotificationRepository) GetTemplateByEventType(ctx context.Context, eventType string) (*model.NotifTemplate, error) {
	return nil, nil
}
