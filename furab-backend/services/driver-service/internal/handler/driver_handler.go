// Package handler provides HTTP handlers for driver-service API endpoints.
package handler

import (
	"encoding/json"
	"net/http"

	"furab-backend/services/driver-service/internal/model"
	"furab-backend/services/driver-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// DriverHandler handles HTTP requests for driver operations.
type DriverHandler struct {
	service service.DriverService
}

// NewDriverHandler creates a new DriverHandler with the given service.
func NewDriverHandler(svc service.DriverService) *DriverHandler {
	return &DriverHandler{service: svc}
}

// RegisterRoutes registers all driver routes on the given chi router.
func (h *DriverHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/drivers", func(r chi.Router) {
		r.Post("/", h.CreateDriver)
		r.Get("/{driverID}", h.GetDriver)
		r.Put("/{driverID}", h.UpdateDriver)
		r.Put("/{driverID}/status", h.UpdateStatus)
		r.Put("/{driverID}/location", h.UpdateLocation)
	})
}

// CreateDriver handles POST /api/v1/drivers
func (h *DriverHandler) CreateDriver(w http.ResponseWriter, r *http.Request) {
	var req model.CreateDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.CreateDriver(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusCreated, resp.Message, resp)
}

// GetDriver handles GET /api/v1/drivers/{driverID}
func (h *DriverHandler) GetDriver(w http.ResponseWriter, r *http.Request) {
	driverID := chi.URLParam(r, "driverID")

	driver, err := h.service.GetDriver(r.Context(), driverID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessResponse(w, http.StatusOK, driver)
}

// UpdateDriver handles PUT /api/v1/drivers/{driverID}
func (h *DriverHandler) UpdateDriver(w http.ResponseWriter, r *http.Request) {
	driverID := chi.URLParam(r, "driverID")

	var req model.UpdateDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.UpdateDriver(r.Context(), driverID, &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, resp.Message, resp)
}

// UpdateStatus handles PUT /api/v1/drivers/{driverID}/status
func (h *DriverHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	driverID := chi.URLParam(r, "driverID")

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.UpdateStatus(r.Context(), driverID, body.Status)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, resp.Message, resp)
}

// UpdateLocation handles PUT /api/v1/drivers/{driverID}/location
func (h *DriverHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	driverID := chi.URLParam(r, "driverID")

	var body struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	resp, err := h.service.UpdateLocation(r.Context(), driverID, body.Latitude, body.Longitude)
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
		"service": "driver-service",
	})
}

// handleServiceError maps service errors to HTTP responses.
func handleServiceError(w http.ResponseWriter, err error) {
	switch err {
	case service.ErrDriverNotFound:
		utils.ErrorResponse(w, http.StatusNotFound, "driver not found")
	case service.ErrInvalidRequest:
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
	case service.ErrStatusInvalid:
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid driver status")
	case service.ErrLocationInvalid:
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid location coordinates")
	default:
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
	}
}
