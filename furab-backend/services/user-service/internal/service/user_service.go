// Package service implements the business logic for user management.
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"furab-backend/services/user-service/internal/model"
	"furab-backend/services/user-service/internal/repository"
)

// Common service errors.
var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrUserNotFound   = errors.New("user not found")
)

// UserService defines the interface for user business logic.
// This interface is used for dependency injection in handlers and can be mocked in tests.
type UserService interface {
	// CreateUser creates a new user account.
	CreateUser(ctx context.Context, req *model.CreateUserRequest) (*model.CreateUserResponse, error)

	// GetUser retrieves a user by their ID.
	GetUser(ctx context.Context, userID string) (*model.User, error)

	// UpdateUser updates an existing user's information.
	UpdateUser(ctx context.Context, userID string, req *model.UpdateUserRequest) (*model.MessageResponse, error)

	// DeactivateUser deactivates a user account.
	DeactivateUser(ctx context.Context, userID string) (*model.MessageResponse, error)
}

// userServiceImpl is the concrete implementation of UserService.
type userServiceImpl struct {
	repo repository.UserRepository
}

// NewUserService creates a new UserService with the given dependencies.
func NewUserService(repo repository.UserRepository) UserService {
	return &userServiceImpl{
		repo: repo,
	}
}

// CreateUser creates a new user account.
func (s *userServiceImpl) CreateUser(ctx context.Context, req *model.CreateUserRequest) (*model.CreateUserResponse, error) {
	// Validate request
	if req == nil {
		return nil, ErrInvalidRequest
	}

	// Normalize input
	req.UserID = strings.TrimSpace(req.UserID)
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Email = strings.TrimSpace(req.Email)

	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Create user
	now := time.Now().UTC()
	user := &model.User{
		UserID:    req.UserID,
		Name:      req.Name,
		Phone:     req.Phone,
		Email:     req.Email,
		Status:    model.UserStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save to database
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &model.CreateUserResponse{
		UserID:  user.UserID,
		Message: "user created successfully",
	}, nil
}

// GetUser retrieves a user by their ID.
func (s *userServiceImpl) GetUser(ctx context.Context, userID string) (*model.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidRequest
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// UpdateUser updates an existing user's information.
func (s *userServiceImpl) UpdateUser(ctx context.Context, userID string, req *model.UpdateUserRequest) (*model.MessageResponse, error) {
	// Validate inputs
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidRequest
	}

	if req == nil {
		return nil, ErrInvalidRequest
	}

	// Normalize input
	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Email = strings.TrimSpace(req.Email)

	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get existing user
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Update fields
	user.Name = req.Name
	user.Phone = req.Phone
	user.Email = req.Email
	user.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &model.MessageResponse{
		Message: "user updated successfully",
	}, nil
}

// DeactivateUser deactivates a user account.
func (s *userServiceImpl) DeactivateUser(ctx context.Context, userID string) (*model.MessageResponse, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidRequest
	}

	// Get existing user
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Deactivate
	user.Status = model.UserStatusInactive
	user.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &model.MessageResponse{
		Message: "user deactivated successfully",
	}, nil
}