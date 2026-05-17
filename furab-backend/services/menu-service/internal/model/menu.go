package model

import "time"

// Menu represents the menu entity in the system.
type Menu struct {
	ID          string    `json:"id"`
	MerchantID  string    `json:"merchant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	ImageURL    string    `json:"image_url"`
	IsAvailable bool      `json:"is_available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SearchMenuRequest represents the criteria for searching menus.
type SearchMenuRequest struct {
	MerchantID string `json:"merchant_id"`
	Query      string `json:"query"`
	Category   string `json:"category"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}
