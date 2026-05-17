// Package handler provides HTTP handlers for cart-service.
package handler

import (
	"net/http"

	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// CartHandler handles HTTP requests for cart-service.
type CartHandler struct {
	// TODO: add service dependency
}

// NewCartHandler creates a new CartHandler.
func NewCartHandler() *CartHandler {
	return &CartHandler{}
}

// RegisterRoutes registers all cart-service routes.
func (h *CartHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/carts", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "cart-service",
			})
		})
		// TODO: Register endpoint routes
	})
}
