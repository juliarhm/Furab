// Package handler provides HTTP handlers for emergency-service.
package handler

import (
	"net/http"

	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// EmergencyHandler handles HTTP requests for emergency-service.
type EmergencyHandler struct {
	// TODO: add service dependency
}

// NewEmergencyHandler creates a new EmergencyHandler.
func NewEmergencyHandler() *EmergencyHandler {
	return &EmergencyHandler{}
}

// RegisterRoutes registers all emergency-service routes.
func (h *EmergencyHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/emergencys", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "emergency-service",
			})
		})
		// TODO: Register endpoint routes
	})
}
