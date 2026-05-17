// Package handler provides HTTP handlers for auth-service API endpoints.
package handler

import (
	"encoding/json"
	"net/http"

	"furab-backend/services/auth-service/internal/model"
	"furab-backend/services/auth-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// AuthHandler handles HTTP requests for auth operations.
type AuthHandler struct {
	service service.AuthService
}

// NewAuthHandler creates a new AuthHandler with the given service.
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

// RegisterRoutes registers all auth routes on the given chi router.
func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/otp/request", h.RequestOTP)
		r.Post("/otp/verify", h.VerifyOTP)
		r.Post("/token/validate", h.ValidateToken)
	})
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.Register(r.Context(), req.Contact)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusCreated, resp.Message, resp)
}

// RequestOTP handles POST /api/v1/auth/otp/request
func (h *AuthHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req model.OTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.RequestOTP(r.Context(), req.Contact)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, resp.Message, resp)
}

// VerifyOTP handles POST /api/v1/auth/otp/verify
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req model.OTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.VerifyOTP(r.Context(), req.Contact, req.OTPCode)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, resp.Message, resp)
}

// ValidateToken handles POST /api/v1/auth/token/validate
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	var req model.TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.ValidateToken(r.Context(), req.Token)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessResponse(w, http.StatusOK, resp)
}

// HealthCheck handles GET /health
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	utils.SuccessResponse(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "auth-service",
	})
}

// handleServiceError maps service errors to HTTP responses.
func handleServiceError(w http.ResponseWriter, err error) {
	switch err {
	case service.ErrContactRequired, service.ErrContactInvalidFormat, service.ErrInputRequired:
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
	case service.ErrOTPInvalid:
		utils.ErrorResponse(w, http.StatusUnauthorized, "invalid OTP")
	case service.ErrUserNotFound:
		utils.ErrorResponse(w, http.StatusNotFound, "user not found")
	default:
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
	}
}
