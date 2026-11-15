package audit

import (
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"wms/backend/internal/httpx"
)

const (
	EntityUser     = "USER"
	EntityProduct  = "PRODUCT"
	EntityCategory = "CATEGORY"
	EntityUnit     = "UNIT"
)

const (
	ActionUserCreated        = "USER_CREATED"
	ActionUserActivated      = "USER_ACTIVATED"
	ActionUserDeactivated    = "USER_DEACTIVATED"
	ActionUserProfileUpdate  = "USER_PROFILE_UPDATED"
	ActionUserPassword       = "USER_PASSWORD_CHANGED"
	ActionProductCost        = "PRODUCT_COST_UPDATED"
	ActionProductActivated   = "PRODUCT_ACTIVATED"
	ActionProductDeactivated = "PRODUCT_DEACTIVATED"
	ActionProductUpdated     = "PRODUCT_UPDATED"
)

const (
	MinReasonLength = 5
	MaxReasonLength = 500
)

var validEntities = map[string]bool{
	EntityUser:     true,
	EntityProduct:  true,
	EntityCategory: true,
	EntityUnit:     true,
}

var validActions = map[string]bool{
	ActionUserCreated:        true,
	ActionUserActivated:      true,
	ActionUserDeactivated:    true,
	ActionUserProfileUpdate:  true,
	ActionUserPassword:       true,
	ActionProductCost:        true,
	ActionProductActivated:   true,
	ActionProductDeactivated: true,
	ActionProductUpdated:     true,
}

var statusActions = map[string]bool{
	ActionUserActivated:   true,
	ActionUserDeactivated: true,
}

type Actor struct {
	ID        uuid.UUID
	FullName  string
	Email     string
	Role      string
	RequestID string
}

type Entry struct {
	EntityType    string
	EntityID      uuid.UUID
	Action        string
	EntityLabel   string
	Actor         Actor
	Reason        string
	ChangedFields []string
	Before        map[string]any
	After         map[string]any
}

type ActorDTO struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
}

type DTO struct {
	ID             uuid.UUID      `json:"id"`
	EntityType     string         `json:"entity_type"`
	EntityID       uuid.UUID      `json:"entity_id"`
	EntityLabel    string         `json:"entity_label"`
	Action         string         `json:"action"`
	Actor          ActorDTO       `json:"actor"`
	Reason         *string        `json:"reason"`
	ChangedFields  []string       `json:"changed_fields"`
	Before         map[string]any `json:"before"`
	After          map[string]any `json:"after"`
	RedactedFields []string       `json:"redacted_fields"`
	RequestID      *string        `json:"request_id"`
	CreatedAt      time.Time      `json:"created_at"`
}

type ListFilter struct {
	EntityType  string
	EntityID    *uuid.UUID
	Actions     []string
	ActorID     *uuid.UUID
	ActorSearch string
	From        *time.Time
	To          *time.Time
	Page        int
	PageSize    int
}

func (f ListFilter) Offset() int {
	if f.Page < 1 {
		return 0
	}
	return (f.Page - 1) * f.PageSize
}

func ValidEntityType(value string) bool {
	return validEntities[value]
}

func ValidAction(value string) bool {
	return validActions[value]
}

func IsStatusAction(value string) bool {
	return statusActions[value]
}

func NormalizeReason(raw string, required bool) (string, *httpx.AppError) {
	reason := strings.TrimSpace(raw)
	if reason == "" {
		if required {
			return "", httpx.FieldError("reason", "is required")
		}
		return "", nil
	}
	length := utf8.RuneCountInString(reason)
	if length < MinReasonLength {
		return "", httpx.FieldError("reason", "must be at least 5 characters")
	}
	if length > MaxReasonLength {
		return "", httpx.FieldError("reason", "must be at most 500 characters")
	}
	return reason, nil
}

func ActorFromIdentity(identity *httpx.Identity, requestID string) Actor {
	if identity == nil {
		return Actor{RequestID: requestID}
	}
	return Actor{
		ID:        identity.ID,
		FullName:  identity.FullName,
		Email:     identity.Email,
		Role:      identity.Role,
		RequestID: requestID,
	}
}

func ChangedFields(before, after map[string]any) []string {
	keys := make(map[string]bool, len(before)+len(after))
	for key := range before {
		keys[key] = true
	}
	for key := range after {
		keys[key] = true
	}
	changed := make([]string, 0, len(keys))
	for key := range keys {
		if !reflect.DeepEqual(before[key], after[key]) {
			changed = append(changed, key)
		}
	}
	sort.Strings(changed)
	return changed
}

func Subset(values map[string]any, fields []string) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	out := make(map[string]any, len(fields))
	for _, field := range fields {
		out[field] = values[field]
	}
	return out
}

func Excluding(fields []string, excluded ...string) []string {
	skip := make(map[string]bool, len(excluded))
	for _, value := range excluded {
		skip[value] = true
	}
	out := make([]string, 0, len(fields))
	for _, value := range fields {
		if !skip[value] {
			out = append(out, value)
		}
	}
	return out
}

func CloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func RedactCosts(entries []DTO, includeCost bool) {
	if includeCost {
		return
	}
	for i := range entries {
		if entries[i].EntityType != EntityProduct {
			continue
		}
		if !contains(entries[i].ChangedFields, "unit_cost") {
			continue
		}
		if entries[i].Before != nil {
			delete(entries[i].Before, "unit_cost")
		}
		if entries[i].After != nil {
			delete(entries[i].After, "unit_cost")
		}
		if !contains(entries[i].RedactedFields, "unit_cost") {
			entries[i].RedactedFields = append(entries[i].RedactedFields, "unit_cost")
		}
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
