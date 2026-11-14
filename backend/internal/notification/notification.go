package notification

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/httpx"
)

type Notification struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID  `gorm:"column:user_id;type:uuid"`
	Type        string     `gorm:"column:type"`
	Title       string     `gorm:"column:title"`
	Body        string     `gorm:"column:body"`
	RelatedType *string    `gorm:"column:related_type"`
	RelatedID   *uuid.UUID `gorm:"column:related_id;type:uuid"`
	IsRead      bool       `gorm:"column:is_read"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

func (Notification) TableName() string { return "notifications" }

type DTO struct {
	ID          uuid.UUID  `json:"id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	RelatedType *string    `json:"related_type"`
	RelatedID   *uuid.UUID `json:"related_id"`
	IsRead      bool       `json:"is_read"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, page, pageSize int) ([]Notification, int64, error) {
	q := r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ?", userID)
	if unreadOnly {
		q = q.Where("is_read = false")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Notification
	err := q.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	return rows, total, err
}

func (r *Repository) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND is_read = false", userID).Count(&n).Error
	return n, err
}

func (r *Repository) MarkRead(ctx context.Context, userID, id uuid.UUID) (bool, error) {
	res := r.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ? AND user_id = ?", id, userID).Update("is_read", true)
	return res.RowsAffected > 0, res.Error
}

func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND is_read = false", userID).Update("is_read", true).Error
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func toDTO(n *Notification) DTO {
	return DTO{
		ID: n.ID, Type: n.Type, Title: n.Title, Body: n.Body,
		RelatedType: n.RelatedType, RelatedID: n.RelatedID,
		IsRead: n.IsRead, CreatedAt: n.CreatedAt,
	}
}

func (h *Handler) List(c *gin.Context) {
	p, perr := httpx.ParsePagination(c)
	if perr != nil {
		httpx.WriteError(c, perr)
		return
	}
	id := httpx.MustIdentity(c)
	rows, total, err := h.repo.List(c.Request.Context(), id.ID, c.Query("unread_only") == "true", p.Page, p.PageSize)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	out := make([]DTO, 0, len(rows))
	for i := range rows {
		out = append(out, toDTO(&rows[i]))
	}
	meta := httpx.NewMeta(p, total)
	httpx.List(c, out, meta)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	id := httpx.MustIdentity(c)
	n, err := h.repo.UnreadCount(c.Request.Context(), id.ID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, gin.H{"unread_count": n})
}

func (h *Handler) MarkRead(c *gin.Context) {
	notifID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("notification not found"))
		return
	}
	id := httpx.MustIdentity(c)
	ok, uerr := h.repo.MarkRead(c.Request.Context(), id.ID, notifID)
	if uerr != nil {
		httpx.WriteError(c, uerr)
		return
	}
	if !ok {
		httpx.WriteError(c, httpx.NotFound("notification not found"))
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	id := httpx.MustIdentity(c)
	if err := h.repo.MarkAllRead(c.Request.Context(), id.ID); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.NoContent(c)
}
