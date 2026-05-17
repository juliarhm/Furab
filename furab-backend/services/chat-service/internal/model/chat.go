// Package model defines the domain models for chat-service.
package model

import "time"

// ChatSession represents an active chat session.
type ChatSession struct {
	OrderID   string     `json:"order_id"`
	UserID    string     `json:"user_id"`
	DriverID  string     `json:"driver_id"`
	CreatedAt time.Time  `json:"created_at"`
	ClosedAt  *time.Time `json:"closed_at"` // nil if active
}

// Message represents a chat message.
type Message struct {
	MessageID  string    `json:"message_id"`
	OrderID    string    `json:"order_id"`
	SenderID   string    `json:"sender_id"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
	ReadStatus string    `json:"read_status"` // sent, delivered, read
}

// SendMessageRequest represents the request to send a message.
type SendMessageRequest struct {
	OrderID     string `json:"order_id"`
	SenderID    string `json:"sender_id"`
	SenderType  string `json:"sender_type"` // user, driver
	ReceiverID  string `json:"receiver_id"`
	MessageText string `json:"message_text"`
}

// ReadReceiptRequest represents the request to update message read status.
type ReadReceiptRequest struct {
	MessageID string `json:"message_id"`
	OrderID   string `json:"order_id"`
	Status    string `json:"status"` // delivered, read
}

// SendMessageResponse represents the response when sending a message.
type SendMessageResponse struct {
	MessageID   string    `json:"message_id"`
	SenderID    string    `json:"sender_id"`
	MessageText string    `json:"message_text"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}
