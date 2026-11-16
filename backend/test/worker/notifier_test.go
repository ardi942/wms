package worker_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/messaging"
	"wms/backend/internal/testkit"
	"wms/backend/internal/worker"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testkit.Cleanup()
	os.Exit(code)
}

func TestNotifierIdempotent(t *testing.T) {
	env := testkit.New(t)
	n := worker.NewNotifier(env.DB, slog.New(slog.NewTextHandler(io.Discard, nil)))
	eventID := uuid.Must(uuid.NewV7())
	payload, _ := json.Marshal(map[string]any{"request_number": "REQ-1", "item_count": 2})
	msg := messaging.Envelope{
		ID:          eventID,
		Type:        messaging.EventRequestCreated,
		AggregateID: uuid.Must(uuid.NewV7()),
		Payload:     payload,
	}
	ctx := context.Background()
	if err := n.Handle(ctx, msg); err != nil {
		t.Fatalf("first handle: %v", err)
	}
	if err := n.Handle(ctx, msg); err != nil {
		t.Fatalf("second handle: %v", err)
	}
	var processed int64
	if err := env.DB.Raw(`SELECT COUNT(*) FROM processed_events WHERE consumer='notifications' AND event_id = ?`, eventID).Scan(&processed).Error; err != nil {
		t.Fatalf("count processed: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed event, got %d", processed)
	}
	var notifs int64
	if err := env.DB.Raw(`SELECT COUNT(*) FROM notifications WHERE type = ?`, messaging.EventRequestCreated).Scan(&notifs).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if notifs == 0 {
		t.Fatalf("expected notifications to be created")
	}
}
