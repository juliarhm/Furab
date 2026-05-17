package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/menu-service/internal/model"
	"furab-backend/services/menu-service/internal/repository"
	"github.com/google/uuid"
)

// MenuService defines the business logic for menus.
type MenuService interface {
	CreateMenu(ctx context.Context, m *model.Menu) error
	GetMenu(ctx context.Context, id string) (*model.Menu, error)
	UpdateMenu(ctx context.Context, m *model.Menu) error
	DeleteMenu(ctx context.Context, id string) error
	SearchMenus(ctx context.Context, req model.SearchMenuRequest) ([]model.Menu, int, error)
}

type menuService struct {
	repo repository.MenuRepository
}

// NewMenuService creates a new instance of menu service.
func NewMenuService(repo repository.MenuRepository) MenuService {
	return &menuService{repo: repo}
}

func (s *menuService) CreateMenu(ctx context.Context, m *model.Menu) error {
	if m.MerchantID == "" {
		return errors.New("merchant id is required")
	}
	if m.Name == "" {
		return errors.New("menu name is required")
	}
	if m.Price <= 0 {
		return errors.New("price must be greater than zero")
	}

	m.ID = uuid.New().String()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	m.IsAvailable = true

	return s.repo.Save(ctx, m)
}

func (s *menuService) GetMenu(ctx context.Context, id string) (*model.Menu, error) {
	if id == "" {
		return nil, errors.New("menu id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *menuService) UpdateMenu(ctx context.Context, m *model.Menu) error {
	if m.ID == "" {
		return errors.New("menu id is required")
	}

	existing, err := s.repo.GetByID(ctx, m.ID)
	if err != nil {
		return err
	}

	if m.Name != "" {
		existing.Name = m.Name
	}
	if m.Description != "" {
		existing.Description = m.Description
	}
	if m.Price > 0 {
		existing.Price = m.Price
	}
	if m.Category != "" {
		existing.Category = m.Category
	}
	if m.ImageURL != "" {
		existing.ImageURL = m.ImageURL
	}
	existing.IsAvailable = m.IsAvailable
	existing.UpdatedAt = time.Now()

	return s.repo.Update(ctx, existing)
}

func (s *menuService) DeleteMenu(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("menu id is required")
	}
	return s.repo.Delete(ctx, id)
}

func (s *menuService) SearchMenus(ctx context.Context, req model.SearchMenuRequest) ([]model.Menu, int, error) {
	return s.repo.Search(ctx, req)
}
