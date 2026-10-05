package xmq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Event struct {
	EventID   string         `json:"event_id"`
	EventType string         `json:"event_type"`
	TraceID   string         `json:"trace_id"`
	UserID    uint64         `json:"user_id"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

type Producer interface {
	Publish(ctx context.Context, event Event) error
	Close() error
}

type RabbitProducer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	enabled  bool
}

func NewRabbitProducer(url, exchange string) *RabbitProducer {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("rabbitmq unavailable, event publishing degraded: %v", err)
		return &RabbitProducer{exchange: exchange, enabled: false}
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		log.Printf("rabbitmq channel unavailable, event publishing degraded: %v", err)
		return &RabbitProducer{exchange: exchange, enabled: false}
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		log.Printf("rabbitmq exchange declare failed, event publishing degraded: %v", err)
		return &RabbitProducer{conn: conn, channel: ch, exchange: exchange, enabled: false}
	}
	return &RabbitProducer{conn: conn, channel: ch, exchange: exchange, enabled: true}
}

func (p *RabbitProducer) Publish(ctx context.Context, event Event) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if !p.enabled {
		log.Printf("mq degraded event_type=%s event_id=%s body=%s", event.EventType, event.EventID, string(body))
		return nil
	}
	return p.channel.PublishWithContext(ctx, p.exchange, event.EventType, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		Timestamp:   event.CreatedAt,
	})
}

func (p *RabbitProducer) Close() error {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
