package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/user-service/internal/model"
	"furab-backend/services/user-service/internal/repository"
	"furab-backend/services/user-service/internal/service"
)

// Manual mock for UserRepository
type MockUserRepository struct {
	CreateFunc  func(ctx context.Context, user *model.User) error
	GetByIDFunc func(ctx context.Context, id string) (*model.User, error)
	UpdateFunc  func(ctx context.Context, user *model.User) error
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	return m.CreateFunc(ctx, user)
}
func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	return m.GetByIDFunc(ctx, id)
}
func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	return m.UpdateFunc(ctx, user)
}

func TestUserService_CreateUser(t *testing.T) {
	t.Run("Success - User berhasil dibuat", func(t *testing.T) {
		repo := &MockUserRepository{
			CreateFunc: func(ctx context.Context, user *model.User) error {
				if user.UserID != "1" {
					t.Errorf("Expected UserID 1, got %s", user.UserID)
				}
				return nil
			},
		}
		svc := service.NewUserService(repo)

		req := &model.CreateUserRequest{
			UserID: "1",
			Name:   "Erv",
			Email:  "erv@mail.com",
			Phone:  "08123456",
		}

		res, err := svc.CreateUser(context.Background(), req)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.UserID != "1" {
			t.Errorf("Expected response UserID 1, got %s", res.UserID)
		}
	})

	t.Run("Error - Email kosong", func(t *testing.T) {
		repo := &MockUserRepository{
			CreateFunc: func(ctx context.Context, user *model.User) error {
				t.Fatal("Repository Create() should not be called")
				return nil
			},
		}
		svc := service.NewUserService(repo)

		req := &model.CreateUserRequest{
			UserID: "1",
			Name:   "Erv",
			Email:  "", // Kosong
			Phone:  "08123456",
		}

		res, err := svc.CreateUser(context.Background(), req)

		if err == nil {
			t.Fatalf("Expected error, got none")
		}
		if res != nil {
			t.Errorf("Expected nil response, got %v", res)
		}
	})

	t.Run("Error - Data tidak lengkap", func(t *testing.T) {
		repo := &MockUserRepository{
			CreateFunc: func(ctx context.Context, user *model.User) error {
				t.Fatal("Repository Create() should not be called")
				return nil
			},
		}
		svc := service.NewUserService(repo)

		req := &model.CreateUserRequest{
			UserID: "1",
			Name:   "", // Kosong
			Email:  "erv@mail.com",
			Phone:  "08123",
		}

		res, err := svc.CreateUser(context.Background(), req)

		if err == nil {
			t.Fatal("Expected validation error, got none")
		}
		if res != nil {
			t.Errorf("Expected nil response, got %v", res)
		}
	})
}

func TestUserService_GetUser(t *testing.T) {
	t.Run("Success - User ditemukan", func(t *testing.T) {
		repo := &MockUserRepository{
			GetByIDFunc: func(ctx context.Context, userID string) (*model.User, error) {
				if userID != "1" {
					t.Errorf("Expected UserID 1, got %s", userID)
				}
				return &model.User{UserID: "1", Name: "Erv"}, nil
			},
		}
		svc := service.NewUserService(repo)

		user, err := svc.GetUser(context.Background(), "1")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if user == nil || user.UserID != "1" {
			t.Errorf("Expected User 1, got %v", user)
		}
	})

	t.Run("Error - User tidak ditemukan", func(t *testing.T) {
		repo := &MockUserRepository{
			GetByIDFunc: func(ctx context.Context, userID string) (*model.User, error) {
				return nil, repository.ErrUserNotFound
			},
		}
		svc := service.NewUserService(repo)

		user, err := svc.GetUser(context.Background(), "99")

		if err == nil || !errors.Is(err, service.ErrUserNotFound) {
			t.Fatalf("Expected 'user not found' error, got %v", err)
		}
		if user != nil {
			t.Errorf("Expected nil user, got %v", user)
		}
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	t.Run("Success - Data berhasil diupdate", func(t *testing.T) {
		repo := &MockUserRepository{
			GetByIDFunc: func(ctx context.Context, userID string) (*model.User, error) {
				return &model.User{UserID: "1", Name: "Erv", Email: "erv@mail.com"}, nil
			},
			UpdateFunc: func(ctx context.Context, user *model.User) error {
				if user.Name != "Erv Update" || user.Email != "erv_update@mail.com" {
					t.Errorf("Updated user data mismatch, got: %+v", user)
				}
				return nil
			},
		}
		svc := service.NewUserService(repo)

		req := &model.UpdateUserRequest{
			Name:  "Erv Update",
			Email: "erv_update@mail.com",
			Phone: "08123456",
		}

		res, err := svc.UpdateUser(context.Background(), "1", req)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res == nil {
			t.Fatalf("Expected response")
		}
	})

	t.Run("Error - User tidak ditemukan", func(t *testing.T) {
		repo := &MockUserRepository{
			GetByIDFunc: func(ctx context.Context, userID string) (*model.User, error) {
				return nil, repository.ErrUserNotFound
			},
		}
		svc := service.NewUserService(repo)

		req := &model.UpdateUserRequest{
			Name:  "Erv Update",
			Email: "erv_update@mail.com",
			Phone: "08123456",
		}

		_, err := svc.UpdateUser(context.Background(), "99", req)

		if err == nil || !errors.Is(err, service.ErrUserNotFound) {
			t.Fatalf("Expected 'user not found' error, got %v", err)
		}
	})
}

func TestUserService_DeactivateUser(t *testing.T) {
	t.Run("Success - User dinonaktifkan", func(t *testing.T) {
		repo := &MockUserRepository{
			GetByIDFunc: func(ctx context.Context, userID string) (*model.User, error) {
				return &model.User{UserID: "1", Status: model.UserStatusActive}, nil
			},
			UpdateFunc: func(ctx context.Context, user *model.User) error {
				if user.Status != model.UserStatusInactive {
					t.Errorf("Expected user status to be inactive")
				}
				return nil
			},
		}
		svc := service.NewUserService(repo)

		res, err := svc.DeactivateUser(context.Background(), "1")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res == nil {
			t.Fatalf("Expected response")
		}
	})
}
