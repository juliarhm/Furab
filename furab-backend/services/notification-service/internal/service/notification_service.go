// Package service implements the business logic for notification-service.
package service

import (
	"context"
	"errors"
	"time"

	"furab-backend/services/notification-service/internal/model"
	"furab-backend/services/notification-service/internal/repository"

	"github.com/google/uuid"
)

// EmailClient defines the interface for calling the Email Service.
type EmailClient interface {
	SendEmail(ctx context.Context, receiverID string, title string, message string) error
}

// NotificationService defines the interface for notification-service business logic.
type NotificationService interface {
	SendNotification(ctx context.Context, req model.EventNotificationRequest) (*model.NotificationResponse, error)
	GenerateNotificationTemplate(ctx context.Context, eventType string) (*model.NotifTemplate, error)
}

// notificationServiceImpl is the concrete implementation of NotificationService.
type notificationServiceImpl struct {
	repo       repository.NotificationRepository
	emailClient EmailClient
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(repo repository.NotificationRepository, emailClient EmailClient) NotificationService {
	return &notificationServiceImpl{
		repo:       repo,
		emailClient: emailClient,
	}
}

func (s *notificationServiceImpl) SendNotification(ctx context.Context, req model.EventNotificationRequest) (*model.NotificationResponse, error) {
	if req.EventType == "" {
		return nil, errors.New("event type is required")
	}
	if req.ReceiverID == "" {
		return nil, errors.New("receiver_id is required")
	}
	if req.ReferenceID == "" {
		return nil, errors.New("reference_id is required")
	}
	if req.Channel == "" {
		return nil, errors.New("channel is required")
	}
	if req.Channel != "push" && req.Channel != "email" {
		return nil, errors.New("invalid channel")
	}

	template, err := s.GenerateNotificationTemplate(ctx, req.EventType)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	notifID := uuid.New().String()

	log := model.NotificationLog{
		NotificationID: notifID,
		ReceiverID:     req.ReceiverID,
		Title:          template.TitleTemplate,
		Message:        template.MessageTemplate,
		Channel:        req.Channel,
		ReferenceID:    req.ReferenceID,
		Timestamp:      now,
		Status:         "sent",
	}

	if err := s.repo.SaveNotificationLog(ctx, log); err != nil {
		return nil, err
	}

	if req.Channel == "email" {
		if err := s.emailClient.SendEmail(ctx, req.ReceiverID, template.TitleTemplate, template.MessageTemplate); err != nil {
			log.Status = "failed"
			s.repo.SaveNotificationLog(ctx, log) // Update status
			return nil, err
		}
	}

	return &model.NotificationResponse{
		NotificationID: notifID,
		ReceiverID:     req.ReceiverID,
		Channel:        req.Channel,
		Status:         "success",
		Message:        "notifikasi berhasil",
		Timestamp:      now,
	}, nil
}

func (s *notificationServiceImpl) GenerateNotificationTemplate(ctx context.Context, eventType string) (*model.NotifTemplate, error) {
	template, err := s.repo.GetTemplateByEventType(ctx, eventType)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, errors.New("template not found")
	}
	return template, nil
}
