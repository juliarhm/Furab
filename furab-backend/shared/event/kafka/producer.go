// Package kafka provides a Kafka-based event publisher implementation.
// Used for high-throughput events like ride/food order events and location updates.
package kafka

import (
	"context"
	"encoding/json"

	"furab-backend/shared/event"

	"github.com/segmentio/kafka-go"
)

// Publisher implements event.Publisher using Apache Kafka.
type Publisher struct {
	writers map[string]*kafka.Writer
	brokers []string
}

// NewPublisher creates a new Kafka Publisher.
func NewPublisher(brokers []string) *Publisher {
	return &Publisher{
		writers: make(map[string]*kafka.Writer),
		brokers: brokers,
	}
}

// getWriter returns or creates a kafka.Writer for the given topic.
func (p *Publisher) getWriter(topic string) *kafka.Writer {
	if w, ok := p.writers[topic]; ok {
		return w
	}

	w := &kafka.Writer{
		Addr:     kafka.TCP(p.brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	p.writers[topic] = w
	return w
}

// Publish sends an event to the specified Kafka topic.
func (p *Publisher) Publish(ctx context.Context, topic string, evt *event.Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	writer := p.getWriter(topic)
	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(evt.ID),
		Value: data,
	})
}

// Close shuts down all Kafka writers gracefully.
func (p *Publisher) Close() error {
	for _, w := range p.writers {
		if err := w.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Subscriber implements event.Subscriber using Apache Kafka.
type Subscriber struct {
	readers map[string]*kafka.Reader
	brokers []string
	groupID string
}

// NewSubscriber creates a new Kafka Subscriber.
func NewSubscriber(brokers []string, groupID string) *Subscriber {
	return &Subscriber{
		readers: make(map[string]*kafka.Reader),
		brokers: brokers,
		groupID: groupID,
	}
}

// Subscribe registers an event handler for the specified Kafka topic.
func (s *Subscriber) Subscribe(ctx context.Context, topic string, handler event.EventHandler) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: s.brokers,
		Topic:   topic,
		GroupID: s.groupID,
	})
	s.readers[topic] = reader

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					continue
				}

				var evt event.Event
				if err := json.Unmarshal(msg.Value, &evt); err != nil {
					continue
				}

				if err := handler(ctx, &evt); err != nil {
					// Log error, implement retry logic as needed
					continue
				}
			}
		}
	}()

	return nil
}

// Close shuts down all Kafka readers gracefully.
func (s *Subscriber) Close() error {
	for _, r := range s.readers {
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}
