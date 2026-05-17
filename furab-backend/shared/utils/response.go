// Package utils provides utility functions shared across all Furab microservices.
package utils

import (
	"encoding/json"
	"net/http"
)

// APIResponse represents the standard JSON response format for all API endpoints.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse wraps API response with pagination metadata.
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
	Error      string      `json:"error,omitempty"`
	Message    string      `json:"message,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Pagination holds pagination metadata.
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

// WriteJSON writes a JSON response to the http.ResponseWriter.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// SuccessResponse sends a successful JSON response.
func SuccessResponse(w http.ResponseWriter, status int, data interface{}) {
	WriteJSON(w, status, APIResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessMessageResponse sends a successful JSON response with a message.
func SuccessMessageResponse(w http.ResponseWriter, status int, message string, data interface{}) {
	WriteJSON(w, status, APIResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// ErrorResponse sends an error JSON response.
func ErrorResponse(w http.ResponseWriter, status int, err string) {
	WriteJSON(w, status, APIResponse{
		Success: false,
		Error:   err,
	})
}

// PaginatedSuccessResponse sends a paginated success JSON response.
func PaginatedSuccessResponse(w http.ResponseWriter, data interface{}, page, limit, totalItems int) {
	totalPages := totalItems / limit
	if totalItems%limit > 0 {
		totalPages++
	}

	WriteJSON(w, http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    data,
		Pagination: &Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	})
}

// DecodeJSON decodes the request body into the given struct.
func DecodeJSON(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}
