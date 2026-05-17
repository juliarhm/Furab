// Package handler provides HTTP handlers for wallet-service.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"furab-backend/services/wallet-service/internal/model"
	"furab-backend/services/wallet-service/internal/repository"
	"furab-backend/services/wallet-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// WalletHandler handles HTTP requests for wallet-service.
type WalletHandler struct {
	service service.WalletService
}

// NewWalletHandler creates a new WalletHandler.
func NewWalletHandler(svc service.WalletService) *WalletHandler {
	return &WalletHandler{service: svc}
}

// RegisterRoutes registers all wallet-service routes.
func (h *WalletHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/wallets", func(r chi.Router) {
		r.Get("/health", HealthCheck)
		r.Post("/hold", h.HoldBalance)
		r.Post("/release", h.ReleaseBalance)
		r.Post("/debit", h.DebitBalance)
		r.Post("/credit", h.CreditBalance)
		r.Post("/refund", h.Refund)
	})
}

func (h *WalletHandler) HoldBalance(w http.ResponseWriter, r *http.Request) {
	var req model.LockBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	h.handleWalletOperation(w, r, req.UserID, req.Amount, req.Reference, h.service.HoldBalance, "balance hold success")
}

func (h *WalletHandler) ReleaseBalance(w http.ResponseWriter, r *http.Request) {
	var req model.UnlockBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	h.handleWalletOperation(w, r, req.UserID, req.Amount, req.Reference, h.service.ReleaseBalance, "balance release success")
}

func (h *WalletHandler) DebitBalance(w http.ResponseWriter, r *http.Request) {
	var req model.DeductBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	h.handleWalletOperation(w, r, req.UserID, req.Amount, req.Reference, h.service.DebitBalance, "balance debit success")
}

func (h *WalletHandler) CreditBalance(w http.ResponseWriter, r *http.Request) {
	var req model.CreditBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	h.handleWalletOperation(w, r, req.UserID, req.Amount, req.Reference, h.service.CreditBalance, "balance credit success")
}

func (h *WalletHandler) Refund(w http.ResponseWriter, r *http.Request) {
	var req model.RefundBalanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	h.handleWalletOperation(w, r, req.UserID, req.Amount, req.Reference, h.service.Refund, "refund success")
}

func (h *WalletHandler) handleWalletOperation(
	w http.ResponseWriter,
	r *http.Request,
	userID string,
	amount float64,
	reference string,
	op func(ctx context.Context, userID string, amount float64, referenceID string) (*model.WalletResult, error),
	successMessage string,
) {
	if h.service == nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "wallet service is not initialized")
		return
	}

	res, err := op(r.Context(), userID, amount, reference)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, successMessage, model.WalletTransactionResponse{
		TransactionID:  res.TransactionID,
		Status:         res.Status,
		CurrentBalance: res.CurrentBalance,
		Amount:         amount,
		CreatedAt:      time.Now().UTC(),
	})
}

// HealthCheck handles GET /health.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	utils.SuccessResponse(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"service": "wallet-service",
	})
}

// handleServiceError maps service/repository errors to HTTP responses.
func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidRequest):
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInsufficientBalance):
		utils.ErrorResponse(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrReferenceNotFound):
		utils.ErrorResponse(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidRefund):
		utils.ErrorResponse(w, http.StatusConflict, err.Error())
	case errors.Is(err, repository.ErrWalletNotFound):
		utils.ErrorResponse(w, http.StatusNotFound, err.Error())
	default:
		utils.ErrorResponse(w, http.StatusInternalServerError, "internal server error")
	}
}
