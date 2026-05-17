package repository

import (
	"context"
	"database/sql"
	"fmt"

	"furab-backend/services/menu-service/internal/model"
)

// MenuRepository defines the data access layer for menus.
type MenuRepository interface {
	Save(ctx context.Context, menu *model.Menu) error
	GetByID(ctx context.Context, id string) (*model.Menu, error)
	Update(ctx context.Context, menu *model.Menu) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, req model.SearchMenuRequest) ([]model.Menu, int, error)
}

type menuRepository struct {
	db *sql.DB
}

// NewMenuRepository creates a new instance of menu repository.
func NewMenuRepository(db *sql.DB) MenuRepository {
	return &menuRepository{db: db}
}

func (r *menuRepository) Save(ctx context.Context, m *model.Menu) error {
	query := `INSERT INTO menus (id, merchant_id, name, description, price, category, image_url, is_available, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, query, m.ID, m.MerchantID, m.Name, m.Description, m.Price, m.Category, m.ImageURL, m.IsAvailable, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save menu: %w", err)
	}
	return nil
}

func (r *menuRepository) GetByID(ctx context.Context, id string) (*model.Menu, error) {
	query := `SELECT id, merchant_id, name, description, price, category, image_url, is_available, created_at, updated_at FROM menus WHERE id = $1`
	var m model.Menu
	err := r.db.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.MerchantID, &m.Name, &m.Description, &m.Price, &m.Category, &m.ImageURL, &m.IsAvailable, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("menu not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get menu: %w", err)
	}
	return &m, nil
}

func (r *menuRepository) Update(ctx context.Context, m *model.Menu) error {
	query := `UPDATE menus SET name = $1, description = $2, price = $3, category = $4, image_url = $5, is_available = $6, updated_at = $7 WHERE id = $8`
	_, err := r.db.ExecContext(ctx, query, m.Name, m.Description, m.Price, m.Category, m.ImageURL, m.IsAvailable, m.UpdatedAt, m.ID)
	if err != nil {
		return fmt.Errorf("failed to update menu: %w", err)
	}
	return nil
}

func (r *menuRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM menus WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete menu: %w", err)
	}
	return nil
}

func (r *menuRepository) Search(ctx context.Context, req model.SearchMenuRequest) ([]model.Menu, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if req.MerchantID != "" {
		where += fmt.Sprintf(" AND merchant_id = $%d", argIdx)
		args = append(args, req.MerchantID)
		argIdx++
	}

	if req.Query != "" {
		where += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+req.Query+"%")
		argIdx++
	}

	if req.Category != "" {
		where += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, req.Category)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM menus %s", where)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count menus: %w", err)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := req.Offset

	searchQuery := fmt.Sprintf("SELECT id, merchant_id, name, description, price, category, image_url, is_available, created_at, updated_at FROM menus %s LIMIT $%d OFFSET $%d",
		where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search menus: %w", err)
	}
	defer rows.Close()

	menus := []model.Menu{}
	for rows.Next() {
		var m model.Menu
		if err := rows.Scan(&m.ID, &m.MerchantID, &m.Name, &m.Description, &m.Price, &m.Category, &m.ImageURL, &m.IsAvailable, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan menu: %w", err)
		}
		menus = append(menus, m)
	}

	return menus, total, nil
}
