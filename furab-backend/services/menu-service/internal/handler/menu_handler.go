package handler

import (
	"net/http"
	"strconv"

	"furab-backend/services/menu-service/internal/model"
	"furab-backend/services/menu-service/internal/service"
	"furab-backend/shared/utils"
	"github.com/go-chi/chi/v5"
)

// MenuHandler handles HTTP requests for menus.
type MenuHandler struct {
	service service.MenuService
}

// NewMenuHandler creates a new instance of menu handler.
func NewMenuHandler(s service.MenuService) *MenuHandler {
	return &MenuHandler{service: s}
}

// Routes returns the chi router for menu service.
func (h *MenuHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/", h.Search)
	return r
}

// Create handles the POST / request.
func (h *MenuHandler) Create(w http.ResponseWriter, r *http.Request) {
	var m model.Menu
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.service.CreateMenu(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusCreated, m)
}

// GetByID handles the GET /{id} request.
func (h *MenuHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.service.GetMenu(r.Context(), id)
	if err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "menu not found")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Update handles the PUT /{id} request.
func (h *MenuHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var m model.Menu
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	m.ID = id

	if err := h.service.UpdateMenu(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Delete handles the DELETE /{id} request.
func (h *MenuHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteMenu(r.Context(), id); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, "menu deleted successfully", nil)
}

// Search handles the GET / request with query parameters.
func (h *MenuHandler) Search(w http.ResponseWriter, r *http.Request) {
	merchantID := r.URL.Query().Get("merchant_id")
	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
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

	req := model.SearchMenuRequest{
		MerchantID: merchantID,
		Query:      query,
		Category:   category,
		Limit:      limit,
		Offset:     offset,
	}

	menus, total, err := h.service.SearchMenus(r.Context(), req)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.PaginatedSuccessResponse(w, menus, page, limit, total)
}
