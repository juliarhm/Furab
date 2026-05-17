// Package model defines the domain models for matching-service.
package model

import (
	"errors"
	"time"
)

// MatchStatus represents the current state of a match request.
type MatchStatus string

const (
	MatchStatusSearching MatchStatus = "SEARCHING"
	MatchStatusOffered   MatchStatus = "OFFERED"
	MatchStatusMatched   MatchStatus = "MATCHED"
	MatchStatusFailed    MatchStatus = "FAILED"
	MatchStatusCancelled MatchStatus = "CANCELLED"
)

// ValidMatchTransitions defines allowed state transitions.
var ValidMatchTransitions = map[MatchStatus][]MatchStatus{
	MatchStatusSearching: {MatchStatusOffered, MatchStatusFailed, MatchStatusCancelled},
	MatchStatusOffered:   {MatchStatusMatched, MatchStatusSearching, MatchStatusCancelled},
	MatchStatusMatched:   {},
	MatchStatusFailed:    {},
	MatchStatusCancelled: {},
}

// CanTransitionTo checks if transition is valid.
func (s MatchStatus) CanTransitionTo(target MatchStatus) bool {
	allowed, exists := ValidMatchTransitions[s]
	if !exists {
		return false
	}
	for _, status := range allowed {
		if status == target {
			return true
		}
	}
	return false
}

// Location represents a geographical coordinate.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

// MatchRequest represents a driver matching request.
type MatchRequest struct {
	ID              string      `json:"id"`
	OrderID         string      `json:"order_id"`
	OrderType       string      `json:"order_type"` // "ride" or "food"
	UserID          string      `json:"user_id"`
	PickupLocation  Location    `json:"pickup_location"`
	DropoffLocation Location    `json:"dropoff_location"`
	Status          MatchStatus `json:"status"`
	DriverID        string      `json:"driver_id,omitempty"`
	AttemptCount    int         `json:"attempt_count"`
	MaxAttempts     int         `json:"max_attempts"`
	Radius          float64     `json:"radius"` // km
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// --- Request DTOs ---

// FindDriverRequest is the request body to initiate driver matching.
type FindDriverRequest struct {
	OrderID         string   `json:"order_id"`
	OrderType       string   `json:"order_type"` // "ride" or "food"
	UserID          string   `json:"user_id"`
	PickupLocation  Location `json:"pickup_location"`
	DropoffLocation Location `json:"dropoff_location"`
}

// Validate validates the find driver request.
func (r *FindDriverRequest) Validate() error {
	if r.OrderID == "" {
		return errors.New("order_id is required")
	}
	if r.OrderType != "ride" && r.OrderType != "food" {
		return errors.New("order_type must be 'ride' or 'food'")
	}
	if r.UserID == "" {
		return errors.New("user_id is required")
	}
	if r.PickupLocation.Address == "" {
		return errors.New("pickup address is required")
	}
	return nil
}

// --- Event Payloads ---

type MatchSearchingEvent struct {
	MatchID   string `json:"match_id"`
	OrderID   string `json:"order_id"`
	OrderType string `json:"order_type"`
}

type MatchOfferedEvent struct {
	MatchID  string `json:"match_id"`
	OrderID  string `json:"order_id"`
	DriverID string `json:"driver_id"`
}

type MatchAcceptedEvent struct {
	MatchID   string `json:"match_id"`
	OrderID   string `json:"order_id"`
	OrderType string `json:"order_type"`
	DriverID  string `json:"driver_id"`
	UserID    string `json:"user_id"`
}

type MatchFailedEvent struct {
	MatchID      string `json:"match_id"`
	OrderID      string `json:"order_id"`
	AttemptCount int    `json:"attempt_count"`
	Reason       string `json:"reason"`
}
