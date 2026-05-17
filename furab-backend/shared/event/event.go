// Package event provides event-driven messaging interfaces and models
// for inter-service communication in Furab microservices.
package event

import (
	"context"
	"encoding/json"
	"time"
)

// Event represents a domain event that can be published and consumed.
type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// NewEvent creates a new Event with the given type, source, and payload.
func NewEvent(eventType, source string, payload interface{}) (*Event, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UTC(),
		Payload:   data,
	}, nil
}

// UnmarshalPayload decodes the event payload into the given struct.
func (e *Event) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal(e.Payload, v)
}

// Publisher defines the interface for publishing events.
type Publisher interface {
	// Publish sends an event to the specified topic.
	Publish(ctx context.Context, topic string, event *Event) error
	// Close gracefully shuts down the publisher.
	Close() error
}

// Subscriber defines the interface for subscribing to events.
type Subscriber interface {
	// Subscribe registers a handler for events on the specified topic.
	Subscribe(ctx context.Context, topic string, handler EventHandler) error
	// Close gracefully shuts down the subscriber.
	Close() error
}

// EventHandler is a function that processes received events.
type EventHandler func(ctx context.Context, event *Event) error

// --- Event Topics (constants) ---

const (
	// Ride events
	TopicRideCreated          = "ride.created"
	TopicRideAssigned         = "ride.assigned"
	TopicRidePickingUp        = "ride.picking_up"
	TopicRideOnTheWay         = "ride.on_the_way"
	TopicRideStarted          = "ride.started" // kept for backward compat
	TopicRideCompleted        = "ride.completed"
	TopicRideCancelled        = "ride.cancelled"
	TopicRideDriverCancelled  = "ride.driver_cancelled"

	// Food events
	TopicFoodCreated  = "food.created"
	TopicFoodConfirmed = "food.confirmed"
	TopicFoodPreparing = "food.preparing"
	TopicFoodReady     = "food.ready"
	TopicFoodPickedUp  = "food.picked_up"
	TopicFoodCompleted = "food.completed"

	// Payment events
	TopicPaymentAuthorized = "payment.authorized"
	TopicPaymentCaptured   = "payment.captured"
	TopicPaymentFailed     = "payment.failed"

	// Wallet events
	TopicWalletLock   = "wallet.lock"
	TopicWalletUnlock = "wallet.unlock"
	TopicWalletDeduct = "wallet.deduct"

	// Settlement events
	TopicSettlementCompleted = "settlement.completed"

	// Notification events
	TopicNotificationSent = "notification.sent"
)

// generateEventID generates a unique event ID.
// In production, use a proper UUID library.
func generateEventID() string {
	return time.Now().UTC().Format("20060102150405.000000000")
}
