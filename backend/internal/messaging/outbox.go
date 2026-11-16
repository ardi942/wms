package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OutboxEvent struct {
	ID            uuid.UUID
	EventType     string
	AggregateType string
	AggregateID   uuid.UUID
	Payload       []byte
	Attempts      int
	CreatedAt     time.Time
}

type Publisher interface {
	Publish(ctx context.Context, env Envelope) error
}

type Relay struct {
	db       *gorm.DB
	pub      Publisher
	log      *slog.Logger
	interval time.Duration
	batch    int
}

func NewRelay(db *gorm.DB, pub Publisher, log *slog.Logger, interval time.Duration) *Relay {
	return &Relay{db: db, pub: pub, log: log, interval: interval, batch: 50}
}

func (r *Relay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for {
				n, err := r.publishBatch(ctx)
				if err != nil {
					r.log.Error("outbox publish failed", "error", err)
					break
				}
				if n == 0 {
					break
				}
			}
		}
	}
}

func (r *Relay) PublishBatch(ctx context.Context) (int, error) {
	return r.publishBatch(ctx)
}

func (r *Relay) publishBatch(ctx context.Context) (int, error) {
	published := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var events []OutboxEvent
		if err := tx.Raw(`
			SELECT id, event_type, aggregate_type, aggregate_id, payload, attempts, created_at
			FROM outbox_events
			WHERE status = 'PENDING' AND next_attempt_at <= now()
			ORDER BY created_at
			LIMIT ?
			FOR UPDATE SKIP LOCKED`, r.batch).Scan(&events).Error; err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		for i := range events {
			e := &events[i]
			env := Envelope{
				ID:            e.ID,
				Type:          e.EventType,
				AggregateType: e.AggregateType,
				AggregateID:   e.AggregateID,
				Payload:       json.RawMessage(e.Payload),
				CreatedAt:     e.CreatedAt.UTC().Format(time.RFC3339Nano),
			}
			if err := r.pub.Publish(ctx, env); err != nil {
				attempts := e.Attempts + 1
				backoff := time.Duration(1<<min(attempts, 6)) * time.Second
				status := "PENDING"
				if attempts >= 10 {
					status = "FAILED"
				}
				if uerr := tx.Exec(`
					UPDATE outbox_events
					SET attempts = ?, last_error = ?, status = ?, next_attempt_at = ?
					WHERE id = ?`,
					attempts, truncate(err.Error(), 1000), status, time.Now().Add(backoff), e.ID).Error; uerr != nil {
					return uerr
				}
				r.log.Warn("outbox event publish failed", "event_id", e.ID, "attempts", attempts, "error", err)
				continue
			}
			if err := tx.Exec(`
				UPDATE outbox_events SET status = 'PUBLISHED', published_at = now(), attempts = attempts + 1
				WHERE id = ?`, e.ID).Error; err != nil {
				return err
			}
			published++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("publish outbox batch: %w", err)
	}
	return published, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
