package unit

import (
	"context"
	"errors"
	"testing"

	"furab-backend/services/auth-service/internal/model"
	"furab-backend/services/auth-service/internal/service"
)

type MockUserService struct {
	CreateUserFunc func(ctx context.Context, contact string) error
	GetUserFunc    func(ctx context.Context, contact string) (*model.User, error)
}
func (m *MockUserService) CreateUser(ctx context.Context, contact string) error { return m.CreateUserFunc(ctx, contact) }
func (m *MockUserService) GetUser(ctx context.Context, contact string) (*model.User, error) { return m.GetUserFunc(ctx, contact) }

type MockOTPService struct {
	GenerateOTPFunc func(ctx context.Context, contact string) error
	VerifyOTPFunc   func(ctx context.Context, contact, otpCode string) (bool, error)
}
func (m *MockOTPService) GenerateOTP(ctx context.Context, contact string) error { return m.GenerateOTPFunc(ctx, contact) }
func (m *MockOTPService) VerifyOTP(ctx context.Context, contact, otpCode string) (bool, error) { return m.VerifyOTPFunc(ctx, contact, otpCode) }

type MockTokenGenerator struct {
	GenerateTokenFunc func(userID string) (string, error)
	ValidateTokenFunc func(token string) (bool, error)
}
func (m *MockTokenGenerator) GenerateToken(userID string) (string, error) { return m.GenerateTokenFunc(userID) }
func (m *MockTokenGenerator) ValidateToken(token string) (bool, error) { return m.ValidateTokenFunc(token) }

func TestAuthService_Register(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUser := &MockUserService{
			CreateUserFunc: func(ctx context.Context, contact string) error { return nil },
		}
		mockOTP := &MockOTPService{
			GenerateOTPFunc: func(ctx context.Context, contact string) error { return nil },
		}
		svc := service.NewAuthService(mockUser, mockOTP, &MockTokenGenerator{})

		res, err := svc.Register(context.Background(), "erv@mail.com")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.Status != "success" {
			t.Errorf("Expected status success")
		}
	})

	t.Run("Error - Invalid Contact", func(t *testing.T) {
		svc := service.NewAuthService(&MockUserService{}, &MockOTPService{}, &MockTokenGenerator{})
		_, err := svc.Register(context.Background(), "invalid-contact")
		if err == nil {
			t.Fatalf("Expected validation error")
		}
	})
}

func TestAuthService_RequestOTP(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockOTP := &MockOTPService{
			GenerateOTPFunc: func(ctx context.Context, contact string) error { return nil },
		}
		svc := service.NewAuthService(&MockUserService{}, mockOTP, &MockTokenGenerator{})

		res, err := svc.RequestOTP(context.Background(), "erv@mail.com")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.Status != "success" {
			t.Errorf("Expected status success")
		}
	})
}

func TestAuthService_VerifyOTP(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockUser := &MockUserService{
			GetUserFunc: func(ctx context.Context, contact string) (*model.User, error) {
				return &model.User{ID: "1"}, nil
			},
		}
		mockOTP := &MockOTPService{
			VerifyOTPFunc: func(ctx context.Context, contact, otpCode string) (bool, error) {
				return true, nil
			},
		}
		mockToken := &MockTokenGenerator{
			GenerateTokenFunc: func(userID string) (string, error) {
				return "token123", nil
			},
		}
		svc := service.NewAuthService(mockUser, mockOTP, mockToken)

		res, err := svc.VerifyOTP(context.Background(), "erv@mail.com", "123456")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.AccessToken != "token123" {
			t.Errorf("Expected token123, got %s", res.AccessToken)
		}
	})

	t.Run("Error - Invalid OTP", func(t *testing.T) {
		mockOTP := &MockOTPService{
			VerifyOTPFunc: func(ctx context.Context, contact, otpCode string) (bool, error) {
				return false, nil
			},
		}
		svc := service.NewAuthService(&MockUserService{}, mockOTP, &MockTokenGenerator{})

		_, err := svc.VerifyOTP(context.Background(), "erv@mail.com", "wrong")
		if err == nil || !errors.Is(err, service.ErrOTPInvalid) {
			t.Fatalf("Expected invalid OTP error")
		}
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	t.Run("Valid Token", func(t *testing.T) {
		mockToken := &MockTokenGenerator{
			ValidateTokenFunc: func(token string) (bool, error) {
				return true, nil
			},
		}
		svc := service.NewAuthService(&MockUserService{}, &MockOTPService{}, mockToken)

		res, err := svc.ValidateToken(context.Background(), "token123")
		if err != nil {
			t.Fatalf("Expected no error")
		}
		if res.Status != "valid" {
			t.Errorf("Expected status valid")
		}
	})

	t.Run("Invalid Token", func(t *testing.T) {
		mockToken := &MockTokenGenerator{
			ValidateTokenFunc: func(token string) (bool, error) {
				return false, nil
			},
		}
		svc := service.NewAuthService(&MockUserService{}, &MockOTPService{}, mockToken)

		res, err := svc.ValidateToken(context.Background(), "wrong")
		if err != nil {
			t.Fatalf("Expected no error")
		}
		if res.Status != "invalid" {
			t.Errorf("Expected status invalid")
		}
	})
}
