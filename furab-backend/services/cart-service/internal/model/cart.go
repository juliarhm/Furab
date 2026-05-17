// Package model defines the domain models for cart-service.
package model

import (
	"errors"
	"time"
)

// CartItem represents a single item in the shopping cart.
type CartItem struct {
	ID         string  `json:"id"`
	MenuItemID string  `json:"menu_item_id"`
	MerchantID string  `json:"merchant_id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Quantity   int     `json:"quantity"`
	Notes      string  `json:"notes,omitempty"`
}

// Cart represents a user's shopping cart.
type Cart struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	MerchantID string     `json:"merchant_id"`
	Items      []CartItem `json:"items"`
	TotalPrice float64    `json:"total_price"`
	ItemCount  int        `json:"item_count"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// CalculateTotal recalculates total price and item count.
func (c *Cart) CalculateTotal() {
	total := 0.0
	count := 0
	for _, item := range c.Items {
		total += item.Price * float64(item.Quantity)
		count += item.Quantity
	}
	c.TotalPrice = total
	c.ItemCount = count
}

// FindItem finds an item in the cart by its ID.
func (c *Cart) FindItem(itemID string) (*CartItem, int) {
	for i, item := range c.Items {
		if item.ID == itemID {
			return &c.Items[i], i
		}
	}
	return nil, -1
}

// FindItemByMenuID finds an item by menu item ID.
func (c *Cart) FindItemByMenuID(menuItemID string) (*CartItem, int) {
	for i, item := range c.Items {
		if item.MenuItemID == menuItemID {
			return &c.Items[i], i
		}
	}
	return nil, -1
}

// --- Request DTOs ---

// AddItemRequest is the request body to add an item to the cart.
type AddItemRequest struct {
	MenuItemID string  `json:"menu_item_id" validate:"required"`
	MerchantID string  `json:"merchant_id" validate:"required"`
	Name       string  `json:"name" validate:"required"`
	Price      float64 `json:"price" validate:"required"`
	Quantity   int     `json:"quantity" validate:"required"`
	Notes      string  `json:"notes,omitempty"`
}

// Validate validates the add item request.
func (r *AddItemRequest) Validate() error {
	if r.MenuItemID == "" {
		return errors.New("menu_item_id is required")
	}
	if r.MerchantID == "" {
		return errors.New("merchant_id is required")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Price <= 0 {
		return errors.New("price must be greater than 0")
	}
	if r.Quantity <= 0 {
		return errors.New("quantity must be greater than 0")
	}
	return nil
}

// UpdateQuantityRequest is the request body to update item quantity.
type UpdateQuantityRequest struct {
	Quantity int `json:"quantity" validate:"required"`
}
