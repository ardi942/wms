package idempotency

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/httpx"
)

type Store struct{ db *gorm.DB }

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

type Replay struct {
	StatusCode int
	Body       []byte
}

type row struct {
	RequestHash string    `gorm:"column:request_hash"`
	UserID      uuid.UUID `gorm:"column:user_id"`
	StatusCode  *int      `gorm:"column:status_code"`
	Body        *string   `gorm:"column:body"`
}

// Reserve inserts the key if absent. It returns a replay when the same key
// already completed successfully, and a conflict when it is in progress or
// was used with a different body.
func (s *Store) Reserve(ctx context.Context, operation, key string, userID uuid.UUID, hash string) (*Replay, error) {
	res := s.db.WithContext(ctx).Exec(
		`INSERT INTO idempotency_keys (operation, key, user_id, request_hash)
		 VALUES (?, ?, ?, ?) ON CONFLICT (operation, key) DO NOTHING`,
		operation, key, userID, hash,
	)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 1 {
		return nil, nil
	}
	var r row
	err := s.db.WithContext(ctx).Raw(
		`SELECT request_hash, user_id, status_code, response_body::text AS body
		 FROM idempotency_keys WHERE operation = ? AND key = ?`, operation, key,
	).Scan(&r).Error
	if err != nil {
		return nil, err
	}
	if r.UserID != userID {
		return nil, httpx.Conflict("IDEMPOTENCY_KEY_REUSED", "idempotency key was used by a different user", nil)
	}
	if r.RequestHash != hash {
		return nil, httpx.Conflict("IDEMPOTENCY_KEY_REUSED", "idempotency key was used with a different request body", nil)
	}
	if r.StatusCode == nil || r.Body == nil {
		return nil, httpx.Conflict("DUPLICATE_REQUEST", "a request with this idempotency key is already in progress", nil)
	}
	return &Replay{StatusCode: *r.StatusCode, Body: []byte(*r.Body)}, nil
}

func (s *Store) Complete(ctx context.Context, operation, key string, status int, body []byte) error {
	if !json.Valid(body) {
		body = []byte(`null`)
	}
	return s.db.WithContext(ctx).Exec(
		`UPDATE idempotency_keys SET status_code = ?, response_body = ?::jsonb
		 WHERE operation = ? AND key = ?`,
		status, string(body), operation, key,
	).Error
}

func (s *Store) Release(ctx context.Context, operation, key string) error {
	return s.db.WithContext(ctx).Exec(
		`DELETE FROM idempotency_keys WHERE operation = ? AND key = ? AND status_code IS NULL`,
		operation, key,
	).Error
}

func (s *Store) Cleanup(ctx context.Context) error {
	err := s.db.WithContext(ctx).Exec(
		`DELETE FROM idempotency_keys WHERE created_at < now() - interval '7 days'`,
	).Error
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Exec(
		`DELETE FROM idempotency_keys WHERE status_code IS NULL AND created_at < now() - interval '15 minutes'`,
	).Error
}

var ErrNoKey = errors.New("idempotency key is required")
