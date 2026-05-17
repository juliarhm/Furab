// Package handler provides HTTP handlers for OTP service API endpoints.
package handler

import (
	"encoding/json"
	"net/http"

	"furab-backend/services/otp-service/internal/model"
	"furab-backend/services/otp-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// OTPHandler handles HTTP requests for OTP operations.
type OTPHandler struct {
	service service.OTPService
}

// NewOTPHandler creates a new OTPHandler with the given service.
func NewOTPHandler(svc service.OTPService) *OTPHandler {
	return &OTPHandler{service: svc}
}

// RegisterRoutes registers all OTP routes on the given chi router.
func (h *OTPHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/otps", func(r chi.Router) {
		r.Post("/generate", h.GenerateOTP) // POST /api/v1/otps/generate
		r.Post("/verify", h.VerifyOTP)     // POST /api/v1/otps/verify
	})
}

// GenerateOTP handles POST /api/v1/otps/generate
// Generates a new OTP for the given target (phone/email).
func (h *OTPHandler) GenerateOTP(w http.ResponseWriter, r *http.Request) {
	var req model.GenerateOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.GenerateOTP(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusCreated, resp.Message, resp)
}

// VerifyOTP handles POST /api/v1/otps/verify
// Verifies an OTP code for the given target.
func (h *OTPHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req model.VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.VerifyOTP(r.Context(), &req)
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
		"service": "otp-service",
	})
}

// handleServiceError maps service errors to HTTP responses.
func handleServiceError(w http.ResponseWriter, err error) {
	switch err {
	case service.ErrOTPNotFound:
		utils.ErrorResponse(w, http.StatusNotFound, "otp not found")
	case service.ErrOTPExpired:
		utils.ErrorResponse(w, http.StatusGone, "otp expired")
	case service.ErrOTPInvalid:
		utils.ErrorResponse(w, http.StatusUnauthorized, "invalid otp code")
	case service.ErrInvalidRequest:
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
	default:
		// Validation errors from model
		if err.Error() != "" {
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "internal server error")
	}
}
