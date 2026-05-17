// Package model defines the domain models for location-service.
package model

import "time"

// UpdateLocationRequest represents the input to update a driver's location.
type UpdateLocationRequest struct {
	DriverID  string    `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

// UpdateStatusRequest represents the input to update a driver's status.
type UpdateStatusRequest struct {
	DriverID     string `json:"driver_id"`
	DriverStatus string `json:"driver_status"` // "available" or "busy"
}

// SearchDriverRequest represents the input to search for nearby drivers.
type SearchDriverRequest struct {
	LatitudeOrigin  float64 `json:"latitude_origin"`
	LongitudeOrigin float64 `json:"longitude_origin"`
	Radius          float64 `json:"radius"` // in km
}

// DriverLocationResponse represents a single driver in the search results.
type DriverLocationResponse struct {
	DriverID     string  `json:"driver_id"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Distance     float64 `json:"distance"` // distance from origin in km
	DriverStatus string  `json:"driver_status"`
}

// TrackLocationResponse represents the output for tracking a specific driver.
type TrackLocationResponse struct {
	DriverID  string    `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}
