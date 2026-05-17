package repository

import (
	"context"
	"database/sql"
	"fmt"

	"furab-backend/services/audit-log-service/internal/model"
)

// AuditLogRepository defines the data access layer for audit logs.
type AuditLogRepository interface {
	Save(ctx context.Context, log *model.AuditLog) error
	GetByID(ctx context.Context, id string) (*model.AuditLog, error)
	Update(ctx context.Context, log *model.AuditLog) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, req model.SearchAuditLogRequest) ([]model.AuditLog, int, error)
}

type auditLogRepository struct {
	db *sql.DB
}

// NewAuditLogRepository creates a new instance of audit log repository.
func NewAuditLogRepository(db *sql.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Save(ctx context.Context, m *model.AuditLog) error {
	query := `INSERT INTO audit_logs (id, user_id, action, entity, entity_id, old_data, new_data, ip_address, user_agent, created_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query, m.ID, m.UserID, m.Action, m.Entity, m.EntityID, m.OldData, m.NewData, m.IPAddress, m.UserAgent, m.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save audit log: %w", err)
	}
	return nil
}

func (r *auditLogRepository) GetByID(ctx context.Context, id string) (*model.AuditLog, error) {
	query := `SELECT id, user_id, action, entity, entity_id, old_data, new_data, ip_address, user_agent, created_at FROM audit_logs WHERE id = $1`
	var m model.AuditLog
	err := r.db.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.UserID, &m.Action, &m.Entity, &m.EntityID, &m.OldData, &m.NewData, &m.IPAddress, &m.UserAgent, &m.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("audit log not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}
	return &m, nil
}

func (r *auditLogRepository) Update(ctx context.Context, m *model.AuditLog) error {
	query := `UPDATE audit_logs SET user_id = $1, action = $2, entity = $3, entity_id = $4, old_data = $5, new_data = $6 WHERE id = $7`
	_, err := r.db.ExecContext(ctx, query, m.UserID, m.Action, m.Entity, m.EntityID, m.OldData, m.NewData, m.ID)
	if err != nil {
		return fmt.Errorf("failed to update audit log: %w", err)
	}
	return nil
}

func (r *auditLogRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM audit_logs WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete audit log: %w", err)
	}
	return nil
}

func (r *auditLogRepository) Search(ctx context.Context, req model.SearchAuditLogRequest) ([]model.AuditLog, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if req.UserID != "" {
		where += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, req.UserID)
		argIdx++
	}

	if req.Action != "" {
		where += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, req.Action)
		argIdx++
	}

	if req.Entity != "" {
		where += fmt.Sprintf(" AND entity = $%d", argIdx)
		args = append(args, req.Entity)
		argIdx++
	}

	if !req.StartDate.IsZero() {
		where += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, req.StartDate)
		argIdx++
	}

	if !req.EndDate.IsZero() {
		where += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, req.EndDate)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", where)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := req.Offset

	searchQuery := fmt.Sprintf("SELECT id, user_id, action, entity, entity_id, old_data, new_data, ip_address, user_agent, created_at FROM audit_logs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search audit logs: %w", err)
	}
	defer rows.Close()

	logs := []model.AuditLog{}
	for rows.Next() {
		var m model.AuditLog
		if err := rows.Scan(&m.ID, &m.UserID, &m.Action, &m.Entity, &m.EntityID, &m.OldData, &m.NewData, &m.IPAddress, &m.UserAgent, &m.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, m)
	}

	return logs, total, nil
}
