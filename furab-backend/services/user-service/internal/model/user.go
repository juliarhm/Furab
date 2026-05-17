// Package model defines the domain models and DTOs for the user service.
package model

import (
	"errors"
	"time"
)

// UserStatus represents the current state of a user account.
type UserStatus string

const (
	// UserStatusActive indicates the user account is active and operational.
	UserStatusActive UserStatus = "active"
	// UserStatusInactive indicates the user account has been deactivated.
	UserStatusInactive UserStatus = "inactive"
)

// IsValid checks if the user status is a known valid status.
func (s UserStatus) IsValid() bool {
	switch s {
	case UserStatusActive, UserStatusInactive:
		return true
	}
	return false
}

// User represents the core user entity in the system.
type User struct {
	UserID    string     `json:"user_id"`
	Name      string     `json:"name"`
	Phone     string     `json:"phone"`
	Email     string     `json:"email"`
	Status    UserStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// --- Request DTOs ---

// CreateUserRequest is the request body for creating a new user.
type CreateUserRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Name   string `json:"name" validate:"required"`
	Phone  string `json:"phone" validate:"required"`
	Email  string `json:"email" validate:"required"`
}

// Validate validates the create user request.
func (r *CreateUserRequest) Validate() error {
	if r.UserID == "" {
		return errors.New("user_id is required")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Phone == "" {
		return errors.New("phone is required")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Phone string `json:"phone" validate:"required"`
	Email string `json:"email" validate:"required"`
}

// Validate validates the update user request.
func (r *UpdateUserRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Phone == "" {
		return errors.New("phone is required")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

// --- Response DTOs ---

// CreateUserResponse is the response body for creating a new user.
type CreateUserResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// MessageResponse is a generic response with a message.
type MessageResponse struct {
	Message string `json:"message"`
}
