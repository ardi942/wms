package outbox_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"wms/backend/internal/messaging"
	"wms/backend/internal/testkit"
)

type fakePublisher struct {
	calls   atomic.Int32
	failFor int32
}

func (f *fakePublisher) Publish(_ context.Context, _ messaging.Envelope) error {
	n := f.calls.Add(1)
	if f.failFor > 0 && n <= f.failFor {
		return errors.New("broker down")
	}
	return nil
}

func TestMain(m *testing.M) {
	code := m.Run()
	testkit.Cleanup()
	os.Exit(code)
}

func TestRelayDoesNotDoublePublish(t *testing.T) {
	env := testkit.New(t)
	pub := &fakePublisher{}
	relay := messaging.NewRelay(env.DB, pub, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Millisecond)

	id := uuid.Must(uuid.NewV7())
	if err := env.DB.Exec(`INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, attempts, next_attempt_at)
		VALUES (?, 'test.event', 'test', ?, '{"ok":true}'::jsonb, 'PENDING', 0, now())`, id, uuid.Must(uuid.NewV7())).Error; err != nil {
		t.Fatalf("insert outbox: %v", err)
	}
	n, err := relay.PublishBatch(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("first publish n=%d err=%v", n, err)
	}
	n, err = relay.PublishBatch(context.Background())
	if err != nil || n != 0 {
		t.Fatalf("second publish n=%d err=%v", n, err)
	}
	if got := pub.calls.Load(); got != 1 {
		t.Fatalf("expected 1 publish call, got %d", got)
	}
	var status string
	if err := env.DB.Raw(`SELECT status FROM outbox_events WHERE id = ?`, id).Scan(&status).Error; err != nil {
		t.Fatalf("load status: %v", err)
	}
	if status != "PUBLISHED" {
		t.Fatalf("expected PUBLISHED got %s", status)
	}
}

func TestRelayRetryAndFailure(t *testing.T) {
	env := testkit.New(t)
	pub := &fakePublisher{failFor: 1}
	relay := messaging.NewRelay(env.DB, pub, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Millisecond)
	id := uuid.Must(uuid.NewV7())
	if err := env.DB.Exec(`INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, attempts, next_attempt_at)
		VALUES (?, 'test.event', 'test', ?, '{"ok":true}'::jsonb, 'PENDING', 0, now())`, id, uuid.Must(uuid.NewV7())).Error; err != nil {
		t.Fatalf("insert outbox: %v", err)
	}
	n, err := relay.PublishBatch(context.Background())
	if err != nil {
		t.Fatalf("publish batch err: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 published on failure, got %d", n)
	}
	var attempts int
	var status string
	if err := env.DB.Raw(`SELECT attempts, status FROM outbox_events WHERE id = ?`, id).Row().Scan(&attempts, &status); err != nil {
		t.Fatalf("load row: %v", err)
	}
	if attempts != 1 || status != "PENDING" {
		t.Fatalf("expected attempts=1 status=PENDING got attempts=%d status=%s", attempts, status)
	}
	if err := env.DB.Exec(`UPDATE outbox_events SET next_attempt_at = now() WHERE id = ?`, id).Error; err != nil {
		t.Fatalf("reset next_attempt: %v", err)
	}
	n, err = relay.PublishBatch(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("retry publish n=%d err=%v", n, err)
	}
}

func TestRelayMarksFailedAfterMaxAttempts(t *testing.T) {
	env := testkit.New(t)
	pub := &fakePublisher{failFor: 100}
	relay := messaging.NewRelay(env.DB, pub, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Millisecond)
	id := uuid.Must(uuid.NewV7())
	if err := env.DB.Exec(`INSERT INTO outbox_events (id, event_type, aggregate_type, aggregate_id, payload, status, attempts, next_attempt_at)
		VALUES (?, 'test.event', 'test', ?, '{"ok":true}'::jsonb, 'PENDING', 9, now())`, id, uuid.Must(uuid.NewV7())).Error; err != nil {
		t.Fatalf("insert outbox: %v", err)
	}
	n, err := relay.PublishBatch(context.Background())
	if err != nil {
		t.Fatalf("publish batch err: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 published on failure, got %d", n)
	}
	var attempts int
	var status string
	if err := env.DB.Raw(`SELECT attempts, status FROM outbox_events WHERE id = ?`, id).Row().Scan(&attempts, &status); err != nil {
		t.Fatalf("load row: %v", err)
	}
	if attempts != 10 || status != "FAILED" {
		t.Fatalf("expected attempts=10 status=FAILED got attempts=%d status=%s", attempts, status)
	}
}
