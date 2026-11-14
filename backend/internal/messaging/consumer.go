package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Handler func(context.Context, Envelope) error

type Consumer struct {
	conn        *amqp.Connection
	queue       string
	handler     Handler
	concurrency int
	log         *slog.Logger
}

func NewConsumer(conn *amqp.Connection, queue string, handler Handler, concurrency int, log *slog.Logger) (*Consumer, error) {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Consumer{conn: conn, queue: queue, handler: handler, concurrency: concurrency, log: log}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("consumer channel: %w", err)
	}
	defer ch.Close()
	if err := ch.ExchangeDeclare(ExchangeName, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(DeadLetterName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead letter queue: %w", err)
	}
	if _, err := ch.QueueDeclare(c.queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := ch.QueueBind(c.queue, "wms.#", ExchangeName, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	if err := ch.Qos(c.concurrency*2, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}
	deliveries, err := ch.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	jobs := make(chan amqp.Delivery, c.concurrency*2)
	var wg sync.WaitGroup
	for i := 0; i < c.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range jobs {
				c.process(ctx, ch, d)
			}
		}()
	}

	c.log.Info("consumer started", "queue", c.queue, "concurrency", c.concurrency)
	for {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil
		case d, ok := <-deliveries:
			if !ok {
				close(jobs)
				wg.Wait()
				return fmt.Errorf("delivery channel closed")
			}
			select {
			case jobs <- d:
			case <-ctx.Done():
				close(jobs)
				wg.Wait()
				return nil
			}
		}
	}
}

func (c *Consumer) process(ctx context.Context, ch *amqp.Channel, d amqp.Delivery) {
	var env Envelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		c.log.Error("invalid event payload", "error", err)
		_ = d.Nack(false, false)
		return
	}
	if err := c.handler(ctx, env); err != nil {
		retries := int64(0)
		if v, ok := d.Headers["x-retry"]; ok {
			if n, ok := v.(int64); ok {
				retries = n
			}
		}
		if retries >= MaxRetries {
			c.log.Error("event exhausted retries; dead-lettering", "event_id", env.ID, "type", env.Type, "error", err)
			_ = ch.PublishWithContext(ctx, "", DeadLetterName, false, false, amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				MessageId:    env.ID.String(),
				Type:         env.Type,
				Body:         d.Body,
				Timestamp:    time.Now().UTC(),
			})
			_ = d.Ack(false)
			return
		}
		if perr := ch.PublishWithContext(ctx, "", c.queue, false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    env.ID.String(),
			Type:         env.Type,
			Body:         d.Body,
			Timestamp:    time.Now().UTC(),
			Headers:      amqp.Table{"x-retry": retries + 1},
		}); perr != nil {
			c.log.Error("retry republish failed", "error", perr)
			_ = d.Nack(false, true)
			return
		}
		c.log.Warn("event handler failed; retrying", "event_id", env.ID, "type", env.Type, "retry", retries+1, "error", err)
		_ = d.Ack(false)
		return
	}
	_ = d.Ack(false)
}
