package repository

import (
	"context"
	"database/sql"
	"fmt"

	"furab-backend/services/rating-service/internal/model"
)

// RatingRepository defines the data access layer for ratings.
type RatingRepository interface {
	Save(ctx context.Context, rating *model.Rating) error
	GetByID(ctx context.Context, id string) (*model.Rating, error)
	Update(ctx context.Context, rating *model.Rating) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, req model.SearchRatingRequest) ([]model.Rating, int, error)
}

type ratingRepository struct {
	db *sql.DB
}

// NewRatingRepository creates a new instance of rating repository.
func NewRatingRepository(db *sql.DB) RatingRepository {
	return &ratingRepository{db: db}
}

func (r *ratingRepository) Save(ctx context.Context, m *model.Rating) error {
	query := `INSERT INTO ratings (id, user_id, target_id, target_type, score, comment, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, m.ID, m.UserID, m.TargetID, m.TargetType, m.Score, m.Comment, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save rating: %w", err)
	}
	return nil
}

func (r *ratingRepository) GetByID(ctx context.Context, id string) (*model.Rating, error) {
	query := `SELECT id, user_id, target_id, target_type, score, comment, created_at, updated_at FROM ratings WHERE id = $1`
	var m model.Rating
	err := r.db.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.UserID, &m.TargetID, &m.TargetType, &m.Score, &m.Comment, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rating not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get rating: %w", err)
	}
	return &m, nil
}

func (r *ratingRepository) Update(ctx context.Context, m *model.Rating) error {
	query := `UPDATE ratings SET score = $1, comment = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, m.Score, m.Comment, m.UpdatedAt, m.ID)
	if err != nil {
		return fmt.Errorf("failed to update rating: %w", err)
	}
	return nil
}

func (r *ratingRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM ratings WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete rating: %w", err)
	}
	return nil
}

func (r *ratingRepository) Search(ctx context.Context, req model.SearchRatingRequest) ([]model.Rating, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if req.TargetID != "" {
		where += fmt.Sprintf(" AND target_id = $%d", argIdx)
		args = append(args, req.TargetID)
		argIdx++
	}

	if req.TargetType != "" {
		where += fmt.Sprintf(" AND target_type = $%d", argIdx)
		args = append(args, req.TargetType)
		argIdx++
	}

	if req.UserID != "" {
		where += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, req.UserID)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM ratings %s", where)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count ratings: %w", err)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := req.Offset

	searchQuery := fmt.Sprintf("SELECT id, user_id, target_id, target_type, score, comment, created_at, updated_at FROM ratings %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search ratings: %w", err)
	}
	defer rows.Close()

	ratings := []model.Rating{}
	for rows.Next() {
		var m model.Rating
		if err := rows.Scan(&m.ID, &m.UserID, &m.TargetID, &m.TargetType, &m.Score, &m.Comment, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan rating: %w", err)
		}
		ratings = append(ratings, m)
	}

	return ratings, total, nil
}
