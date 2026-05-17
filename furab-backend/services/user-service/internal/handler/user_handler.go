// Package handler provides HTTP handlers for user-service API endpoints.
package handler

import (
	"encoding/json"
	"net/http"

	"furab-backend/services/user-service/internal/model"
	"furab-backend/services/user-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// UserHandler handles HTTP requests for user operations.
type UserHandler struct {
	service service.UserService
}

// NewUserHandler creates a new UserHandler with the given service.
func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

// RegisterRoutes registers all user routes on the given chi router.
func (h *UserHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/users", func(r chi.Router) {
		r.Post("/", h.CreateUser)                        // POST   /api/v1/users
		r.Get("/{userID}", h.GetUser)                    // GET    /api/v1/users/{userID}
		r.Put("/{userID}", h.UpdateUser)                 // PUT    /api/v1/users/{userID}
		r.Put("/{userID}/deactivate", h.DeactivateUser)  // PUT    /api/v1/users/{userID}/deactivate
	})
}

// CreateUser handles POST /api/v1/users
// Creates a new user account.
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.CreateUser(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusCreated, resp.Message, resp)
}

// GetUser handles GET /api/v1/users/{userID}
// Retrieves a user by their ID.
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "user ID is required")
		return
	}

	user, err := h.service.GetUser(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessResponse(w, http.StatusOK, user)
}

// UpdateUser handles PUT /api/v1/users/{userID}
// Updates an existing user's information.
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "user ID is required")
		return
	}

	var req model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.UpdateUser(r.Context(), userID, &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, resp.Message, resp)
}

// DeactivateUser handles PUT /api/v1/users/{userID}/deactivate
// Deactivates a user account.
func (h *UserHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "user ID is required")
		return
	}

	resp, err := h.service.DeactivateUser(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, resp.Message, resp)
}

// HealthCheck handles GET /health
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	utils.SuccessResponse(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "user-service",
	})
}

// handleServiceError maps service errors to HTTP responses.
func handleServiceError(w http.ResponseWriter, err error) {
	switch err {
	case service.ErrUserNotFound:
		utils.ErrorResponse(w, http.StatusNotFound, "user not found")
	case service.ErrInvalidRequest:
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
	default:
		// Validation errors from model (e.g., "name is required")
		if err.Error() != "" {
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "internal server error")
	}
}
