// Package model defines the domain models for notification-service.
package model

import "time"

// EventNotificationRequest represents a request to send a notification.
type EventNotificationRequest struct {
	EventType   string `json:"event_type"`
	ReceiverID  string `json:"receiver_id"`
	ReferenceID string `json:"reference_id"`
	Channel     string `json:"channel"` // push, email
}

// NotificationLog represents the stored log of a sent notification.
type NotificationLog struct {
	NotificationID string    `json:"notification_id"`
	ReceiverID     string    `json:"receiver_id"`
	Title          string    `json:"title"`
	Message        string    `json:"message"`
	Channel        string    `json:"channel"`
	Status         string    `json:"status"` // sent, failed
	ReferenceID    string    `json:"reference_id"`
	Timestamp      time.Time `json:"timestamp"`
}

// NotifTemplate represents a message template for an event.
type NotifTemplate struct {
	EventType       string `json:"event_type"`
	TitleTemplate   string `json:"title_template"`
	MessageTemplate string `json:"message_template"`
}

// NotificationResponse represents the response when processing a notification.
type NotificationResponse struct {
	NotificationID string    `json:"notification_id"`
	ReceiverID     string    `json:"receiver_id"`
	Channel        string    `json:"channel"`
	Status         string    `json:"status"`
	Message        string    `json:"message"`
	Timestamp      time.Time `json:"timestamp"`
}
