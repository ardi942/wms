package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type dbRow struct {
	ID            uuid.UUID      `gorm:"column:id"`
	EntityType    string         `gorm:"column:entity_type"`
	EntityID      uuid.UUID      `gorm:"column:entity_id"`
	EntityLabel   string         `gorm:"column:entity_label"`
	Action        string         `gorm:"column:action"`
	ActorID       uuid.UUID      `gorm:"column:actor_id"`
	ActorName     string         `gorm:"column:actor_name"`
	ActorEmail    string         `gorm:"column:actor_email"`
	ActorRole     string         `gorm:"column:actor_role"`
	Reason        *string        `gorm:"column:reason"`
	ChangedFields pq.StringArray `gorm:"column:changed_fields"`
	BeforeData    []byte         `gorm:"column:before_data"`
	AfterData     []byte         `gorm:"column:after_data"`
	RequestID     *string        `gorm:"column:request_id"`
	CreatedAt     time.Time      `gorm:"column:created_at"`
}

func (r *Repository) Insert(ctx context.Context, tx *gorm.DB, entry Entry) error {
	before, err := encodeMap(entry.Before)
	if err != nil {
		return err
	}
	after, err := encodeMap(entry.After)
	if err != nil {
		return err
	}
	changed := entry.ChangedFields
	if changed == nil {
		changed = []string{}
	}
	var reason any
	if entry.Reason != "" {
		reason = entry.Reason
	}
	var requestID any
	if entry.Actor.RequestID != "" {
		requestID = entry.Actor.RequestID
	}
	return tx.WithContext(ctx).Exec(`
		INSERT INTO audit_logs (
			id, entity_type, entity_id, action,
			actor_id, actor_name, actor_email, actor_role,
			reason, changed_fields, before_data, after_data, request_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?::jsonb, ?, ?)`,
		uuid.Must(uuid.NewV7()), entry.EntityType, entry.EntityID, entry.Action,
		entry.Actor.ID, entry.Actor.FullName, entry.Actor.Email, entry.Actor.Role,
		reason, pq.Array(changed), before, after, requestID, time.Now().UTC(),
	).Error
}

func (r *Repository) List(ctx context.Context, f ListFilter) ([]DTO, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	q := r.db.WithContext(ctx).
		Table("audit_logs AS a").
		Joins("LEFT JOIN users u ON a.entity_type = 'USER' AND u.id = a.entity_id").
		Joins("LEFT JOIN products p ON a.entity_type = 'PRODUCT' AND p.id = a.entity_id")
	if f.EntityType != "" {
		q = q.Where("a.entity_type = ?", f.EntityType)
	}
	if f.EntityID != nil {
		q = q.Where("a.entity_id = ?", *f.EntityID)
	}
	if len(f.Actions) > 0 {
		q = q.Where("a.action IN ?", f.Actions)
	}
	if f.ActorID != nil {
		q = q.Where("a.actor_id = ?", *f.ActorID)
	}
	if f.ActorSearch != "" {
		search := "%" + f.ActorSearch + "%"
		q = q.Where("(a.actor_name ILIKE ? OR a.actor_email ILIKE ?)", search, search)
	}
	if f.From != nil {
		q = q.Where("a.created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("a.created_at < ?", *f.To)
	}

	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []dbRow
	err := q.Session(&gorm.Session{}).Select(`
		a.id, a.entity_type, a.entity_id, a.action,
		a.actor_id, a.actor_name, a.actor_email, a.actor_role,
		a.reason, a.changed_fields, a.before_data, a.after_data,
		a.request_id, a.created_at,
		CASE
			WHEN a.entity_type = 'USER' THEN COALESCE(u.full_name, '')
			WHEN a.entity_type = 'PRODUCT' THEN COALESCE(p.sku || ' — ' || p.name, '')
			ELSE ''
		END AS entity_label`).
		Order("a.created_at DESC, a.id DESC").
		Offset(f.Offset()).
		Limit(f.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	out := make([]DTO, 0, len(rows))
	for i := range rows {
		out = append(out, toDTO(&rows[i]))
	}
	return out, total, nil
}

func toDTO(row *dbRow) DTO {
	changed := []string(row.ChangedFields)
	if changed == nil {
		changed = []string{}
	}
	return DTO{
		ID:          row.ID,
		EntityType:  row.EntityType,
		EntityID:    row.EntityID,
		EntityLabel: row.EntityLabel,
		Action:      row.Action,
		Actor: ActorDTO{
			ID:       row.ActorID,
			FullName: row.ActorName,
			Email:    row.ActorEmail,
			Role:     row.ActorRole,
		},
		Reason:         row.Reason,
		ChangedFields:  changed,
		Before:         decodeMap(row.BeforeData),
		After:          decodeMap(row.AfterData),
		RedactedFields: []string{},
		RequestID:      row.RequestID,
		CreatedAt:      row.CreatedAt,
	}
}

func encodeMap(values map[string]any) (any, error) {
	if values == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

func decodeMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return values
}
