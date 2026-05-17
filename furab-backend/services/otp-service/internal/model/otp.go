// Package model defines the domain models and DTOs for the OTP service.
package model

import (
	"errors"
	"time"
)

// OTP represents the core OTP entity in the system.
type OTP struct {
	OTPID     string    `json:"otp_id"`
	Target    string    `json:"target"`    // phone number or email
	OTPCode   string    `json:"otp_code"`
	ExpiredAt time.Time `json:"expired_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired checks if the OTP has expired.
func (o *OTP) IsExpired() bool {
	return time.Now().After(o.ExpiredAt)
}

// --- Request DTOs ---

// GenerateOTPRequest is the request body for generating a new OTP.
type GenerateOTPRequest struct {
	Target string `json:"target" validate:"required"` // phone or email
}

// Validate validates the generate OTP request.
func (r *GenerateOTPRequest) Validate() error {
	if r.Target == "" {
		return errors.New("target is required")
	}
	return nil
}

// VerifyOTPRequest is the request body for verifying an OTP.
type VerifyOTPRequest struct {
	Target  string `json:"target" validate:"required"`
	OTPCode string `json:"otp_code" validate:"required"`
}

// Validate validates the verify OTP request.
func (r *VerifyOTPRequest) Validate() error {
	if r.Target == "" {
		return errors.New("target is required")
	}
	if r.OTPCode == "" {
		return errors.New("otp_code is required")
	}
	return nil
}

// --- Response DTOs ---

// GenerateOTPResponse is the response body for generating an OTP.
type GenerateOTPResponse struct {
	Status  string `json:"status"`  // "success" or "failed"
	Message string `json:"message"`
}

// VerifyOTPResponse is the response body for verifying an OTP.
type VerifyOTPResponse struct {
	Status  string `json:"status"` // "valid" or "invalid"
	Message string `json:"message"`
}
