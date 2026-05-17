// Package service implements the business logic for OTP management.
package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"furab-backend/services/otp-service/internal/model"
	"furab-backend/services/otp-service/internal/repository"

	"github.com/google/uuid"
)

// OTP expiration duration.
const otpExpirationMinutes = 5

// Common service errors.
var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrOTPNotFound    = errors.New("otp not found")
	ErrOTPExpired     = errors.New("otp expired")
	ErrOTPInvalid     = errors.New("otp code invalid")
)

// OTPService defines the interface for OTP business logic.
// This interface is used for dependency injection in handlers and can be mocked in tests.
type OTPService interface {
	// GenerateOTP generates a new OTP for the given target (phone/email).
	GenerateOTP(ctx context.Context, req *model.GenerateOTPRequest) (*model.GenerateOTPResponse, error)

	// VerifyOTP verifies an OTP code for the given target.
	VerifyOTP(ctx context.Context, req *model.VerifyOTPRequest) (*model.VerifyOTPResponse, error)
}

// otpServiceImpl is the concrete implementation of OTPService.
type otpServiceImpl struct {
	repo repository.OTPRepository
}

// NewOTPService creates a new OTPService with the given dependencies.
func NewOTPService(repo repository.OTPRepository) OTPService {
	return &otpServiceImpl{
		repo: repo,
	}
}

// GenerateOTP generates a new OTP for the given target.
// Logic: generate random 6-digit code, set 5-minute expiry, save to repository.
func (s *otpServiceImpl) GenerateOTP(ctx context.Context, req *model.GenerateOTPRequest) (*model.GenerateOTPResponse, error) {
	// Validate request
	if req == nil {
		return nil, ErrInvalidRequest
	}

	// Normalize input
	req.Target = strings.TrimSpace(req.Target)

	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Generate random 6-digit OTP code
	code, err := generateRandomOTP(6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate otp code: %w", err)
	}

	// Create OTP entity
	now := time.Now().UTC()
	otp := &model.OTP{
		OTPID:     uuid.New().String(),
		Target:    req.Target,
		OTPCode:   code,
		ExpiredAt: now.Add(otpExpirationMinutes * time.Minute),
		CreatedAt: now,
	}

	// Save to repository
	if err := s.repo.Save(ctx, otp); err != nil {
		return nil, err
	}

	return &model.GenerateOTPResponse{
		Status:  "success",
		Message: "OTP generated successfully",
	}, nil
}

// VerifyOTP verifies an OTP code for the given target.
// Logic: lookup by target, check existence → check code match → check expiry → delete if valid.
func (s *otpServiceImpl) VerifyOTP(ctx context.Context, req *model.VerifyOTPRequest) (*model.VerifyOTPResponse, error) {
	// Validate request
	if req == nil {
		return nil, ErrInvalidRequest
	}

	// Normalize input
	req.Target = strings.TrimSpace(req.Target)
	req.OTPCode = strings.TrimSpace(req.OTPCode)

	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Retrieve OTP from repository
	otp, err := s.repo.GetByTarget(ctx, req.Target)
	if err != nil {
		if errors.Is(err, repository.ErrOTPNotFound) {
			return nil, ErrOTPNotFound
		}
		return nil, err
	}

	// Check if OTP code matches
	if otp.OTPCode != req.OTPCode {
		return nil, ErrOTPInvalid
	}

	// Check if OTP is expired
	if otp.IsExpired() {
		return nil, ErrOTPExpired
	}

	// OTP is valid — delete it (one-time use)
	_ = s.repo.Delete(ctx, otp.OTPID)

	return &model.VerifyOTPResponse{
		Status:  "valid",
		Message: "OTP verified successfully",
	}, nil
}

// generateRandomOTP generates a cryptographically secure random numeric OTP of the given length.
func generateRandomOTP(length int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digits[b[i]%10]
	}
	return string(b), nil
}