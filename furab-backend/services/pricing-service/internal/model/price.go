// Package model defines the domain models for pricing-service.
package model

// PriceCalculationRequest is the incoming API request payload.
type PriceCalculationRequest struct {
	OrderID string `json:"order_id"`
}

// PriceCalculationResponse is the API response for price estimation.
type PriceCalculationResponse struct {
	OrderID     string  `json:"order_id"`
	TotalAmount float64 `json:"total_amount"`
	ItemPrice   float64 `json:"item_price"`
	DeliveryFee float64 `json:"delivery_fee"`
	ServiceFee  float64 `json:"service_fee"`
}

// PriceRule represents a pricing rule stored in pricing_rules.
type PriceRule struct {
	RuleID      string  `json:"rule_id"`
	Type        string  `json:"type"`
	Value       float64 `json:"value"`
	Description string  `json:"description"`
}

// OrderItem is a simplified item model for order pricing.
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

