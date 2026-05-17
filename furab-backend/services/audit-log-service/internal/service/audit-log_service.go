package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/audit-log-service/internal/model"
	"furab-backend/services/audit-log-service/internal/repository"
	"github.com/google/uuid"
)

// AuditLogService defines the business logic for audit logs.
type AuditLogService interface {
	CreateAuditLog(ctx context.Context, m *model.AuditLog) error
	GetAuditLog(ctx context.Context, id string) (*model.AuditLog, error)
	UpdateAuditLog(ctx context.Context, m *model.AuditLog) error
	DeleteAuditLog(ctx context.Context, id string) error
	SearchAuditLogs(ctx context.Context, req model.SearchAuditLogRequest) ([]model.AuditLog, int, error)
}

type auditLogService struct {
	repo repository.AuditLogRepository
}

// NewAuditLogService creates a new instance of audit log service.
func NewAuditLogService(repo repository.AuditLogRepository) AuditLogService {
	return &auditLogService{repo: repo}
}

func (s *auditLogService) CreateAuditLog(ctx context.Context, m *model.AuditLog) error {
	if m.UserID == "" {
		return errors.New("user id is required")
	}
	if m.Action == "" {
		return errors.New("action is required")
	}
	if m.Entity == "" {
		return errors.New("entity is required")
	}

	m.ID = uuid.New().String()
	m.CreatedAt = time.Now()

	return s.repo.Save(ctx, m)
}

func (s *auditLogService) GetAuditLog(ctx context.Context, id string) (*model.AuditLog, error) {
	if id == "" {
		return nil, errors.New("audit log id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *auditLogService) UpdateAuditLog(ctx context.Context, m *model.AuditLog) error {
	if m.ID == "" {
		return errors.New("audit log id is required")
	}

	existing, err := s.repo.GetByID(ctx, m.ID)
	if err != nil {
		return err
	}

	if m.UserID != "" {
		existing.UserID = m.UserID
	}
	if m.Action != "" {
		existing.Action = m.Action
	}
	if m.Entity != "" {
		existing.Entity = m.Entity
	}
	if m.EntityID != "" {
		existing.EntityID = m.EntityID
	}
	existing.OldData = m.OldData
	existing.NewData = m.NewData

	return s.repo.Update(ctx, existing)
}

func (s *auditLogService) DeleteAuditLog(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("audit log id is required")
	}
	return s.repo.Delete(ctx, id)
}

func (s *auditLogService) SearchAuditLogs(ctx context.Context, req model.SearchAuditLogRequest) ([]model.AuditLog, int, error) {
	return s.repo.Search(ctx, req)
}
