package category

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/httpx"
)

type Category struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (Category) TableName() string { return "categories" }

type DTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context) ([]Category, error) {
	var rows []Category
	err := r.db.WithContext(ctx).Order("name ASC").Find(&rows).Error
	return rows, err
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Category, error) {
	var row Category
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) ExistsByName(ctx context.Context, name string, except *uuid.UUID) (bool, error) {
	q := r.db.WithContext(ctx).Model(&Category{}).Where("lower(name) = lower(?)", name)
	if except != nil {
		q = q.Where("id <> ?", *except)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

func (r *Repository) Create(ctx context.Context, row *Category) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, name string) error {
	return r.db.WithContext(ctx).Model(&Category{}).Where("id = ?", id).
		Updates(map[string]any{"name": name, "updated_at": time.Now().UTC()}).Error
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

type nameRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

func (h *Handler) List(c *gin.Context) {
	rows, err := h.repo.List(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	out := make([]DTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, DTO{ID: r.ID, Name: r.Name})
	}
	httpx.OK(c, out)
}

func (h *Handler) Create(c *gin.Context) {
	var req nameRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	exists, err := h.repo.ExistsByName(c.Request.Context(), name, nil)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	if exists {
		httpx.WriteError(c, httpx.Conflict("DUPLICATE", "category already exists", nil))
		return
	}
	now := time.Now().UTC()
	row := &Category{ID: uuid.Must(uuid.NewV7()), Name: name, CreatedAt: now, UpdatedAt: now}
	if err := h.repo.Create(c.Request.Context(), row); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Created(c, DTO{ID: row.ID, Name: row.Name})
}

func (h *Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.WriteError(c, httpx.NotFound("category not found"))
		return
	}
	var req nameRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.WriteError(c, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	exists, err := h.repo.ExistsByName(c.Request.Context(), name, &id)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	if exists {
		httpx.WriteError(c, httpx.Conflict("DUPLICATE", "category already exists", nil))
		return
	}
	if err := h.repo.Update(c.Request.Context(), id, name); err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.OK(c, DTO{ID: id, Name: name})
}
