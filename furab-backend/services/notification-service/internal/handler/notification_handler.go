// Package handler provides HTTP handlers for notification-service.
package handler

import (
	"net/http"

	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// NotificationHandler handles HTTP requests for notification-service.
type NotificationHandler struct {
	// TODO: add service dependency
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

// RegisterRoutes registers all notification-service routes.
func (h *NotificationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/notifications", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "notification-service",
			})
		})
		// TODO: Register endpoint routes
	})
}
