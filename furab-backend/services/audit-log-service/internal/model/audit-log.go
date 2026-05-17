package model

import "time"

// AuditLog represents the audit log entity in the system.
type AuditLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entity_id"`
	OldData   string    `json:"old_data"`
	NewData   string    `json:"new_data"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchAuditLogRequest represents the criteria for searching audit logs.
type SearchAuditLogRequest struct {
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}
