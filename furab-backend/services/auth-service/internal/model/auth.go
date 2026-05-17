// Package model defines the domain models and DTOs for the auth service.
package model

import "time"

// User represents a user in the auth context.
type User struct {
	ID        string    `json:"id"`
	Contact   string    `json:"contact"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Session represents an auth session.
type Session struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// --- Request DTOs ---

// RegisterRequest is the request body for registration.
type RegisterRequest struct {
	Contact string `json:"contact"` // phone or email
}

// OTPRequest is the request body for requesting/verifying OTP.
type OTPRequest struct {
	Contact string `json:"contact"`
	OTPCode string `json:"otp_code,omitempty"`
}

// TokenRequest is the request body for token validation.
type TokenRequest struct {
	Token string `json:"token"`
}

// --- Response DTOs ---

// AuthResponse represents standard auth response.
type AuthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// LoginResponse represents login response with token.
type LoginResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	AccessToken string `json:"access_token"`
}

// TokenResponse represents token validation response.
type TokenResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
