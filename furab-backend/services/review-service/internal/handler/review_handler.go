package handler

import (
	"net/http"
	"strconv"

	"furab-backend/services/review-service/internal/model"
	"furab-backend/services/review-service/internal/service"
	"furab-backend/shared/utils"
	"github.com/go-chi/chi/v5"
)

// ReviewHandler handles HTTP requests for reviews.
type ReviewHandler struct {
	service service.ReviewService
}

// NewReviewHandler creates a new instance of review handler.
func NewReviewHandler(s service.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: s}
}

// Routes returns the chi router for review service.
func (h *ReviewHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/", h.Search)
	return r
}

// Create handles the POST / request.
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	var m model.Review
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.service.CreateReview(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusCreated, m)
}

// GetByID handles the GET /{id} request.
func (h *ReviewHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.service.GetReview(r.Context(), id)
	if err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "review not found")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Update handles the PUT /{id} request.
func (h *ReviewHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var m model.Review
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	m.ID = id

	if err := h.service.UpdateReview(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Delete handles the DELETE /{id} request.
func (h *ReviewHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteReview(r.Context(), id); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, "review deleted successfully", nil)
}

// Search handles the GET / request with query parameters.
func (h *ReviewHandler) Search(w http.ResponseWriter, r *http.Request) {
	merchantID := r.URL.Query().Get("merchant_id")
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

	req := model.SearchReviewRequest{
		MerchantID: merchantID,
		UserID:     userID,
		Limit:      limit,
		Offset:     offset,
	}

	reviews, total, err := h.service.SearchReviews(r.Context(), req)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.PaginatedSuccessResponse(w, reviews, page, limit, total)
}
