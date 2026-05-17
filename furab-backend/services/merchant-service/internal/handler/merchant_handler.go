package handler

import (
	"net/http"
	"strconv"

	"furab-backend/services/merchant-service/internal/model"
	"furab-backend/services/merchant-service/internal/service"
	"furab-backend/shared/utils"
	"github.com/go-chi/chi/v5"
)

// MerchantHandler handles HTTP requests for merchants.
type MerchantHandler struct {
	service service.MerchantService
}

// NewMerchantHandler creates a new instance of merchant handler.
func NewMerchantHandler(s service.MerchantService) *MerchantHandler {
	return &MerchantHandler{service: s}
}

// Routes returns the chi router for merchant service.
func (h *MerchantHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/", h.Search)
	return r
}

// Create handles the POST / request.
func (h *MerchantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var m model.Merchant
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.service.CreateMerchant(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusCreated, m)
}

// GetByID handles the GET /{id} request.
func (h *MerchantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.service.GetMerchant(r.Context(), id)
	if err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "merchant not found")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Update handles the PUT /{id} request.
func (h *MerchantHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var m model.Merchant
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	m.ID = id

	if err := h.service.UpdateMerchant(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Delete handles the DELETE /{id} request.
func (h *MerchantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteMerchant(r.Context(), id); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, "merchant deleted successfully", nil)
}

// Search handles the GET / request with query parameters.
func (h *MerchantHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
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

	req := model.SearchMerchantRequest{
		Query:  query,
		Limit:  limit,
		Offset: offset,
	}

	merchants, total, err := h.service.SearchMerchants(r.Context(), req)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.PaginatedSuccessResponse(w, merchants, page, limit, total)
}
