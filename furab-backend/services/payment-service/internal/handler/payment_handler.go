// Package handler provides HTTP handlers for payment-service.
package handler

import (
	"encoding/json"
	"net/http"

	"furab-backend/services/payment-service/internal/model"
	"furab-backend/services/payment-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// PaymentHandler handles HTTP requests for payment-service.
type PaymentHandler struct {
	service service.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler with the given service.
func NewPaymentHandler(svc service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: svc}
}

// RegisterRoutes registers all payment-service routes.
func (h *PaymentHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/payments", func(r chi.Router) {
		r.Get("/health", h.HealthCheck)
		r.Post("/", h.InitiatePayment)
		r.Get("/{paymentID}", h.GetPayment)
		r.Put("/{paymentID}/capture", h.CapturePayment)
		r.Put("/{paymentID}/cancel", h.CancelPayment)
		r.Put("/{paymentID}/refund", h.RefundPayment)
	})
}

// HealthCheck handles GET /api/v1/payments/health.
func (h *PaymentHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	utils.SuccessResponse(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "payment-service",
	})
}

// InitiatePayment handles POST /api/v1/payments/.
func (h *PaymentHandler) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	var req model.InitiatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	payment, err := h.service.InitiatePayment(r.Context(), &req)
	if err != nil {
		switch err {
		case service.ErrInvalidRequest:
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(w, http.StatusInternalServerError, "failed to initiate payment")
		}
		return
	}

	utils.SuccessMessageResponse(w, http.StatusCreated, "payment initiated successfully", payment)
}

// GetPayment handles GET /api/v1/payments/{paymentID}.
func (h *PaymentHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "paymentID")
	if paymentID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "payment ID is required")
		return
	}

	payment, err := h.service.GetPayment(r.Context(), paymentID)
	if err != nil {
		switch err {
		case service.ErrPaymentNotFound:
			utils.ErrorResponse(w, http.StatusNotFound, err.Error())
		case service.ErrInvalidRequest:
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(w, http.StatusInternalServerError, "failed to get payment")
		}
		return
	}

	utils.SuccessResponse(w, http.StatusOK, payment)
}

// CapturePayment handles PUT /api/v1/payments/{paymentID}/capture.
func (h *PaymentHandler) CapturePayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "paymentID")
	if paymentID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "payment ID is required")
		return
	}

	payment, err := h.service.CapturePayment(r.Context(), paymentID)
	if err != nil {
		switch err {
		case service.ErrPaymentNotFound:
			utils.ErrorResponse(w, http.StatusNotFound, err.Error())
		case service.ErrInvalidRequest:
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, "payment captured successfully", payment)
}

// CancelPayment handles PUT /api/v1/payments/{paymentID}/cancel.
func (h *PaymentHandler) CancelPayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "paymentID")
	if paymentID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "payment ID is required")
		return
	}

	payment, err := h.service.CancelPayment(r.Context(), paymentID)
	if err != nil {
		switch err {
		case service.ErrPaymentNotFound:
			utils.ErrorResponse(w, http.StatusNotFound, err.Error())
		case service.ErrInvalidRequest:
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, "payment cancelled successfully", payment)
}

// RefundPayment handles PUT /api/v1/payments/{paymentID}/refund.
func (h *PaymentHandler) RefundPayment(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "paymentID")
	if paymentID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "payment ID is required")
		return
	}

	payment, err := h.service.RefundPayment(r.Context(), paymentID)
	if err != nil {
		switch err {
		case service.ErrPaymentNotFound:
			utils.ErrorResponse(w, http.StatusNotFound, err.Error())
		case service.ErrInvalidRequest:
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		default:
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, "payment refunded successfully", payment)
}
