package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"furab-backend/services/otp-service/internal/model"
	"furab-backend/services/otp-service/internal/service"
)

type MockOTPRepository struct {
	SaveFunc        func(ctx context.Context, otp *model.OTP) error
	GetByTargetFunc func(ctx context.Context, target string) (*model.OTP, error)
	DeleteFunc      func(ctx context.Context, otpID string) error
}

func (m *MockOTPRepository) Save(ctx context.Context, otp *model.OTP) error {
	return m.SaveFunc(ctx, otp)
}
func (m *MockOTPRepository) GetByTarget(ctx context.Context, target string) (*model.OTP, error) {
	return m.GetByTargetFunc(ctx, target)
}
func (m *MockOTPRepository) Delete(ctx context.Context, otpID string) error {
	return m.DeleteFunc(ctx, otpID)
}

func TestOTPService_GenerateOTP(t *testing.T) {
	t.Run("Success - input valid", func(t *testing.T) {
		repo := &MockOTPRepository{
			SaveFunc: func(ctx context.Context, otp *model.OTP) error {
				if otp.Target != "08123456" {
					t.Errorf("Expected target 08123456, got %s", otp.Target)
				}
				if len(otp.OTPCode) != 6 {
					t.Errorf("Expected 6 digit OTP")
				}
				return nil
			},
		}
		svc := service.NewOTPService(repo)

		req := &model.GenerateOTPRequest{
			Target: "08123456",
		}
		res, err := svc.GenerateOTP(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.Status != "success" {
			t.Errorf("Expected status success, got %s", res.Status)
		}
	})

	t.Run("Error - input kosong", func(t *testing.T) {
		repo := &MockOTPRepository{}
		svc := service.NewOTPService(repo)

		req := &model.GenerateOTPRequest{
			Target: "",
		}
		res, err := svc.GenerateOTP(context.Background(), req)
		if err == nil {
			t.Fatalf("Expected validation error, got none")
		}
		if res != nil {
			t.Errorf("Expected nil response")
		}
	})
}

func TestOTPService_VerifyOTP(t *testing.T) {
	t.Run("Success - OTP valid", func(t *testing.T) {
		repo := &MockOTPRepository{
			GetByTargetFunc: func(ctx context.Context, target string) (*model.OTP, error) {
				return &model.OTP{
					OTPID:     "1",
					Target:    "08123456",
					OTPCode:   "123456",
					ExpiredAt: time.Now().Add(5 * time.Minute),
				}, nil
			},
			DeleteFunc: func(ctx context.Context, otpID string) error {
				if otpID != "1" {
					t.Errorf("Expected to delete OTP ID 1")
				}
				return nil
			},
		}
		svc := service.NewOTPService(repo)

		req := &model.VerifyOTPRequest{
			Target:  "08123456",
			OTPCode: "123456",
		}
		res, err := svc.VerifyOTP(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if res.Status != "valid" {
			t.Errorf("Expected valid status, got %s", res.Status)
		}
	})

	t.Run("Error - OTP tidak valid", func(t *testing.T) {
		repo := &MockOTPRepository{
			GetByTargetFunc: func(ctx context.Context, target string) (*model.OTP, error) {
				return &model.OTP{
					OTPID:     "1",
					Target:    "08123456",
					OTPCode:   "123456",
					ExpiredAt: time.Now().Add(5 * time.Minute),
				}, nil
			},
		}
		svc := service.NewOTPService(repo)

		req := &model.VerifyOTPRequest{
			Target:  "08123456",
			OTPCode: "999999", // wrong code
		}
		_, err := svc.VerifyOTP(context.Background(), req)
		if err == nil || !errors.Is(err, service.ErrOTPInvalid) {
			t.Fatalf("Expected invalid otp error, got %v", err)
		}
	})

	t.Run("Error - OTP expired", func(t *testing.T) {
		repo := &MockOTPRepository{
			GetByTargetFunc: func(ctx context.Context, target string) (*model.OTP, error) {
				return &model.OTP{
					OTPID:     "1",
					Target:    "08123456",
					OTPCode:   "123456",
					ExpiredAt: time.Now().Add(-5 * time.Minute), // expired
				}, nil
			},
		}
		svc := service.NewOTPService(repo)

		req := &model.VerifyOTPRequest{
			Target:  "08123456",
			OTPCode: "123456",
		}
		_, err := svc.VerifyOTP(context.Background(), req)
		if err == nil || !errors.Is(err, service.ErrOTPExpired) {
			t.Fatalf("Expected expired otp error, got %v", err)
		}
	})
}