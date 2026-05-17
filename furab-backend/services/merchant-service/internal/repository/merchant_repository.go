package repository

import (
	"context"
	"database/sql"
	"fmt"

	"furab-backend/services/merchant-service/internal/model"
)

// MerchantRepository defines the data access layer for merchants.
type MerchantRepository interface {
	Save(ctx context.Context, merchant *model.Merchant) error
	GetByID(ctx context.Context, id string) (*model.Merchant, error)
	Update(ctx context.Context, merchant *model.Merchant) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, req model.SearchMerchantRequest) ([]model.Merchant, int, error)
}

type merchantRepository struct {
	db *sql.DB
}

// NewMerchantRepository creates a new instance of merchant repository.
func NewMerchantRepository(db *sql.DB) MerchantRepository {
	return &merchantRepository{db: db}
}

func (r *merchantRepository) Save(ctx context.Context, m *model.Merchant) error {
	query := `INSERT INTO merchants (id, name, address, description, rating, is_open, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, m.ID, m.Name, m.Address, m.Description, m.Rating, m.IsOpen, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save merchant: %w", err)
	}
	return nil
}

func (r *merchantRepository) GetByID(ctx context.Context, id string) (*model.Merchant, error) {
	query := `SELECT id, name, address, description, rating, is_open, created_at, updated_at FROM merchants WHERE id = $1`
	var m model.Merchant
	err := r.db.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.Name, &m.Address, &m.Description, &m.Rating, &m.IsOpen, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("merchant not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get merchant: %w", err)
	}
	return &m, nil
}

func (r *merchantRepository) Update(ctx context.Context, m *model.Merchant) error {
	query := `UPDATE merchants SET name = $1, address = $2, description = $3, rating = $4, is_open = $5, updated_at = $6 WHERE id = $7`
	_, err := r.db.ExecContext(ctx, query, m.Name, m.Address, m.Description, m.Rating, m.IsOpen, m.UpdatedAt, m.ID)
	if err != nil {
		return fmt.Errorf("failed to update merchant: %w", err)
	}
	return nil
}

func (r *merchantRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM merchants WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete merchant: %w", err)
	}
	return nil
}

func (r *merchantRepository) Search(ctx context.Context, req model.SearchMerchantRequest) ([]model.Merchant, int, error) {
	where := ""
	args := []interface{}{}
	if req.Query != "" {
		where = "WHERE name ILIKE $1 OR description ILIKE $1"
		args = append(args, "%"+req.Query+"%")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM merchants %s", where)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count merchants: %w", err)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := req.Offset

	searchQuery := fmt.Sprintf("SELECT id, name, address, description, rating, is_open, created_at, updated_at FROM merchants %s LIMIT $%d OFFSET $%d",
		where, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search merchants: %w", err)
	}
	defer rows.Close()

	merchants := []model.Merchant{}
	for rows.Next() {
		var m model.Merchant
		if err := rows.Scan(&m.ID, &m.Name, &m.Address, &m.Description, &m.Rating, &m.IsOpen, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan merchant: %w", err)
		}
		merchants = append(merchants, m)
	}

	return merchants, total, nil
}
