// Package handler provides HTTP handlers for pricing-service.
package handler

import (
	"errors"
	"net/http"

	"furab-backend/services/pricing-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// PriceHandler handles HTTP requests for pricing-service.
type PriceHandler struct {
	service service.PriceService
}

// NewPriceHandler creates a new PriceHandler.
func NewPriceHandler(svc service.PriceService) *PriceHandler {
	return &PriceHandler{service: svc}
}

// RegisterRoutes registers all pricing-service routes.
func (h *PriceHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/prices", func(r chi.Router) {
		r.Get("/health", HealthCheck)
		r.Get("/{orderID}", h.GetPriceEstimate)
	})
}

// GetPriceEstimate handles GET /api/v1/prices/{orderID}
func (h *PriceHandler) GetPriceEstimate(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	if orderID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "order ID is required")
		return
	}

	response, err := h.service.CalculatePrice(r.Context(), orderID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRequest):
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(w, http.StatusInternalServerError, "failed to calculate price")
		}
		return
	}

	utils.SuccessResponse(w, http.StatusOK, response)
}

// HealthCheck handles GET /health.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	utils.SuccessResponse(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "pricing-service",
	})
}
