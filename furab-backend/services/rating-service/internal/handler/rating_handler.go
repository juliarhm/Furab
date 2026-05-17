package handler

import (
	"net/http"
	"strconv"

	"furab-backend/services/rating-service/internal/model"
	"furab-backend/services/rating-service/internal/service"
	"furab-backend/shared/utils"
	"github.com/go-chi/chi/v5"
)

// RatingHandler handles HTTP requests for ratings.
type RatingHandler struct {
	service service.RatingService
}

// NewRatingHandler creates a new instance of rating handler.
func NewRatingHandler(s service.RatingService) *RatingHandler {
	return &RatingHandler{service: s}
}

// Routes returns the chi router for rating service.
func (h *RatingHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/", h.Search)
	return r
}

// Create handles the POST / request.
func (h *RatingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var m model.Rating
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.service.CreateRating(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusCreated, m)
}

// GetByID handles the GET /{id} request.
func (h *RatingHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.service.GetRating(r.Context(), id)
	if err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "rating not found")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Update handles the PUT /{id} request.
func (h *RatingHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var m model.Rating
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	m.ID = id

	if err := h.service.UpdateRating(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Delete handles the DELETE /{id} request.
func (h *RatingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteRating(r.Context(), id); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, "rating deleted successfully", nil)
}

// Search handles the GET / request with query parameters.
func (h *RatingHandler) Search(w http.ResponseWriter, r *http.Request) {
	targetID := r.URL.Query().Get("target_id")
	targetType := r.URL.Query().Get("target_type")
	userID := r.URL.Query().Get("user_id")
	limitStr := r.URL.Query().Get("limit")
	pageStr := r.URL.Query().Get("page")

	limit, _ := strconv.Atoi(limitStr)
	page, _ := strconv.Atoi(pageStr)

	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	req := model.SearchRatingRequest{
		TargetID:   targetID,
		TargetType: targetType,
		UserID:     userID,
		Limit:      limit,
		Offset:     offset,
	}

	ratings, total, err := h.service.SearchRatings(r.Context(), req)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.PaginatedSuccessResponse(w, ratings, page, limit, total)
}
