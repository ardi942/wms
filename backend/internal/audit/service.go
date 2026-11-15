package audit

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Record(ctx context.Context, tx *gorm.DB, entry Entry) error {
	if entry.Actor.ID == uuid.Nil {
		return errors.New("audit actor is required")
	}
	if !ValidEntityType(entry.EntityType) {
		return errors.New("invalid audit entity type")
	}
	if !ValidAction(entry.Action) {
		return errors.New("invalid audit action")
	}
	reason, err := NormalizeReason(entry.Reason, IsStatusAction(entry.Action))
	if err != nil {
		return err
	}
	entry.Reason = reason
	return s.repo.Insert(ctx, tx, entry)
}

func (s *Service) List(ctx context.Context, f ListFilter, includeCost bool) ([]DTO, int64, error) {
	entries, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	RedactCosts(entries, includeCost)
	return entries, total, nil
}

func (s *Service) HistoryFor(ctx context.Context, entityType string, entityID uuid.UUID, page, pageSize int, includeCost bool) ([]DTO, int64, error) {
	return s.List(ctx, ListFilter{
		EntityType: entityType,
		EntityID:   &entityID,
		Page:       page,
		PageSize:   pageSize,
	}, includeCost)
}
