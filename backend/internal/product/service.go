package product

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/audit"
	"wms/backend/internal/cache"
	"wms/backend/internal/httpx"
	"wms/backend/internal/user"
)

var skuPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{1,31}$`)

type Service struct {
	db          *gorm.DB
	repo        *Repository
	warehouseID uuid.UUID
	cache       *cache.Cache
	audit       *audit.Service
}

func NewService(db *gorm.DB, repo *Repository, warehouseID uuid.UUID, auditSvc *audit.Service) *Service {
	return &Service{db: db, repo: repo, warehouseID: warehouseID, audit: auditSvc}
}

func (s *Service) SetCache(c *cache.Cache) { s.cache = c }

type ListFilter struct {
	Search     string
	CategoryID *uuid.UUID
	IsActive   *bool
	LowStock   bool
	Sort       string
	Page       int
	PageSize   int
	Role       string
}

type CreateInput struct {
	SKU         string    `json:"sku" binding:"required"`
	Name        string    `json:"name" binding:"required,min=2,max=200"`
	Description *string   `json:"description" binding:"omitempty,max=2000"`
	CategoryID  uuid.UUID `json:"category_id" binding:"required"`
	UnitID      uuid.UUID `json:"unit_id" binding:"required"`
	MinStock    int       `json:"min_stock" binding:"gte=0"`
	UnitCost    *float64  `json:"unit_cost" binding:"omitempty,gte=0"`
}

type UpdateInput struct {
	SKU         *string    `json:"sku"`
	Name        *string    `json:"name" binding:"omitempty,min=2,max=200"`
	Description *string    `json:"description" binding:"omitempty,max=2000"`
	CategoryID  *uuid.UUID `json:"category_id"`
	UnitID      *uuid.UUID `json:"unit_id"`
	MinStock    *int       `json:"min_stock" binding:"omitempty,gte=0"`
	UnitCost    *float64   `json:"unit_cost" binding:"omitempty,gte=0"`
	IsActive    *bool      `json:"is_active"`
}

type listCache struct {
	Items []DTO `json:"items"`
	Total int64 `json:"total"`
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]DTO, int64, error) {
	if f.Role == user.RoleDepartmentUser {
		active := true
		f.IsActive = &active
	}
	var cacheKey string
	if s.cache != nil {
		ver := s.cache.Version(ctx, cache.KeyProductsVersion)
		cacheKey = fmt.Sprintf("wms:v1:products:list:%d:%s", ver, listFingerprint(f))
		var cached listCache
		if s.cache.GetJSON(ctx, cacheKey, &cached) {
			return cached.Items, cached.Total, nil
		}
	}
	rows, total, err := s.repo.List(ctx, Filter{
		Search: f.Search, CategoryID: f.CategoryID, IsActive: f.IsActive,
		LowStock: f.LowStock, Sort: f.Sort, Page: f.Page, PageSize: f.PageSize,
	}, s.warehouseID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]DTO, 0, len(rows))
	for i := range rows {
		dto := toDTO(&rows[i])
		if f.Role != user.RoleSupervisor {
			dto.UnitCost = nil
		}
		out = append(out, dto)
	}
	if s.cache != nil && cacheKey != "" {
		s.cache.SetJSON(ctx, cacheKey, listCache{Items: out, Total: total}, 60*time.Second)
	}
	return out, total, nil
}

func listFingerprint(f ListFilter) string {
	category := ""
	if f.CategoryID != nil {
		category = f.CategoryID.String()
	}
	active := "nil"
	if f.IsActive != nil {
		active = fmt.Sprintf("%t", *f.IsActive)
	}
	raw := fmt.Sprintf("%s|%s|%s|%t|%s|%d|%d|%s", strings.ToLower(f.Search), category, active, f.LowStock, f.Sort, f.Page, f.PageSize, f.Role)
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, role string) (*DTO, error) {
	var cacheKey string
	if s.cache != nil {
		ver := s.cache.Version(ctx, cache.KeyProductsVersion)
		cacheKey = fmt.Sprintf("wms:v1:products:item:%d:%s:%s", ver, role, id.String())
		var cached DTO
		if s.cache.GetJSON(ctx, cacheKey, &cached) {
			return &cached, nil
		}
	}
	row, err := s.repo.Get(ctx, id, s.warehouseID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound("product not found")
		}
		return nil, err
	}
	if role == user.RoleDepartmentUser && !row.IsActive {
		return nil, httpx.NotFound("product not found")
	}
	dto := toDTO(row)
	if role != user.RoleSupervisor {
		dto.UnitCost = nil
	}
	if s.cache != nil && cacheKey != "" {
		s.cache.SetJSON(ctx, cacheKey, dto, 120*time.Second)
	}
	return &dto, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*DTO, error) {
	sku := strings.ToUpper(strings.TrimSpace(in.SKU))
	if !skuPattern.MatchString(sku) {
		return nil, httpx.FieldError("sku", "must match ^[A-Z0-9][A-Z0-9_-]{1,31}$")
	}
	name := strings.TrimSpace(in.Name)
	if len(name) < 2 {
		return nil, httpx.FieldError("name", "must be at least 2 characters")
	}
	if err := s.validateRefs(ctx, in.CategoryID, in.UnitID); err != nil {
		return nil, err
	}
	exists, err := s.repo.ExistsSKU(ctx, sku, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, httpx.Conflict("DUPLICATE", "SKU already exists", nil)
	}
	now := time.Now().UTC()
	p := &Product{
		ID:          uuid.Must(uuid.NewV7()),
		SKU:         sku,
		Name:        name,
		Description: trimPtr(in.Description),
		CategoryID:  in.CategoryID,
		UnitID:      in.UnitID,
		MinStock:    in.MinStock,
		UnitCost:    in.UnitCost,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.InvalidateProduct(ctx)
	}
	return s.Get(ctx, p.ID, user.RoleWarehouseAdmin)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput, role string, actor audit.Actor) (*DTO, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		existing, err := txRepo.GetForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return httpx.NotFound("product not found")
			}
			return err
		}
		before := productSnapshot(existing)
		after := audit.CloneMap(before)
		fields := map[string]any{}

		if in.SKU != nil {
			sku := strings.ToUpper(strings.TrimSpace(*in.SKU))
			if !skuPattern.MatchString(sku) {
				return httpx.FieldError("sku", "must match ^[A-Z0-9][A-Z0-9_-]{1,31}$")
			}
			if sku != existing.SKU {
				has, err := txRepo.HasMovements(ctx, id)
				if err != nil {
					return err
				}
				if has {
					return httpx.Conflict("CONFLICT", "SKU cannot be changed after stock transactions exist", nil)
				}
				exists, err := txRepo.ExistsSKU(ctx, sku, &id)
				if err != nil {
					return err
				}
				if exists {
					return httpx.Conflict("DUPLICATE", "SKU already exists", nil)
				}
				fields["sku"] = sku
				after["sku"] = sku
			}
		}
		if in.Name != nil {
			name := strings.TrimSpace(*in.Name)
			if len(name) < 2 {
				return httpx.FieldError("name", "must be at least 2 characters")
			}
			if existing.Name != name {
				fields["name"] = name
				after["name"] = name
			}
		}
		if in.Description != nil {
			next := trimPtr(in.Description)
			if !sameDescription(existing.Description, next) {
				fields["description"] = next
				after["description"] = descriptionValue(next)
			}
		}
		if in.CategoryID != nil {
			if err := validateRefsWith(ctx, txRepo, *in.CategoryID, existing.UnitID); err != nil {
				return err
			}
			if existing.CategoryID != *in.CategoryID {
				fields["category_id"] = *in.CategoryID
				after["category_id"] = in.CategoryID.String()
			}
		}
		if in.UnitID != nil {
			if err := validateRefsWith(ctx, txRepo, existing.CategoryID, *in.UnitID); err != nil {
				return err
			}
			if existing.UnitID != *in.UnitID {
				fields["unit_id"] = *in.UnitID
				after["unit_id"] = in.UnitID.String()
			}
		}
		if in.MinStock != nil && existing.MinStock != *in.MinStock {
			fields["min_stock"] = *in.MinStock
			after["min_stock"] = *in.MinStock
		}
		if in.UnitCost != nil && !sameFloat(existing.UnitCost, in.UnitCost) {
			fields["unit_cost"] = *in.UnitCost
			after["unit_cost"] = *in.UnitCost
		}
		if in.IsActive != nil && existing.IsActive != *in.IsActive {
			fields["is_active"] = *in.IsActive
			after["is_active"] = *in.IsActive
		}
		if len(fields) == 0 {
			return nil
		}
		if err := txRepo.Update(ctx, id, fields); err != nil {
			return err
		}

		changed := audit.ChangedFields(before, after)
		other := audit.Excluding(changed, "unit_cost", "is_active")
		if len(other) > 0 {
			if err := s.audit.Record(ctx, tx, audit.Entry{
				EntityType:    audit.EntityProduct,
				EntityID:      id,
				Action:        audit.ActionProductUpdated,
				EntityLabel:   existing.SKU + " — " + existing.Name,
				Actor:         actor,
				ChangedFields: other,
				Before:        audit.Subset(before, other),
				After:         audit.Subset(after, other),
			}); err != nil {
				return err
			}
		}
		if slices.Contains(changed, "unit_cost") {
			if err := s.audit.Record(ctx, tx, audit.Entry{
				EntityType:    audit.EntityProduct,
				EntityID:      id,
				Action:        audit.ActionProductCost,
				EntityLabel:   existing.SKU + " — " + existing.Name,
				Actor:         actor,
				ChangedFields: []string{"unit_cost"},
				Before:        audit.Subset(before, []string{"unit_cost"}),
				After:         audit.Subset(after, []string{"unit_cost"}),
			}); err != nil {
				return err
			}
		}
		if slices.Contains(changed, "is_active") {
			action := audit.ActionProductActivated
			if !*in.IsActive {
				action = audit.ActionProductDeactivated
			}
			if err := s.audit.Record(ctx, tx, audit.Entry{
				EntityType:    audit.EntityProduct,
				EntityID:      id,
				Action:        action,
				EntityLabel:   existing.SKU + " — " + existing.Name,
				Actor:         actor,
				ChangedFields: []string{"is_active"},
				Before:        audit.Subset(before, []string{"is_active"}),
				After:         audit.Subset(after, []string{"is_active"}),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.InvalidateProduct(ctx)
	}
	return s.Get(ctx, id, role)
}

func (s *Service) History(ctx context.Context, id uuid.UUID, page, pageSize int, role string) ([]audit.DTO, int64, error) {
	if _, err := s.repo.Get(ctx, id, s.warehouseID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, httpx.NotFound("product not found")
		}
		return nil, 0, err
	}
	return s.audit.HistoryFor(ctx, audit.EntityProduct, id, page, pageSize, role == user.RoleSupervisor)
}

func (s *Service) validateRefs(ctx context.Context, categoryID, unitID uuid.UUID) error {
	return validateRefsWith(ctx, s.repo, categoryID, unitID)
}

func validateRefsWith(ctx context.Context, repo *Repository, categoryID, unitID uuid.UUID) error {
	ok, err := repo.RefExists(ctx, "categories", categoryID)
	if err != nil {
		return err
	}
	if !ok {
		return httpx.FieldError("category_id", "category does not exist")
	}
	ok, err = repo.RefExists(ctx, "units", unitID)
	if err != nil {
		return err
	}
	if !ok {
		return httpx.FieldError("unit_id", "unit does not exist")
	}
	return nil
}

func productSnapshot(p *Product) map[string]any {
	return map[string]any{
		"sku":         p.SKU,
		"name":        p.Name,
		"description": descriptionValue(p.Description),
		"category_id": p.CategoryID.String(),
		"unit_id":     p.UnitID.String(),
		"min_stock":   p.MinStock,
		"unit_cost":   costValue(p.UnitCost),
		"is_active":   p.IsActive,
	}
}

func descriptionValue(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func costValue(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func sameDescription(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func sameFloat(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func toDTO(r *row) DTO {
	return DTO{
		ID:             r.ID,
		SKU:            r.SKU,
		Name:           r.Name,
		Description:    r.Description,
		Category:       CategoryRef{ID: r.CategoryID, Name: r.CategoryName},
		Unit:           UnitRef{ID: r.UnitID, Code: r.UnitCode, Name: r.UnitName},
		MinStock:       r.MinStock,
		UnitCost:       r.UnitCost,
		IsActive:       r.IsActive,
		CurrentStock:   r.Quantity,
		IsLowStock:     r.MinStock > 0 && r.Quantity < r.MinStock,
		ImageURL:       productImageURL(r.ID, r.ImageKey),
		ImageUpdatedAt: r.ImageUpdatedAt,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}

func productImageURL(id uuid.UUID, key *string) *string {
	if key == nil || *key == "" {
		return nil
	}
	url := "/products/" + id.String() + "/image"
	return &url
}

func trimPtr(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}
