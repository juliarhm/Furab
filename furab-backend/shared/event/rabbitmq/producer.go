// Package rabbitmq provides a RabbitMQ-based event publisher implementation.
// Used for transactional events like payments, notifications, and emails.
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"furab-backend/shared/event"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher implements event.Publisher using RabbitMQ.
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewPublisher creates a new RabbitMQ Publisher.
func NewPublisher(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &Publisher{
		conn:    conn,
		channel: ch,
	}, nil
}

// Publish sends an event to the specified RabbitMQ exchange/queue.
func (p *Publisher) Publish(ctx context.Context, topic string, evt *event.Event) error {
	// Declare exchange (topic type for flexible routing)
	err := p.channel.ExchangeDeclare(
		topic,   // exchange name
		"topic", // exchange type
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return p.channel.PublishWithContext(
		ctx,
		topic, // exchange
		topic, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
			MessageId:    evt.ID,
		},
	)
}

// Close shuts down the RabbitMQ connection gracefully.
func (p *Publisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}

// Subscriber implements event.Subscriber using RabbitMQ.
type Subscriber struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewSubscriber creates a new RabbitMQ Subscriber.
func NewSubscriber(url string) (*Subscriber, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &Subscriber{
		conn:    conn,
		channel: ch,
	}, nil
}

// Subscribe registers an event handler for the specified RabbitMQ topic.
func (s *Subscriber) Subscribe(ctx context.Context, topic string, handler event.EventHandler) error {
	// Declare exchange
	err := s.channel.ExchangeDeclare(topic, "topic", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	q, err := s.channel.QueueDeclare(
		fmt.Sprintf("%s.queue", topic),
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = s.channel.QueueBind(q.Name, topic, topic, false, nil)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Start consuming
	msgs, err := s.channel.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to consume: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				var evt event.Event
				if err := json.Unmarshal(msg.Body, &evt); err != nil {
					msg.Nack(false, true)
					continue
				}
				if err := handler(ctx, &evt); err != nil {
					msg.Nack(false, true)
					continue
				}
				msg.Ack(false)
			}
		}
	}()

	return nil
}

// Close shuts down the RabbitMQ connection gracefully.
func (s *Subscriber) Close() error {
	if err := s.channel.Close(); err != nil {
		return err
	}
	return s.conn.Close()
}
