// Package handler provides HTTP handlers for chat-service.
package handler

import (
	"net/http"

	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// ChatHandler handles HTTP requests for chat-service.
type ChatHandler struct {
	// TODO: add service dependency
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}

// RegisterRoutes registers all chat-service routes.
func (h *ChatHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/chats", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "chat-service",
			})
		})
		// TODO: Register endpoint routes
	})
}
