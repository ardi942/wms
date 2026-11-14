package unit

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/httpx"
)

type Unit struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code      string    `gorm:"column:code"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Unit) TableName() string { return "units" }

type DTO struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context) ([]Unit, error) {
	var rows []Unit
	err := r.db.WithContext(ctx).Order("code ASC").Find(&rows).Error
	return rows, err
}

func (r *Repository) ExistsByCode(ctx context.Context, code string, except *uuid.UUID) (bool, error) {
	q := r.db.WithContext(ctx).Model(&Unit{}).Where("code = ?", code)
	if except != nil {
		q = q.Where("id <> ?", *except)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

func (r *Repository) Create(ctx context.Context, row *Unit) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	fields["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).Model(&Unit{}).Where("id = ?", id).Updates(fields).Error
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

type createRequest struct {
	Code string `json:"code" binding:"required,min=1,max=16"`
	Name string `json:"name" binding:"required,min=1,max=100"`
}

type updateRequest struct {
	Code *string `json:"code" binding:"omitempty,min=1,max=16"`
	Name *string `json:"name" binding:"omitempty,min=1,max=100"`
}

func (h *Handler) List(c *gin.Context) {
	rows, err := h.repo.List(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	out := make([]DTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, DTO{ID: r.ID, Code: r.Code, Name: r.Name})
	}
	httpx.OK(c, out)
}

func (h *Handler) Create(c *gin.Context) {
	var req createRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	exists, err := h.repo.ExistsByCode(c.Request.Context(), code, nil)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	if exists {
		httpx.WriteError(c, httpx.Conflict("DUPLICATE", "unit code already exists", nil))
		return
	}
	now := time.Now().UTC()
	row := &Unit{ID: uuid.Must(uuid.NewV7()), Code: code, Name: strings.TrimSpace(req.Name), CreatedAt: now, UpdatedAt: now}
	if err := h.repo.Create(c.Request.Context(), row); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, DTO{ID: row.ID, Code: row.Code, Name: row.Name})
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("unit not found"))
		return
	}
	var req updateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	fields := map[string]any{}
	if req.Code != nil {
		code := strings.ToUpper(strings.TrimSpace(*req.Code))
		exists, err := h.repo.ExistsByCode(c.Request.Context(), code, &id)
		if err != nil {
			httpx.WriteError(c, err)
			return
		}
		if exists {
			httpx.WriteError(c, httpx.Conflict("DUPLICATE", "unit code already exists", nil))
			return
		}
		fields["code"] = code
	}
	if req.Name != nil {
		fields["name"] = strings.TrimSpace(*req.Name)
	}
	if len(fields) > 0 {
		if err := h.repo.Update(c.Request.Context(), id, fields); err != nil {
			httpx.WriteError(c, err)
			return
		}
	}
	httpx.OK(c, gin.H{"id": id})
}
