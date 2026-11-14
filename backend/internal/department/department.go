package department

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/httpx"
)

type Department struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code string    `gorm:"column:code"`
	Name string    `gorm:"column:name"`
}

func (Department) TableName() string { return "departments" }

type DTO struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context) ([]Department, error) {
	var rows []Department
	err := r.db.WithContext(ctx).Order("code ASC").Find(&rows).Error
	return rows, err
}

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(c *gin.Context) {
	rows, err := h.repo.List(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	out := make([]DTO, 0, len(rows))
	for _, d := range rows {
		out = append(out, DTO{ID: d.ID, Code: d.Code, Name: d.Name})
	}
	httpx.OK(c, out)
}
