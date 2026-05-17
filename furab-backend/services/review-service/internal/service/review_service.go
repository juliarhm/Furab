package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/review-service/internal/model"
	"furab-backend/services/review-service/internal/repository"
	"github.com/google/uuid"
)

// ReviewService defines the business logic for reviews.
type ReviewService interface {
	CreateReview(ctx context.Context, m *model.Review) error
	GetReview(ctx context.Context, id string) (*model.Review, error)
	UpdateReview(ctx context.Context, m *model.Review) error
	DeleteReview(ctx context.Context, id string) error
	SearchReviews(ctx context.Context, req model.SearchReviewRequest) ([]model.Review, int, error)
}

type reviewService struct {
	repo repository.ReviewRepository
}

// NewReviewService creates a new instance of review service.
func NewReviewService(repo repository.ReviewRepository) ReviewService {
	return &reviewService{repo: repo}
}

func (s *reviewService) CreateReview(ctx context.Context, m *model.Review) error {
	if m.UserID == "" {
		return errors.New("user id is required")
	}
	if m.MerchantID == "" {
		return errors.New("merchant id is required")
	}
	if m.OrderID == "" {
		return errors.New("order id is required")
	}
	if m.Rating < 1 || m.Rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}

	m.ID = uuid.New().String()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()

	return s.repo.Save(ctx, m)
}

func (s *reviewService) GetReview(ctx context.Context, id string) (*model.Review, error) {
	if id == "" {
		return nil, errors.New("review id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *reviewService) UpdateReview(ctx context.Context, m *model.Review) error {
	if m.ID == "" {
		return errors.New("review id is required")
	}

	existing, err := s.repo.GetByID(ctx, m.ID)
	if err != nil {
		return err
	}

	if m.Rating < 1 || m.Rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}

	existing.Rating = m.Rating
	existing.Comment = m.Comment
	existing.ImageURL = m.ImageURL
	existing.UpdatedAt = time.Now()

	return s.repo.Update(ctx, existing)
}

func (s *reviewService) DeleteReview(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("review id is required")
	}
	return s.repo.Delete(ctx, id)
}

func (s *reviewService) SearchReviews(ctx context.Context, req model.SearchReviewRequest) ([]model.Review, int, error) {
	return s.repo.Search(ctx, req)
}
