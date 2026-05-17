package model

import "time"

// Review represents the review entity in the system.
type Review struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	MerchantID  string    `json:"merchant_id"`
	OrderID     string    `json:"order_id"`
	Rating      int       `json:"rating"`
	Comment     string    `json:"comment"`
	ImageURL    string    `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SearchReviewRequest represents the criteria for searching reviews.
type SearchReviewRequest struct {
	MerchantID string `json:"merchant_id"`
	UserID     string `json:"user_id"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}
