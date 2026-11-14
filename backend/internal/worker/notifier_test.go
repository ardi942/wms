package worker

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/messaging"
)

func TestBuildMessageForFulfilledRequest(t *testing.T) {
	requester := uuid.Must(uuid.NewV7())
	aggregate := uuid.Must(uuid.NewV7())
	n := NewNotifier(nil, slog.Default())
	msg, ok, err := n.buildMessage(context.Background(), messaging.Envelope{
		ID: uuid.Must(uuid.NewV7()), Type: messaging.EventRequestFulfilled, AggregateID: aggregate,
		Payload: []byte(`{"request_number":"REQ-1","requester_id":"` + requester.String() + `","partial":true}`),
	})
	if err != nil || !ok {
		t.Fatalf("unexpected result ok=%v err=%v", ok, err)
	}
	if msg.title != "Stock request partially fulfilled" {
		t.Fatalf("unexpected title %q", msg.title)
	}
	if len(msg.recipients) != 1 || msg.recipients[0] != requester {
		t.Fatalf("unexpected recipients %v", msg.recipients)
	}
}
