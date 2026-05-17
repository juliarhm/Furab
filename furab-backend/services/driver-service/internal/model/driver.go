// Package model defines the domain models and DTOs for the driver service.
package model

import (
	"errors"
	"time"
)

// DriverStatus represents the availability status of a driver.
type DriverStatus string

const (
	DriverStatusOnline  DriverStatus = "ONLINE"
	DriverStatusOffline DriverStatus = "OFFLINE"
	DriverStatusBusy    DriverStatus = "BUSY"
)

// IsValid checks if the driver status is a known valid status.
func (s DriverStatus) IsValid() bool {
	switch s {
	case DriverStatusOnline, DriverStatusOffline, DriverStatusBusy:
		return true
	}
	return false
}

// Driver represents a driver entity.
type Driver struct {
	DriverID    string       `json:"driver_id"`
	Name        string       `json:"name"`
	Phone       string       `json:"phone"`
	VehicleType string       `json:"vehicle_type"`
	Status      DriverStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// DriverLocation represents a driver's GPS coordinates.
type DriverLocation struct {
	DriverID  string    `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Request DTOs ---

// CreateDriverRequest holds the input for creating a driver.
type CreateDriverRequest struct {
	DriverID    string `json:"driver_id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Phone       string `json:"phone" validate:"required"`
	VehicleType string `json:"vehicle_type" validate:"required"`
}

// Validate validates the create driver request.
func (r *CreateDriverRequest) Validate() error {
	if r.DriverID == "" {
		return errors.New("driver_id is required")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Phone == "" {
		return errors.New("phone is required")
	}
	if r.VehicleType == "" {
		return errors.New("vehicle_type is required")
	}
	return nil
}

// UpdateDriverRequest holds the input for updating a driver.
type UpdateDriverRequest struct {
	Name        string `json:"name" validate:"required"`
	Phone       string `json:"phone" validate:"required"`
	VehicleType string `json:"vehicle_type" validate:"required"`
}

// Validate validates the update driver request.
func (r *UpdateDriverRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Phone == "" {
		return errors.New("phone is required")
	}
	if r.VehicleType == "" {
		return errors.New("vehicle_type is required")
	}
	return nil
}

// --- Response DTOs ---

// DriverResponse is a generic response for driver operations.
type DriverResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	DriverID string `json:"driver_id,omitempty"`
}
