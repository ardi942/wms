package audit

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/httpx"
)

func TestNormalizeReason(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		required bool
		want     string
		wantErr  bool
	}{
		{name: "required empty", required: true, wantErr: true},
		{name: "required whitespace", raw: "   ", required: true, wantErr: true},
		{name: "required too short", raw: "abcd", required: true, wantErr: true},
		{name: "required minimum", raw: "abcde", required: true, want: "abcde"},
		{name: "trims surrounding spaces", raw: "  valid reason  ", required: true, want: "valid reason"},
		{name: "maximum", raw: strings.Repeat("a", 500), required: true, want: strings.Repeat("a", 500)},
		{name: "too long", raw: strings.Repeat("a", 501), required: true, wantErr: true},
		{name: "optional empty", required: false, want: ""},
		{name: "optional too short", raw: "bad", required: false, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeReason(tt.raw, tt.required)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				var appErr *httpx.AppError
				if !asAppError(err, &appErr) {
					t.Fatalf("expected AppError, got %T", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func asAppError(err error, target **httpx.AppError) bool {
	appErr, ok := err.(*httpx.AppError)
	if ok {
		*target = appErr
	}
	return ok
}

func TestChangedFields(t *testing.T) {
	before := map[string]any{"name": "Old", "is_active": true, "unit_cost": 10.0}
	after := map[string]any{"name": "New", "is_active": true, "unit_cost": nil}
	got := ChangedFields(before, after)
	want := []string{"name", "unit_cost"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("changed fields = %v, want %v", got, want)
	}
	subset := Subset(after, got)
	if len(subset) != 2 || subset["name"] != "New" {
		t.Fatalf("unexpected subset: %#v", subset)
	}
}

func TestRedactCosts(t *testing.T) {
	entries := []DTO{
		{
			EntityType:    EntityProduct,
			ChangedFields: []string{"unit_cost", "name"},
			Before:        map[string]any{"unit_cost": 10, "name": "Old"},
			After:         map[string]any{"unit_cost": 20, "name": "New"},
		},
	}
	RedactCosts(entries, false)
	if _, ok := entries[0].Before["unit_cost"]; ok {
		t.Fatal("before cost was not redacted")
	}
	if _, ok := entries[0].After["unit_cost"]; ok {
		t.Fatal("after cost was not redacted")
	}
	if !contains(entries[0].RedactedFields, "unit_cost") {
		t.Fatal("redacted fields was not recorded")
	}

	visible := []DTO{{
		EntityType:    EntityProduct,
		ChangedFields: []string{"unit_cost"},
		Before:        map[string]any{"unit_cost": 10},
		After:         map[string]any{"unit_cost": 20},
	}}
	RedactCosts(visible, true)
	if visible[0].Before["unit_cost"] != 10 || visible[0].After["unit_cost"] != 20 {
		t.Fatal("supervisor cost was redacted")
	}
	if len(visible[0].RedactedFields) != 0 {
		t.Fatal("supervisor redaction notice was added")
	}
}

func TestStatusActionClassification(t *testing.T) {
	for _, action := range []string{ActionUserActivated, ActionUserDeactivated} {
		if !IsStatusAction(action) {
			t.Errorf("%s should be a status action", action)
		}
	}
	for _, action := range []string{ActionUserCreated, ActionUserProfileUpdate, ActionProductCost, ActionProductUpdated} {
		if IsStatusAction(action) {
			t.Errorf("%s should not be a status action", action)
		}
	}
}

func TestActorFromIdentity(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	actor := ActorFromIdentity(&httpx.Identity{
		ID:       id,
		FullName: "Ayu Lestari",
		Email:    "ayu@example.com",
		Role:     "WAREHOUSE_ADMIN",
	}, "req-123")
	if actor.ID != id || actor.FullName != "Ayu Lestari" || actor.Email != "ayu@example.com" || actor.Role != "WAREHOUSE_ADMIN" || actor.RequestID != "req-123" {
		t.Fatalf("unexpected actor: %#v", actor)
	}
}
