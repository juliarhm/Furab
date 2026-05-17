// Package handler provides HTTP handlers for settlement-service.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"furab-backend/services/settlement-service/internal/model"
	"furab-backend/services/settlement-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// SettlementHandler handles HTTP requests for settlement-service.
type SettlementHandler struct {
	service service.SettlementService
}

// NewSettlementHandler creates a new SettlementHandler.
func NewSettlementHandler(svc service.SettlementService) *SettlementHandler {
	return &SettlementHandler{service: svc}
}

// RegisterRoutes registers all settlement-service routes.
func (h *SettlementHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/settlements", func(r chi.Router) {
		r.Get("/health", HealthCheck)
		r.Post("/process", h.ProcessSettlement)
	})
}

// ProcessSettlement handles payment CAPTURED trigger to distribute split payment.
func (h *SettlementHandler) ProcessSettlement(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "settlement service is not initialized")
		return
	}

	var req model.ProcessSettlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	res, err := h.service.ProcessSettlement(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, "settlement processed", res)
}

// HealthCheck handles GET /health.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	utils.SuccessResponse(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "settlement-service",
	})
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidRequest):
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrRecipientNotActive):
		utils.ErrorResponse(w, http.StatusConflict, err.Error())
	default:
		utils.ErrorResponse(w, http.StatusInternalServerError, "internal server error")
	}
}
