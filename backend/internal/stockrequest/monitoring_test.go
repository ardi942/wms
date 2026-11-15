package stockrequest

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToMonitoringRequestRow(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	deptID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	created := time.Date(2026, 11, 15, 8, 0, 0, 0, time.UTC)
	updated := created.Add(15 * time.Minute)

	row := toMonitoringRequestRow(ListDTO{
		ID:             id,
		RequestNumber:  "REQ-2026-0001",
		Status:         StatusPartiallyFulfilled,
		Department:     DepartmentRef{ID: deptID, Code: "IT", Name: "Information Technology"},
		RequestedBy:    ActorRef{ID: userID, FullName: "Ayu Lestari"},
		Purpose:        "secret purpose",
		ItemCount:      3,
		TotalRequested: 10,
		TotalIssued:    4,
		AllowedActions: []string{ActionFulfill, ActionReject},
		CreatedAt:      created,
		UpdatedAt:      updated,
	})

	if row.ID != id || row.RequestNumber != "REQ-2026-0001" || row.Status != StatusPartiallyFulfilled {
		t.Fatalf("header not preserved: %+v", row)
	}
	if row.Department.ID != deptID || row.Department.Code != "IT" || row.Department.Name != "Information Technology" {
		t.Fatalf("department not preserved: %+v", row.Department)
	}
	if row.RequestedBy.ID != userID || row.RequestedBy.FullName != "Ayu Lestari" {
		t.Fatalf("requester not preserved: %+v", row.RequestedBy)
	}
	if row.ItemCount != 3 || row.TotalRequested != 10 || row.TotalIssued != 4 {
		t.Fatalf("totals not preserved: %+v", row)
	}
	if !row.CreatedAt.Equal(created) || !row.UpdatedAt.Equal(updated) {
		t.Fatalf("timestamps not preserved: %+v", row)
	}

	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, forbidden := range []string{"allowed_actions", "purpose", "rejection_reason", "items", "processed_by", "reason"} {
		if _, ok := fields[forbidden]; ok {
			t.Fatalf("monitoring row must not expose %q: %s", forbidden, raw)
		}
	}
	if len(fields) != 10 {
		t.Fatalf("unexpected monitoring fields: %s", raw)
	}
}

func TestToMonitoringRequestRowEmptyValues(t *testing.T) {
	row := toMonitoringRequestRow(ListDTO{})
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if fields["request_number"] != "" || fields["status"] != "" {
		t.Fatalf("expected safe empty strings: %s", raw)
	}
	if _, ok := fields["allowed_actions"]; ok {
		t.Fatalf("monitoring row must not expose allowed_actions: %s", raw)
	}
}
