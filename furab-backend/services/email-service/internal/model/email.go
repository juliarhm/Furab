// Package model defines the domain models for email-service.
package model

import "time"

// SendEmailRequest represents direct send email request payload.
type SendEmailRequest struct {
	ReceiverEmail string                 `json:"receiver_email"`
	Subject       string                 `json:"subject"`
	Body          string                 `json:"body"`
	TemplateID    string                 `json:"template_id"` // Optional
	ReceiverID    string                 `json:"receiver_id"`
	ReferenceID   string                 `json:"reference_id"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// EventEmailRequest represents event-driven email request payload.
type EventEmailRequest struct {
	EventType     string                 `json:"event_type"`
	ReceiverEmail string                 `json:"receiver_email"`
	ReceiverID    string                 `json:"receiver_id"`
	ReferenceID   string                 `json:"reference_id"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// EmailLog stores email delivery logs for monitoring and audit.
type EmailLog struct {
	EmailID       string    `json:"email_id"`
	ReceiverEmail string    `json:"receiver_email"`
	Subject       string    `json:"subject"`
	Status        string    `json:"status"` // sent / failed
	Timestamp     time.Time `json:"timestamp"`
	ReceiverID    string    `json:"receiver_id"`
	ReferenceID   string    `json:"reference_id"`
}

// EmailResponse represents service response for email operations.
type EmailResponse struct {
	EmailID       string    `json:"email_id"`
	ReceiverEmail string    `json:"receiver_email"`
	Subject       string    `json:"subject"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}

// EmailTemplate represents a reusable email template.
type EmailTemplate struct {
	TemplateID string `json:"template_id"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
}

// EmailResult is a wrapper for EmailResponse
type EmailResult struct {
	Status  string
	Message string
	Data    *EmailResponse
}

