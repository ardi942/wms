package messaging

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	EventStockReceived    = "stock.received"
	EventStockAdjusted    = "stock.adjusted"
	EventStockIssued      = "stock.issued"
	EventRequestCreated   = "request.created"
	EventRequestFulfilled = "request.fulfilled"
	EventRequestRejected  = "request.rejected"
	EventRequestCanceled  = "request.cancelled"
	EventReturnCreated    = "return.created"
	EventReturnConfirmed  = "return.confirmed"
	EventReturnRejected   = "return.rejected"
	EventReturnCanceled   = "return.cancelled"
)

type Envelope struct {
	ID            uuid.UUID       `json:"id"`
	Type          string          `json:"type"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   uuid.UUID       `json:"aggregate_id"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     string          `json:"created_at"`
}

func Enqueue(tx *gorm.DB, eventType, aggregateType string, aggregateID uuid.UUID, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	id := uuid.Must(uuid.NewV7())
	return tx.Exec(
		`INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload)
		 VALUES (?, ?, ?, ?, ?::jsonb)`,
		id, eventType, aggregateType, aggregateID, string(body),
	).Error
}
