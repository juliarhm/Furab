package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/chat-service/internal/model"
	"github.com/google/uuid"
)

var (
	ErrInvalidRequest   = errors.New("invalid request")
	ErrInvalidSender    = errors.New("invalid sender")
	ErrSessionClosed    = errors.New("chat session is closed")
)

// ChatRepository defines the interface for chat data access.
type ChatRepository interface {
	SaveMessage(ctx context.Context, msg *model.Message) error
	UpdateMessageStatus(ctx context.Context, messageID string, status string) error
	GetMessagesByOrderID(ctx context.Context, orderID string) ([]model.Message, error)
	CloseSession(ctx context.Context, orderID string) error
}

// UserServiceClient defines the interface for communicating with user-service.
type UserServiceClient interface {
	ValidateUser(ctx context.Context, userID string) (bool, error)
}

// DriverServiceClient defines the interface for communicating with driver-service.
type DriverServiceClient interface {
	ValidateDriver(ctx context.Context, driverID string) (bool, error)
}

// NotificationClient defines the interface for communicating with notification-service.
type NotificationClient interface {
	SendNotification(ctx context.Context, receiverID string, message string) error
}

// ChatService defines the interface for chat business logic.
type ChatService interface {
	SendMessage(ctx context.Context, req model.SendMessageRequest) (*model.SendMessageResponse, error)
	UpdateMessageStatus(ctx context.Context, req model.ReadReceiptRequest) error
	GetChatHistory(ctx context.Context, orderID string) ([]model.Message, error)
	CloseChatSession(ctx context.Context, orderID string) error
}

type chatServiceImpl struct {
	repo         ChatRepository
	userClient   UserServiceClient
	driverClient DriverServiceClient
	notifClient  NotificationClient
}

// NewChatService creates a new ChatService.
func NewChatService(
	repo ChatRepository,
	userClient UserServiceClient,
	driverClient DriverServiceClient,
	notifClient NotificationClient,
) ChatService {
	return &chatServiceImpl{
		repo:         repo,
		userClient:   userClient,
		driverClient: driverClient,
		notifClient:  notifClient,
	}
}

func (s *chatServiceImpl) SendMessage(ctx context.Context, req model.SendMessageRequest) (*model.SendMessageResponse, error) {
	if req.OrderID == "" || req.SenderID == "" || req.SenderType == "" || req.ReceiverID == "" || req.MessageText == "" {
		return nil, ErrInvalidRequest
	}

	if req.SenderType == "user" {
		valid, err := s.userClient.ValidateUser(ctx, req.SenderID)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidSender
		}
	} else if req.SenderType == "driver" {
		valid, err := s.driverClient.ValidateDriver(ctx, req.SenderID)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidSender
		}
	} else {
		return nil, ErrInvalidRequest
	}

	now := time.Now().UTC()
	msg := &model.Message{
		MessageID:  uuid.New().String(),
		OrderID:    req.OrderID,
		SenderID:   req.SenderID,
		Content:    req.MessageText,
		Timestamp:  now,
		ReadStatus: "sent",
	}

	if err := s.repo.SaveMessage(ctx, msg); err != nil {
		return nil, err
	}

	if err := s.notifClient.SendNotification(ctx, req.ReceiverID, req.MessageText); err != nil {
		return nil, err
	}

	return &model.SendMessageResponse{
		MessageID:   msg.MessageID,
		SenderID:    msg.SenderID,
		MessageText: msg.Content,
		Timestamp:   msg.Timestamp,
		Status:      msg.ReadStatus,
	}, nil
}

func (s *chatServiceImpl) UpdateMessageStatus(ctx context.Context, req model.ReadReceiptRequest) error {
	if req.MessageID == "" || req.OrderID == "" || req.Status == "" {
		return ErrInvalidRequest
	}

	if req.Status != "sent" && req.Status != "delivered" && req.Status != "read" {
		return ErrInvalidRequest
	}

	return s.repo.UpdateMessageStatus(ctx, req.MessageID, req.Status)
}

func (s *chatServiceImpl) GetChatHistory(ctx context.Context, orderID string) ([]model.Message, error) {
	if orderID == "" {
		return nil, ErrInvalidRequest
	}

	messages, err := s.repo.GetMessagesByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if messages == nil {
		return []model.Message{}, nil
	}

	return messages, nil
}

func (s *chatServiceImpl) CloseChatSession(ctx context.Context, orderID string) error {
	if orderID == "" {
		return ErrInvalidRequest
	}

	return s.repo.CloseSession(ctx, orderID)
}