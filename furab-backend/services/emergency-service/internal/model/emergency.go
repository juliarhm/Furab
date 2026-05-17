// Package model defines the domain models for emergency-service.
package model

import "time"

// TriggerEmergencyRequest represents incoming emergency trigger input.
type TriggerEmergencyRequest struct {
	ActorID       string    `json:"actor_id"`
	ActorType     string    `json:"actor_type"` // user / driver
	OrderID       string    `json:"order_id"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	EmergencyType string    `json:"emergency_type"` // accident / unsafe / other
	Timestamp     time.Time `json:"timestamp"`
}

// TriggerEmergencyResponse represents service output after trigger.
type TriggerEmergencyResponse struct {
	Status      string `json:"status"` // success / failed
	Message     string `json:"message"`
	EmergencyID string `json:"emergency_id"`
}

// EmergencyLocation represents actor location fetched from location service.
type EmergencyLocation struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
	Accuracy  float64   `json:"accuracy"`
}

// EmergencyContact represents destination contact for emergency alert.
type EmergencyContact struct {
	ReceiverID string `json:"receiver_id"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
}

// EmergencyEvent represents persisted emergency data.
type EmergencyEvent struct {
	EmergencyID   string     `json:"emergency_id"`
	ActorID       string     `json:"actor_id"`
	ActorType     string     `json:"actor_type"`
	OrderID       string     `json:"order_id"`
	Latitude      float64    `json:"latitude"`
	Longitude     float64    `json:"longitude"`
	EmergencyType string     `json:"emergency_type"`
	Status        string     `json:"status"` // active / resolved
	CreatedAt     time.Time  `json:"created_at"`
	ResolvedAt    *time.Time `json:"resolved_at"`
}

// EmergencyNotification represents outgoing notification event payload.
type EmergencyNotification struct {
	ReceiverID  string    `json:"receiver_id"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Priority    string    `json:"priority"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	LocationURL string    `json:"location_url"`
	Timestamp   time.Time `json:"timestamp"`
}

