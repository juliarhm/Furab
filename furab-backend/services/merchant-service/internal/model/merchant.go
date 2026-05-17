package model

import "time"

// Merchant represents the merchant entity in the system.
type Merchant struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Address     string    `json:"address"`
	Description string    `json:"description"`
	Rating      float64   `json:"rating"`
	IsOpen      bool      `json:"is_open"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SearchMerchantRequest represents the criteria for searching merchants.
type SearchMerchantRequest struct {
	Query  string `json:"query"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}
