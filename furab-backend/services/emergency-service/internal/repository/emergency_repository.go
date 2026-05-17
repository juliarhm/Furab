// Package repository provides data access layer for emergency-service.
package repository

import (
	"context"

	"furab-backend/services/emergency-service/internal/model"
)

// EmergencyRepository defines the interface for emergency-service data access.
type EmergencyRepository interface {
	SaveEmergencyEvent(ctx context.Context, event model.EmergencyEvent) error
}

// postgresEmergencyRepository implements EmergencyRepository using PostgreSQL.
type postgresEmergencyRepository struct {
	// TODO: add *sql.DB field
}

// NewPostgresEmergencyRepository creates a new PostgreSQL-based repository.
func NewPostgresEmergencyRepository() EmergencyRepository {
	return &postgresEmergencyRepository{}
}

func (r *postgresEmergencyRepository) SaveEmergencyEvent(ctx context.Context, event model.EmergencyEvent) error {
	return nil
}
