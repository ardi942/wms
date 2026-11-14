package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/messaging"
)

type Notifier struct {
	db  *gorm.DB
	log *slog.Logger
}

func NewNotifier(db *gorm.DB, log *slog.Logger) *Notifier {
	return &Notifier{db: db, log: log}
}

type message struct {
	title       string
	body        string
	relatedType string
	relatedID   uuid.UUID
	recipients  []uuid.UUID
}

func (n *Notifier) Handle(ctx context.Context, env messaging.Envelope) error {
	msg, ok, err := n.buildMessage(ctx, env)
	if err != nil {
		return err
	}
	return n.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`INSERT INTO processed_events (consumer, event_id) VALUES (?, ?) ON CONFLICT DO NOTHING`,
			"notifications", env.ID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		if !ok || len(msg.recipients) == 0 {
			return nil
		}
		relatedType := msg.relatedType
		relatedID := msg.relatedID
		for _, userID := range dedupe(msg.recipients) {
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if err := tx.Exec(`
				INSERT INTO notifications (id, user_id, type, title, body, related_type, related_id)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				id, userID, env.Type, msg.title, msg.body, relatedType, relatedID).Error; err != nil {
				return err
			}
		}
		n.log.Info("notifications created", "event_id", env.ID, "type", env.Type, "recipients", len(msg.recipients))
		return nil
	})
}

func (n *Notifier) buildMessage(ctx context.Context, env messaging.Envelope) (*message, bool, error) {
	var p map[string]any
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return nil, false, fmt.Errorf("decode payload: %w", err)
	}
	switch env.Type {
	case messaging.EventRequestCreated:
		return &message{
			title:       "New stock request",
			body:        fmt.Sprintf("%s submitted a request with %s item(s)", text(p, "request_number"), text(p, "item_count")),
			relatedType: "stock_request",
			relatedID:   env.AggregateID,
			recipients:  n.admins(ctx),
		}, true, nil
	case messaging.EventRequestFulfilled:
		label := "fulfilled"
		if b, _ := p["partial"].(bool); b {
			label = "partially fulfilled"
		}
		return &message{
			title:       "Stock request " + label,
			body:        fmt.Sprintf("%s has been %s", text(p, "request_number"), label),
			relatedType: "stock_request",
			relatedID:   env.AggregateID,
			recipients:  idList(p, "requester_id"),
		}, true, nil
	case messaging.EventRequestRejected:
		return &message{
			title:       "Stock request rejected",
			body:        fmt.Sprintf("%s was rejected: %s", text(p, "request_number"), text(p, "reason")),
			relatedType: "stock_request",
			relatedID:   env.AggregateID,
			recipients:  idList(p, "requester_id"),
		}, true, nil
	case messaging.EventReturnCreated:
		return &message{
			title:       "New stock return",
			body:        fmt.Sprintf("%s submitted a return with %s item(s)", text(p, "return_number"), text(p, "item_count")),
			relatedType: "stock_return",
			relatedID:   env.AggregateID,
			recipients:  n.admins(ctx),
		}, true, nil
	case messaging.EventReturnConfirmed:
		return &message{
			title:       "Stock return confirmed",
			body:        fmt.Sprintf("%s has been confirmed", text(p, "return_number")),
			relatedType: "stock_return",
			relatedID:   env.AggregateID,
			recipients:  idList(p, "requester_id"),
		}, true, nil
	case messaging.EventReturnRejected:
		return &message{
			title:       "Stock return rejected",
			body:        fmt.Sprintf("%s was rejected: %s", text(p, "return_number"), text(p, "reason")),
			relatedType: "stock_return",
			relatedID:   env.AggregateID,
			recipients:  idList(p, "requester_id"),
		}, true, nil
	default:
		return nil, false, nil
	}
}

func (n *Notifier) admins(ctx context.Context) []uuid.UUID {
	var ids []uuid.UUID
	if err := n.db.WithContext(ctx).Raw(
		`SELECT id FROM users WHERE role = 'WAREHOUSE_ADMIN' AND is_active = true`).Scan(&ids).Error; err != nil {
		n.log.Error("failed to load notification recipients", "error", err)
	}
	return ids
}

func text(p map[string]any, key string) string {
	if v, ok := p[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func idList(p map[string]any, key string) []uuid.UUID {
	s := text(p, key)
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return []uuid.UUID{id}
}

func dedupe(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
