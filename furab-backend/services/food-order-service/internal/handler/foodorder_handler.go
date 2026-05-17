// Package handler provides HTTP handlers for food-order-service.
package handler

import (
	"net/http"

	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// FoodOrderHandler handles HTTP requests for food-order-service.
type FoodOrderHandler struct {
	// TODO: add service dependency
}

// NewFoodOrderHandler creates a new FoodOrderHandler.
func NewFoodOrderHandler() *FoodOrderHandler {
	return &FoodOrderHandler{}
}

// RegisterRoutes registers all food-order-service routes.
func (h *FoodOrderHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/foodorders", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "food-order-service",
			})
		})
		// TODO: Register endpoint routes
	})
}
