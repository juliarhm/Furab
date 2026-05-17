package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/rating-service/internal/model"
	"furab-backend/services/rating-service/internal/repository"
	"github.com/google/uuid"
)

// RatingService defines the business logic for ratings.
type RatingService interface {
	CreateRating(ctx context.Context, m *model.Rating) error
	GetRating(ctx context.Context, id string) (*model.Rating, error)
	UpdateRating(ctx context.Context, m *model.Rating) error
	DeleteRating(ctx context.Context, id string) error
	SearchRatings(ctx context.Context, req model.SearchRatingRequest) ([]model.Rating, int, error)
}

type ratingService struct {
	repo repository.RatingRepository
}

// NewRatingService creates a new instance of rating service.
func NewRatingService(repo repository.RatingRepository) RatingService {
	return &ratingService{repo: repo}
}

func (s *ratingService) CreateRating(ctx context.Context, m *model.Rating) error {
	if m.UserID == "" {
		return errors.New("user id is required")
	}
	if m.TargetID == "" {
		return errors.New("target id is required")
	}
	if m.TargetType == "" {
		return errors.New("target type is required")
	}
	if m.Score < 1 || m.Score > 5 {
		return errors.New("score must be between 1 and 5")
	}

	m.ID = uuid.New().String()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()

	return s.repo.Save(ctx, m)
}

func (s *ratingService) GetRating(ctx context.Context, id string) (*model.Rating, error) {
	if id == "" {
		return nil, errors.New("rating id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *ratingService) UpdateRating(ctx context.Context, m *model.Rating) error {
	if m.ID == "" {
		return errors.New("rating id is required")
	}

	existing, err := s.repo.GetByID(ctx, m.ID)
	if err != nil {
		return err
	}

	if m.Score < 1 || m.Score > 5 {
		return errors.New("score must be between 1 and 5")
	}

	existing.Score = m.Score
	existing.Comment = m.Comment
	existing.UpdatedAt = time.Now()

	return s.repo.Update(ctx, existing)
}

func (s *ratingService) DeleteRating(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("rating id is required")
	}
	return s.repo.Delete(ctx, id)
}

func (s *ratingService) SearchRatings(ctx context.Context, req model.SearchRatingRequest) ([]model.Rating, int, error) {
	return s.repo.Search(ctx, req)
}
