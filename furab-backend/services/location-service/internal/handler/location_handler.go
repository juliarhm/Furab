// Package handler provides HTTP handlers for location-service.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"furab-backend/services/location-service/internal/model"
	"furab-backend/services/location-service/internal/service"
	"furab-backend/shared/utils"

	"github.com/go-chi/chi/v5"
)

// LocationHandler handles HTTP requests for location-service.
type LocationHandler struct {
	service service.LocationService
}

// NewLocationHandler creates a new LocationHandler.
func NewLocationHandler(svc service.LocationService) *LocationHandler {
	return &LocationHandler{
		service: svc,
	}
}

// RegisterRoutes registers all location-service routes.
func (h *LocationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/locations", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.SuccessResponse(w, http.StatusOK, map[string]string{
				"status":  "healthy",
				"service": "location-service",
			})
		})

		r.Post("/update", h.UpdateLocation)
		r.Post("/status", h.UpdateStatus)
		r.Get("/search", h.SearchNearbyDrivers)
		r.Get("/track/{driver_id}", h.TrackDriver)
	})
}

func (h *LocationHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	if err := h.service.UpdateDriverLocation(r.Context(), req); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to update location")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, map[string]string{"message": "Location updated successfully"})
}

func (h *LocationHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.DriverStatus != "available" && req.DriverStatus != "busy" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid driver_status")
		return
	}

	if err := h.service.UpdateDriverStatus(r.Context(), req); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to update status")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, map[string]string{"message": "Status updated successfully"})
}

func (h *LocationHandler) SearchNearbyDrivers(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("latitude_origin")
	lonStr := r.URL.Query().Get("longitude_origin")
	radiusStr := r.URL.Query().Get("radius")

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid latitude_origin")
		return
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid longitude_origin")
		return
	}
	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid radius")
		return
	}

	req := model.SearchDriverRequest{
		LatitudeOrigin:  lat,
		LongitudeOrigin: lon,
		Radius:          radius,
	}

	drivers, err := h.service.FindNearbyDrivers(r.Context(), req)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Failed to search drivers")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, drivers)
}

func (h *LocationHandler) TrackDriver(w http.ResponseWriter, r *http.Request) {
	driverID := chi.URLParam(r, "driver_id")
	if driverID == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "driver_id is required")
		return
	}

	loc, err := h.service.GetDriverLocation(r.Context(), driverID, "")
	if err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "Driver location not found")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, loc)
}
