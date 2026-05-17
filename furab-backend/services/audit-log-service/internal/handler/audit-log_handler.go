package handler

import (
	"net/http"
	"strconv"
	"time"

	"furab-backend/services/audit-log-service/internal/model"
	"furab-backend/services/audit-log-service/internal/service"
	"furab-backend/shared/utils"
	"github.com/go-chi/chi/v5"
)

// AuditLogHandler handles HTTP requests for audit logs.
type AuditLogHandler struct {
	service service.AuditLogService
}

// NewAuditLogHandler creates a new instance of audit log handler.
func NewAuditLogHandler(s service.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{service: s}
}

// Routes returns the chi router for audit log service.
func (h *AuditLogHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/", h.Search)
	return r
}

// Create handles the POST / request.
func (h *AuditLogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var m model.AuditLog
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.service.CreateAuditLog(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusCreated, m)
}

// GetByID handles the GET /{id} request.
func (h *AuditLogHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := h.service.GetAuditLog(r.Context(), id)
	if err != nil {
		utils.ErrorResponse(w, http.StatusNotFound, "audit log not found")
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Update handles the PUT /{id} request.
func (h *AuditLogHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var m model.AuditLog
	if err := utils.DecodeJSON(r, &m); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	m.ID = id

	if err := h.service.UpdateAuditLog(r.Context(), &m); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(w, http.StatusOK, m)
}

// Delete handles the DELETE /{id} request.
func (h *AuditLogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteAuditLog(r.Context(), id); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessMessageResponse(w, http.StatusOK, "audit log deleted successfully", nil)
}

// Search handles the GET / request with query parameters.
func (h *AuditLogHandler) Search(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	action := r.URL.Query().Get("action")
	entity := r.URL.Query().Get("entity")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
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

	var startDate, endDate time.Time
	if startDateStr != "" {
		startDate, _ = time.Parse(time.RFC3339, startDateStr)
	}
	if endDateStr != "" {
		endDate, _ = time.Parse(time.RFC3339, endDateStr)
	}

	req := model.SearchAuditLogRequest{
		UserID:    userID,
		Action:    action,
		Entity:    entity,
		StartDate: startDate,
		EndDate:   endDate,
		Limit:     limit,
		Offset:    offset,
	}

	logs, total, err := h.service.SearchAuditLogs(r.Context(), req)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.PaginatedSuccessResponse(w, logs, page, limit, total)
}
