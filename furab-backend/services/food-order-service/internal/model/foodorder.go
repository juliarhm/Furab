// Package model defines the domain models for food-order-service.
package model

import (
	"errors"
	"time"
)

// FoodOrderStatus represents the current state of a food order.
type FoodOrderStatus string

const (
	FoodStatusPending    FoodOrderStatus = "PENDING"
	FoodStatusConfirmed  FoodOrderStatus = "CONFIRMED"
	FoodStatusPreparing  FoodOrderStatus = "PREPARING"
	FoodStatusReady      FoodOrderStatus = "READY"
	FoodStatusPickedUp   FoodOrderStatus = "PICKED_UP"
	FoodStatusDelivering FoodOrderStatus = "DELIVERING"
	FoodStatusCompleted  FoodOrderStatus = "COMPLETED"
	FoodStatusCancelled  FoodOrderStatus = "CANCELLED"
)

// PaymentStatus represents the payment state.
type PaymentStatus string

const (
	PaymentStatusNone       PaymentStatus = "NONE"
	PaymentStatusAuthorized PaymentStatus = "AUTHORIZED"
	PaymentStatusCaptured   PaymentStatus = "CAPTURED"
	PaymentStatusRefunded   PaymentStatus = "REFUNDED"
)

// ValidFoodTransitions defines allowed state transitions.
// Flow: PENDING → CONFIRMED → PREPARING → READY → PICKED_UP → DELIVERING → COMPLETED
var ValidFoodTransitions = map[FoodOrderStatus][]FoodOrderStatus{
	FoodStatusPending:    {FoodStatusConfirmed, FoodStatusCancelled},
	FoodStatusConfirmed:  {FoodStatusPreparing, FoodStatusCancelled},
	FoodStatusPreparing:  {FoodStatusReady, FoodStatusCancelled},
	FoodStatusReady:      {FoodStatusPickedUp},
	FoodStatusPickedUp:   {FoodStatusDelivering},
	FoodStatusDelivering: {FoodStatusCompleted},
	FoodStatusCompleted:  {},
	FoodStatusCancelled:  {},
}

// CanTransitionTo checks if the current status can transition to the target.
func (s FoodOrderStatus) CanTransitionTo(target FoodOrderStatus) bool {
	allowed, exists := ValidFoodTransitions[s]
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

// OrderItem represents a single item in the food order.
type OrderItem struct {
	ID         string  `json:"id"`
	MenuItemID string  `json:"menu_item_id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Quantity   int     `json:"quantity"`
	Notes      string  `json:"notes,omitempty"`
}

// Location represents a geographical coordinate.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

// FoodOrder represents a food order in the system.
type FoodOrder struct {
	ID              string          `json:"id"`
	UserID          string          `json:"user_id"`
	MerchantID      string          `json:"merchant_id"`
	DriverID        string          `json:"driver_id,omitempty"`
	Items           []OrderItem     `json:"items"`
	Status          FoodOrderStatus `json:"status"`
	PaymentStatus   PaymentStatus   `json:"payment_status"`
	SubTotal        float64         `json:"sub_total"`
	DeliveryFee     float64         `json:"delivery_fee"`
	Discount        float64         `json:"discount"`
	TotalAmount     float64         `json:"total_amount"`
	DeliveryAddress Location        `json:"delivery_address"`
	MerchantAddress Location        `json:"merchant_address"`
	CancelledBy     string          `json:"cancelled_by,omitempty"`
	CancelReason    string          `json:"cancel_reason,omitempty"`
	Notes           string          `json:"notes,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// CalculateTotal recalculates the order totals.
func (o *FoodOrder) CalculateTotal() {
	sub := 0.0
	for _, item := range o.Items {
		sub += item.Price * float64(item.Quantity)
	}
	o.SubTotal = sub
	o.TotalAmount = sub + o.DeliveryFee - o.Discount
}

// --- Request DTOs ---

// CreateFoodOrderRequest is the request body to create a food order.
type CreateFoodOrderRequest struct {
	UserID          string      `json:"user_id"`
	MerchantID      string      `json:"merchant_id"`
	Items           []OrderItem `json:"items"`
	DeliveryAddress Location    `json:"delivery_address"`
	MerchantAddress Location    `json:"merchant_address"`
	Notes           string      `json:"notes,omitempty"`
}

// Validate validates the create food order request.
func (r *CreateFoodOrderRequest) Validate() error {
	if r.UserID == "" {
		return errors.New("user_id is required")
	}
	if r.MerchantID == "" {
		return errors.New("merchant_id is required")
	}
	if len(r.Items) == 0 {
		return errors.New("at least one item is required")
	}
	if r.DeliveryAddress.Address == "" {
		return errors.New("delivery address is required")
	}
	return nil
}

// CancelFoodOrderRequest is the request body for cancelling.
type CancelFoodOrderRequest struct {
	CancelledBy  string `json:"cancelled_by"`
	CancelReason string `json:"cancel_reason,omitempty"`
}

// --- Event Payloads ---

type FoodCreatedEvent struct {
	OrderID    string `json:"order_id"`
	UserID     string `json:"user_id"`
	MerchantID string `json:"merchant_id"`
	TotalAmount float64 `json:"total_amount"`
}

type FoodConfirmedEvent struct {
	OrderID    string `json:"order_id"`
	MerchantID string `json:"merchant_id"`
}

type FoodReadyEvent struct {
	OrderID         string   `json:"order_id"`
	MerchantID      string   `json:"merchant_id"`
	MerchantAddress Location `json:"merchant_address"`
	DeliveryAddress Location `json:"delivery_address"`
}

type FoodCompletedEvent struct {
	OrderID     string  `json:"order_id"`
	UserID      string  `json:"user_id"`
	DriverID    string  `json:"driver_id"`
	MerchantID  string  `json:"merchant_id"`
	TotalAmount float64 `json:"total_amount"`
}

type FoodCancelledEvent struct {
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	MerchantID  string `json:"merchant_id"`
	CancelledBy string `json:"cancelled_by"`
	Reason      string `json:"reason,omitempty"`
}
