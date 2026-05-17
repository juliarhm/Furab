// Package service implements the business logic for auth-service.
package service

import (
	"context"
	"errors"
	"strings"

	"furab-backend/services/auth-service/internal/model"
)

// Common service errors.
var (
	ErrContactRequired      = errors.New("phone/email required")
	ErrContactInvalidFormat = errors.New("phone/email format invalid")
	ErrInputRequired        = errors.New("input required")
	ErrOTPInvalid           = errors.New("OTP invalid")
	ErrUserNotFound         = errors.New("user not found")
	ErrUserIDMissing        = errors.New("user id missing")
)

// UserService defines the interface for interacting with user data.
type UserService interface {
	CreateUser(ctx context.Context, contact string) error
	GetUser(ctx context.Context, contact string) (*model.User, error)
}

// OTPService defines the interface for OTP operations.
type OTPService interface {
	GenerateOTP(ctx context.Context, contact string) error
	VerifyOTP(ctx context.Context, contact, otpCode string) (bool, error)
}

// TokenGenerator defines the interface for token operations.
type TokenGenerator interface {
	GenerateToken(userID string) (string, error)
	ValidateToken(token string) (bool, error)
}

// AuthService defines the interface for auth-service business logic.
type AuthService interface {
	// Register registers a new user and sends OTP.
	Register(ctx context.Context, contact string) (*model.AuthResponse, error)

	// RequestOTP generates and sends OTP to the contact.
	RequestOTP(ctx context.Context, contact string) (*model.AuthResponse, error)

	// VerifyOTP verifies OTP and returns access token on success.
	VerifyOTP(ctx context.Context, contact, otpCode string) (*model.LoginResponse, error)

	// ValidateToken validates an access token.
	ValidateToken(ctx context.Context, token string) (*model.TokenResponse, error)
}

// authServiceImpl is the concrete implementation of AuthService.
type authServiceImpl struct {
	userService    UserService
	otpService     OTPService
	tokenGenerator TokenGenerator
}

// NewAuthService creates a new AuthService with the given dependencies.
func NewAuthService(userService UserService, otpService OTPService, tokenGenerator TokenGenerator) AuthService {
	return &authServiceImpl{
		userService:    userService,
		otpService:     otpService,
		tokenGenerator: tokenGenerator,
	}
}

// --- Validation helpers ---

func normalizeInput(s string) string {
	return strings.TrimSpace(s)
}

func isValidEmail(contact string) bool {
	at := strings.LastIndex(contact, "@")
	if at <= 0 || at == len(contact)-1 {
		return false
	}
	local, domain := contact[:at], contact[at+1:]
	if local == "" || domain == "" {
		return false
	}
	return strings.Contains(domain, ".")
}

// isValidPhone accepts optional leading '+', ASCII digits, and separators space/hyphen only.
func isValidPhone(contact string) bool {
	s := strings.TrimSpace(contact)
	if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	digits := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ' || c == '-':
			continue
		case c >= '0' && c <= '9':
			digits++
		default:
			return false
		}
	}
	return digits >= 5 && digits <= 15
}

// canonicalPhone collapses a validated phone to +digits or digits-only (no separators).
func canonicalPhone(contact string) string {
	s := strings.TrimSpace(contact)
	hasPlus := strings.HasPrefix(s, "+")
	if hasPlus {
		s = s[1:]
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '-' {
			continue
		}
		if c >= '0' && c <= '9' {
			b.WriteByte(c)
		}
	}
	out := b.String()
	if hasPlus {
		return "+" + out
	}
	return out
}

func canonicalContact(contact string) string {
	if isValidEmail(contact) {
		return contact
	}
	return canonicalPhone(contact)
}

func validateContact(contact string) error {
	if contact == "" {
		return ErrContactRequired
	}
	if !isValidEmail(contact) && !isValidPhone(contact) {
		return ErrContactInvalidFormat
	}
	return nil
}

// --- Service Methods ---

// Register registers a new user and sends OTP.
func (s *authServiceImpl) Register(ctx context.Context, contact string) (*model.AuthResponse, error) {
	contact = normalizeInput(contact)
	if err := validateContact(contact); err != nil {
		return nil, err
	}
	contact = canonicalContact(contact)

	if err := s.userService.CreateUser(ctx, contact); err != nil {
		return nil, err
	}

	if err := s.otpService.GenerateOTP(ctx, contact); err != nil {
		return nil, err
	}

	return &model.AuthResponse{Status: "success", Message: "register success"}, nil
}

// RequestOTP generates and sends OTP to the contact.
func (s *authServiceImpl) RequestOTP(ctx context.Context, contact string) (*model.AuthResponse, error) {
	contact = normalizeInput(contact)
	if err := validateContact(contact); err != nil {
		return nil, err
	}
	contact = canonicalContact(contact)

	if err := s.otpService.GenerateOTP(ctx, contact); err != nil {
		return nil, err
	}

	return &model.AuthResponse{Status: "success", Message: "OTP sent"}, nil
}

// VerifyOTP verifies OTP and returns access token on success.
func (s *authServiceImpl) VerifyOTP(ctx context.Context, contact, otpCode string) (*model.LoginResponse, error) {
	contact = normalizeInput(contact)
	otpCode = normalizeInput(otpCode)

	if contact == "" || otpCode == "" {
		return nil, ErrInputRequired
	}
	if err := validateContact(contact); err != nil {
		return nil, err
	}
	contact = canonicalContact(contact)

	valid, err := s.otpService.VerifyOTP(ctx, contact, otpCode)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrOTPInvalid
	}

	user, err := s.userService.GetUser(ctx, contact)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if strings.TrimSpace(user.ID) == "" {
		return nil, ErrUserIDMissing
	}

	token, err := s.tokenGenerator.GenerateToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		Status:      "success",
		Message:     "login successful",
		AccessToken: token,
	}, nil
}

// ValidateToken validates an access token.
func (s *authServiceImpl) ValidateToken(ctx context.Context, token string) (*model.TokenResponse, error) {
	token = normalizeInput(token)
	if token == "" {
		return &model.TokenResponse{Status: "invalid", Message: "token invalid"}, nil
	}

	valid, err := s.tokenGenerator.ValidateToken(token)
	if err != nil {
		return nil, err
	}
	if !valid {
		return &model.TokenResponse{Status: "invalid", Message: "token invalid"}, nil
	}

	return &model.TokenResponse{Status: "valid", Message: "token valid"}, nil
}
