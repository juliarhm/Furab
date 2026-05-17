// Package service implements the business logic for emergency-service.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"furab-backend/services/emergency-service/internal/model"
	"furab-backend/services/emergency-service/internal/repository"

	"github.com/google/uuid"
)

// EmergencyService defines the interface for emergency-service business logic.
type EmergencyService interface {
	TriggerEmergency(ctx context.Context, req model.TriggerEmergencyRequest) (*model.TriggerEmergencyResponse, error)
}

// LocationClient defines location-service contract used by emergency service.
type LocationClient interface {
	GetLastLocation(ctx context.Context, actorID, actorType string) (*model.EmergencyLocation, error)
}

// ActorClient defines user/driver service contract used by emergency service.
type ActorClient interface {
	ValidateActor(ctx context.Context, actorID, actorType string) (bool, error)
	ValidateOrder(ctx context.Context, orderID string) (bool, error)
	GetEmergencyContact(ctx context.Context, actorID, actorType string) (*model.EmergencyContact, error)
}

// NotificationClient defines notification-service contract used by emergency service.
type NotificationClient interface {
	SendNotification(ctx context.Context, notification model.EmergencyNotification) error
	SendEmergencyContact(ctx context.Context, contact model.EmergencyContact, notification model.EmergencyNotification) error
}

// emergencyServiceImpl is the concrete implementation of EmergencyService.
type emergencyServiceImpl struct {
	repo              repository.EmergencyRepository
	locationClient    LocationClient
	actorClient       ActorClient
	notificationClient NotificationClient
	idGenerator       func() string
}

// NewEmergencyService creates a new EmergencyService.
func NewEmergencyService() EmergencyService {
	return &emergencyServiceImpl{
		idGenerator: func() string { return uuid.NewString() },
	}
}

// NewEmergencyServiceWithDependencies creates emergency service with mocked/real dependencies.
func NewEmergencyServiceWithDependencies(
	repo repository.EmergencyRepository,
	locationClient LocationClient,
	actorClient ActorClient,
	notificationClient NotificationClient,
) EmergencyService {
	return &emergencyServiceImpl{
		repo:               repo,
		locationClient:     locationClient,
		actorClient:        actorClient,
		notificationClient: notificationClient,
		idGenerator:        func() string { return uuid.NewString() },
	}
}

func (s *emergencyServiceImpl) TriggerEmergency(ctx context.Context, req model.TriggerEmergencyRequest) (*model.TriggerEmergencyResponse, error) {
	if req.ActorID == "" || (req.ActorType != "user" && req.ActorType != "driver") {
		return &model.TriggerEmergencyResponse{
			Status:  "failed",
			Message: "invalid actor",
		}, errors.New("invalid actor")
	}
	if s.repo == nil || s.locationClient == nil || s.actorClient == nil || s.notificationClient == nil {
		return nil, errors.New("service dependencies are not configured")
	}

	validActor, err := s.actorClient.ValidateActor(ctx, req.ActorID, req.ActorType)
	if err != nil || !validActor {
		return &model.TriggerEmergencyResponse{
			Status:  "failed",
			Message: "invalid actor",
		}, errors.New("invalid actor")
	}

	if req.OrderID != "" {
		// Order validation is best-effort: emergency must still be processed even when invalid.
		_, _ = s.actorClient.ValidateOrder(ctx, req.OrderID)
	}

	location, err := s.locationClient.GetLastLocation(ctx, req.ActorID, req.ActorType)
	if err != nil || location == nil {
		// Fallback to request payload (including zero-values) when location-service is unavailable.
		location = &model.EmergencyLocation{
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
			Timestamp: req.Timestamp,
		}
	}

	event := model.EmergencyEvent{
		EmergencyID:   s.idGenerator(),
		ActorID:       req.ActorID,
		ActorType:     req.ActorType,
		OrderID:       req.OrderID,
		Latitude:      location.Latitude,
		Longitude:     location.Longitude,
		EmergencyType: req.EmergencyType,
		Status:        "active",
		CreatedAt:     req.Timestamp,
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	if err := s.repo.SaveEmergencyEvent(ctx, event); err != nil {
		return nil, err
	}

	contact, _ := s.actorClient.GetEmergencyContact(ctx, req.ActorID, req.ActorType)
	receiverID := req.ActorID
	if contact != nil && contact.ReceiverID != "" {
		receiverID = contact.ReceiverID
	}

	notification := model.EmergencyNotification{
		ReceiverID:  receiverID,
		Title:       "Emergency Alert",
		Message:     fmt.Sprintf("Emergency reported by %s", req.ActorType),
		Priority:    "high",
		Latitude:    location.Latitude,
		Longitude:   location.Longitude,
		LocationURL: fmt.Sprintf("https://maps.google.com/?q=%f,%f", location.Latitude, location.Longitude),
		Timestamp:   event.CreatedAt,
	}
	// Notification failures must not stop emergency flow.
	_ = s.notificationClient.SendNotification(ctx, notification)
	if contact != nil {
		_ = s.notificationClient.SendEmergencyContact(ctx, *contact, notification)
	}

	return &model.TriggerEmergencyResponse{
		Status:      "success",
		Message:     "emergency created",
		EmergencyID: event.EmergencyID,
	}, nil
}
