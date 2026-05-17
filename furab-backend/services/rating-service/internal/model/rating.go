package model

import "time"

// Rating represents the rating entity in the system.
type Rating struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	TargetID    string    `json:"target_id"`   // Can be merchant_id or driver_id
	TargetType  string    `json:"target_type"` // "merchant" or "driver"
	Score       int       `json:"score"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SearchRatingRequest represents the criteria for searching ratings.
type SearchRatingRequest struct {
	TargetID   string `json:"target_id"`
	TargetType string `json:"target_type"`
	UserID     string `json:"user_id"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}
