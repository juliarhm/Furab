// Package handler provides HTTP handlers for promo-service.
package handler

import (
	"net/http"

	"furab-backend/services/promo-service/internal/model"
	"furab-backend/services/promo-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// PromoHandler handles HTTP requests for promo-service.
type PromoHandler struct {
	service service.PromoService
}

// NewPromoHandler creates a new PromoHandler.
func NewPromoHandler(svc service.PromoService) *PromoHandler {
	return &PromoHandler{service: svc}
}

// RegisterRoutes registers all promo-service routes.
func (h *PromoHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/promos", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "promo-service",
			})
		})
		r.Post("/validate", h.ValidatePromo)
	})
}

// ValidatePromo handles POST /api/v1/promos/validate
func (h *PromoHandler) ValidatePromo(w http.ResponseWriter, r *http.Request) {
	var req model.PromoValidationRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.ValidatePromo(r.Context(), req.PromoCode, req.UserID, req.OrderID, req.TotalAmount)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to validate promo: "+err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusOK, result)
}
