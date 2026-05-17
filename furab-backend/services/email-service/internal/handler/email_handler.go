// Package handler provides HTTP handlers for email-service.
package handler

import (
	"net/http"

	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// EmailHandler handles HTTP requests for email-service.
type EmailHandler struct {
	// TODO: add service dependency
}

// NewEmailHandler creates a new EmailHandler.
func NewEmailHandler() *EmailHandler {
	return &EmailHandler{}
}

// RegisterRoutes registers all email-service routes.
func (h *EmailHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/emails", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "email-service",
			})
		})
		// TODO: Register endpoint routes
	})
}
