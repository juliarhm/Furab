package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/merchant-service/internal/model"
	"furab-backend/services/merchant-service/internal/repository"
	"github.com/google/uuid"
)

// MerchantService defines the business logic for merchants.
type MerchantService interface {
	CreateMerchant(ctx context.Context, m *model.Merchant) error
	GetMerchant(ctx context.Context, id string) (*model.Merchant, error)
	UpdateMerchant(ctx context.Context, m *model.Merchant) error
	DeleteMerchant(ctx context.Context, id string) error
	SearchMerchants(ctx context.Context, req model.SearchMerchantRequest) ([]model.Merchant, int, error)
}

type merchantService struct {
	repo repository.MerchantRepository
}

// NewMerchantService creates a new instance of merchant service.
func NewMerchantService(repo repository.MerchantRepository) MerchantService {
	return &merchantService{repo: repo}
}

func (s *merchantService) CreateMerchant(ctx context.Context, m *model.Merchant) error {
	if m.Name == "" {
		return errors.New("merchant name is required")
	}
	if m.Address == "" {
		return errors.New("merchant address is required")
	}

	m.ID = uuid.New().String()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	m.Rating = 0
	m.IsOpen = true

	return s.repo.Save(ctx, m)
}

func (s *merchantService) GetMerchant(ctx context.Context, id string) (*model.Merchant, error) {
	if id == "" {
		return nil, errors.New("merchant id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *merchantService) UpdateMerchant(ctx context.Context, m *model.Merchant) error {
	if m.ID == "" {
		return errors.New("merchant id is required")
	}

	existing, err := s.repo.GetByID(ctx, m.ID)
	if err != nil {
		return err
	}

	if m.Name != "" {
		existing.Name = m.Name
	}
	if m.Address != "" {
		existing.Address = m.Address
	}
	if m.Description != "" {
		existing.Description = m.Description
	}
	existing.IsOpen = m.IsOpen
	existing.UpdatedAt = time.Now()

	return s.repo.Update(ctx, existing)
}

func (s *merchantService) DeleteMerchant(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("merchant id is required")
	}
	return s.repo.Delete(ctx, id)
}

func (s *merchantService) SearchMerchants(ctx context.Context, req model.SearchMerchantRequest) ([]model.Merchant, int, error) {
	return s.repo.Search(ctx, req)
}
