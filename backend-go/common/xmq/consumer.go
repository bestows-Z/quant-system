package xmq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Handler func(context.Context, Event) error

type Consumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
	queue    string
	enabled  bool
}

func NewRabbitConsumer(url, exchange, queue string, routingKeys []string) *Consumer {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Printf("rabbitmq consumer unavailable queue=%s: %v", queue, err)
		return &Consumer{exchange: exchange, queue: queue, enabled: false}
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		log.Printf("rabbitmq consumer channel unavailable queue=%s: %v", queue, err)
		return &Consumer{exchange: exchange, queue: queue, enabled: false}
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		log.Printf("rabbitmq consumer exchange declare failed queue=%s: %v", queue, err)
		return &Consumer{conn: conn, channel: ch, exchange: exchange, queue: queue, enabled: false}
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		log.Printf("rabbitmq queue declare failed queue=%s: %v", queue, err)
		return &Consumer{conn: conn, channel: ch, exchange: exchange, queue: queue, enabled: false}
	}
	if len(routingKeys) == 0 {
		routingKeys = []string{"#"}
	}
	for _, key := range routingKeys {
		if err := ch.QueueBind(queue, key, exchange, false, nil); err != nil {
			log.Printf("rabbitmq queue bind failed queue=%s key=%s: %v", queue, key, err)
		}
	}
	return &Consumer{conn: conn, channel: ch, exchange: exchange, queue: queue, enabled: true}
}

func (c *Consumer) Start(ctx context.Context, handler Handler) {
	if !c.enabled {
		log.Printf("mq consumer degraded queue=%s", c.queue)
		return
	}
	msgs, err := c.channel.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		log.Printf("rabbitmq consume failed queue=%s: %v", c.queue, err)
		return
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
				var event Event
				if err := json.Unmarshal(msg.Body, &event); err != nil {
					log.Printf("invalid mq event queue=%s: %v body=%s", c.queue, err, string(msg.Body))
					_ = msg.Ack(false)
					continue
				}
				if event.CreatedAt.IsZero() {
					event.CreatedAt = time.Now()
				}
				if err := handler(ctx, event); err != nil {
					log.Printf("mq handler failed queue=%s event_type=%s event_id=%s: %v", c.queue, event.EventType, event.EventID, err)
				}
				_ = msg.Ack(false)
			}
		}
	}()
}

func (c *Consumer) Close() error {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
