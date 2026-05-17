// Package handler provides HTTP handlers for matching-service.
package handler

import (
	"net/http"

	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// MatchHandler handles HTTP requests for matching-service.
type MatchHandler struct {
	// TODO: add service dependency
}

// NewMatchHandler creates a new MatchHandler.
func NewMatchHandler() *MatchHandler {
	return &MatchHandler{}
}

// RegisterRoutes registers all matching-service routes.
func (h *MatchHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/matchs", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "matching-service",
			})
		})
		// TODO: Register endpoint routes
	})
}
