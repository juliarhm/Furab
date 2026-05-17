// Package repository provides data access layer for chat-service.
package repository

import (
	"context"

	"furab-backend/services/chat-service/internal/model"
)

// ChatRepository defines the interface for chat-service data access.
type ChatRepository interface {
	SaveMessage(ctx context.Context, msg model.Message) error
	UpdateMessageStatus(ctx context.Context, messageID string, status string) error
	GetMessagesByOrderID(ctx context.Context, orderID string) ([]model.Message, error)
	GetChatSession(ctx context.Context, orderID string) (*model.ChatSession, error)
}

// postgresChatRepository implements ChatRepository using PostgreSQL.
type postgresChatRepository struct {
	// TODO: add *sql.DB field
}

// NewPostgresChatRepository creates a new PostgreSQL-based repository.
func NewPostgresChatRepository() ChatRepository {
	return &postgresChatRepository{}
}

func (r *postgresChatRepository) SaveMessage(ctx context.Context, msg model.Message) error {
	return nil
}

func (r *postgresChatRepository) UpdateMessageStatus(ctx context.Context, messageID string, status string) error {
	return nil
}

func (r *postgresChatRepository) GetMessagesByOrderID(ctx context.Context, orderID string) ([]model.Message, error) {
	return nil, nil
}

func (r *postgresChatRepository) GetChatSession(ctx context.Context, orderID string) (*model.ChatSession, error) {
	return nil, nil
}
