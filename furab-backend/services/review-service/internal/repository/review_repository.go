package repository

import (
	"context"
	"database/sql"
	"fmt"

	"furab-backend/services/review-service/internal/model"
)

// ReviewRepository defines the data access layer for reviews.
type ReviewRepository interface {
	Save(ctx context.Context, review *model.Review) error
	GetByID(ctx context.Context, id string) (*model.Review, error)
	Update(ctx context.Context, review *model.Review) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, req model.SearchReviewRequest) ([]model.Review, int, error)
}

type reviewRepository struct {
	db *sql.DB
}

// NewReviewRepository creates a new instance of review repository.
func NewReviewRepository(db *sql.DB) ReviewRepository {
	return &reviewRepository{db: db}
}

func (r *reviewRepository) Save(ctx context.Context, m *model.Review) error {
	query := `INSERT INTO reviews (id, user_id, merchant_id, order_id, rating, comment, image_url, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query, m.ID, m.UserID, m.MerchantID, m.OrderID, m.Rating, m.Comment, m.ImageURL, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save review: %w", err)
	}
	return nil
}

func (r *reviewRepository) GetByID(ctx context.Context, id string) (*model.Review, error) {
	query := `SELECT id, user_id, merchant_id, order_id, rating, comment, image_url, created_at, updated_at FROM reviews WHERE id = $1`
	var m model.Review
	err := r.db.QueryRowContext(ctx, query, id).Scan(&m.ID, &m.UserID, &m.MerchantID, &m.OrderID, &m.Rating, &m.Comment, &m.ImageURL, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("review not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}
	return &m, nil
}

func (r *reviewRepository) Update(ctx context.Context, m *model.Review) error {
	query := `UPDATE reviews SET rating = $1, comment = $2, image_url = $3, updated_at = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, m.Rating, m.Comment, m.ImageURL, m.UpdatedAt, m.ID)
	if err != nil {
		return fmt.Errorf("failed to update review: %w", err)
	}
	return nil
}

func (r *reviewRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM reviews WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete review: %w", err)
	}
	return nil
}

func (r *reviewRepository) Search(ctx context.Context, req model.SearchReviewRequest) ([]model.Review, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if req.MerchantID != "" {
		where += fmt.Sprintf(" AND merchant_id = $%d", argIdx)
		args = append(args, req.MerchantID)
		argIdx++
	}

	if req.UserID != "" {
		where += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, req.UserID)
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM reviews %s", where)
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := req.Offset

	searchQuery := fmt.Sprintf("SELECT id, user_id, merchant_id, order_id, rating, comment, image_url, created_at, updated_at FROM reviews %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search reviews: %w", err)
	}
	defer rows.Close()

	reviews := []model.Review{}
	for rows.Next() {
		var m model.Review
		if err := rows.Scan(&m.ID, &m.UserID, &m.MerchantID, &m.OrderID, &m.Rating, &m.Comment, &m.ImageURL, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan review: %w", err)
		}
		reviews = append(reviews, m)
	}

	return reviews, total, nil
}
