package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/audit"
	"wms/backend/internal/httpx"
	"wms/backend/internal/media"
)

type Service struct {
	db    *gorm.DB
	repo  *Repository
	audit *audit.Service
	media *media.Service
}

func NewService(db *gorm.DB, repo *Repository, auditSvc *audit.Service, mediaSvc *media.Service) *Service {
	return &Service{db: db, repo: repo, audit: auditSvc, media: mediaSvc}
}

type ListFilter struct {
	Search   string
	Role     string
	IsActive *bool
	Page     int
	PageSize int
}

type CreateInput struct {
	Email        string     `json:"email" binding:"required,email"`
	FullName     string     `json:"full_name" binding:"required,min=2,max=200"`
	Password     string     `json:"password" binding:"required,min=8,max=72"`
	Role         string     `json:"role"`
	DepartmentID *uuid.UUID `json:"department_id"`
}

type UpdateInput struct {
	FullName     *string    `json:"full_name" binding:"omitempty,min=2,max=200"`
	Password     *string    `json:"password" binding:"omitempty,min=8,max=72"`
	DepartmentID *uuid.UUID `json:"department_id"`
	IsActive     *bool      `json:"is_active"`
}

type StatusInput struct {
	Reason string `json:"reason"`
}

func parseRole(raw string) (string, error) {
	role := strings.ToUpper(strings.TrimSpace(raw))
	switch role {
	case "", RoleDepartmentUser, RoleSupervisor, RoleWarehouseAdmin:
		return role, nil
	default:
		return "", httpx.FieldError("role", "must be DEPARTMENT_USER, SUPERVISOR, or WAREHOUSE_ADMIN")
	}
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]DTO, int64, error) {
	users, total, err := s.repo.List(ctx, strings.TrimSpace(f.Search), f.Role, f.IsActive, f.Page, f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]DTO, 0, len(users))
	for i := range users {
		out = append(out, s.toDTO(ctx, &users[i]))
	}
	return out, total, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput, actor audit.Actor) (*DTO, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, httpx.FieldError("email", "must be a valid email")
	}
	name := strings.TrimSpace(in.FullName)
	if len(name) < 2 {
		return nil, httpx.FieldError("full_name", "must be at least 2 characters")
	}
	role, err := parseRole(in.Role)
	if err != nil {
		return nil, err
	}
	if role == "" {
		role = RoleDepartmentUser
	}
	var departmentID *uuid.UUID
	if role == RoleDepartmentUser {
		if in.DepartmentID == nil {
			return nil, httpx.FieldError("department_id", "department is required for this role")
		}
		deptOK, err := s.repo.DepartmentExists(ctx, *in.DepartmentID)
		if err != nil {
			return nil, err
		}
		if !deptOK {
			return nil, httpx.FieldError("department_id", "department does not exist")
		}
		departmentID = in.DepartmentID
	} else if in.DepartmentID != nil {
		return nil, httpx.FieldError("department_id", "must be empty for this role")
	}
	exists, err := s.repo.EmailExists(ctx, email, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, httpx.Conflict("DUPLICATE", "email already in use", nil)
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	u := &User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        email,
		PasswordHash: hash,
		FullName:     name,
		Role:         role,
		DepartmentID: departmentID,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		if err := txRepo.Create(ctx, u); err != nil {
			return err
		}
		after := userSnapshot(u)
		return s.audit.Record(ctx, tx, audit.Entry{
			EntityType:    audit.EntityUser,
			EntityID:      u.ID,
			Action:        audit.ActionUserCreated,
			EntityLabel:   u.FullName,
			Actor:         actor,
			ChangedFields: audit.ChangedFields(map[string]any{}, after),
			After:         after,
		})
	}); err != nil {
		return nil, err
	}
	dto := s.toDTO(ctx, u)
	return &dto, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput, actor audit.Actor) (*DTO, error) {
	if in.IsActive != nil {
		return nil, httpx.FieldError("is_active", "must be changed through the activate or deactivate endpoint")
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		current, err := txRepo.GetByIDForUpdate(ctx, id)
		if err != nil {
			return mapUserError(err)
		}
		if in.DepartmentID != nil && current.Role != RoleDepartmentUser {
			return httpx.FieldError("department_id", "must be empty for this role")
		}
		before := userSnapshot(current)
		after := audit.CloneMap(before)
		fields := map[string]any{}
		if in.FullName != nil {
			name := strings.TrimSpace(*in.FullName)
			if len(name) < 2 {
				return httpx.FieldError("full_name", "must be at least 2 characters")
			}
			if current.FullName != name {
				fields["full_name"] = name
				after["full_name"] = name
			}
		}
		if in.DepartmentID != nil {
			ok, err := txRepo.DepartmentExists(ctx, *in.DepartmentID)
			if err != nil {
				return err
			}
			if !ok {
				return httpx.FieldError("department_id", "department does not exist")
			}
			currentDepartment := ""
			if current.DepartmentID != nil {
				currentDepartment = current.DepartmentID.String()
			}
			if currentDepartment != in.DepartmentID.String() {
				fields["department_id"] = *in.DepartmentID
				after["department_id"] = in.DepartmentID.String()
			}
		}
		passwordChanged := false
		if in.Password != nil {
			hash, err := HashPassword(*in.Password)
			if err != nil {
				return err
			}
			fields["password_hash"] = hash
			passwordChanged = true
		}
		if len(fields) > 0 {
			if err := txRepo.Update(ctx, id, fields); err != nil {
				return err
			}
		}
		changed := audit.ChangedFields(before, after)
		if len(changed) > 0 {
			if err := s.audit.Record(ctx, tx, audit.Entry{
				EntityType:    audit.EntityUser,
				EntityID:      id,
				Action:        audit.ActionUserProfileUpdate,
				EntityLabel:   current.FullName,
				Actor:         actor,
				ChangedFields: changed,
				Before:        audit.Subset(before, changed),
				After:         audit.Subset(after, changed),
			}); err != nil {
				return err
			}
		}
		if passwordChanged {
			if err := s.audit.Record(ctx, tx, audit.Entry{
				EntityType:    audit.EntityUser,
				EntityID:      id,
				Action:        audit.ActionUserPassword,
				EntityLabel:   current.FullName,
				Actor:         actor,
				ChangedFields: []string{"password"},
				After:         map[string]any{"password_changed": true},
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := s.toDTO(ctx, updated)
	return &dto, nil
}

func (s *Service) Activate(ctx context.Context, id uuid.UUID, reason string, actor audit.Actor) (*DTO, bool, error) {
	return s.setActive(ctx, id, true, reason, actor)
}

func (s *Service) Deactivate(ctx context.Context, id uuid.UUID, reason string, actor audit.Actor) (*DTO, bool, error) {
	return s.setActive(ctx, id, false, reason, actor)
}

func (s *Service) setActive(ctx context.Context, id uuid.UUID, active bool, reason string, actor audit.Actor) (*DTO, bool, error) {
	reason, rerr := audit.NormalizeReason(reason, true)
	if rerr != nil {
		return nil, false, rerr
	}
	changed := false
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		if err := txRepo.LockStatusChanges(ctx); err != nil {
			return err
		}
		current, err := txRepo.GetByIDForUpdate(ctx, id)
		if err != nil {
			return mapUserError(err)
		}
		if !active && current.ID == actor.ID {
			return httpx.Conflict("CONFLICT", "you cannot deactivate your own account", nil)
		}
		if current.IsActive == active {
			return nil
		}
		if !active && current.Role == RoleWarehouseAdmin {
			adminIDs, err := txRepo.LockActiveIDsByRole(ctx, RoleWarehouseAdmin)
			if err != nil {
				return err
			}
			if len(adminIDs) <= 1 {
				return httpx.Conflict("CONFLICT", "at least one active warehouse admin is required", nil)
			}
		}
		action := audit.ActionUserActivated
		if !active {
			action = audit.ActionUserDeactivated
		}
		if err := txRepo.Update(ctx, id, map[string]any{"is_active": active}); err != nil {
			return err
		}
		if err := s.audit.Record(ctx, tx, audit.Entry{
			EntityType:    audit.EntityUser,
			EntityID:      id,
			Action:        action,
			EntityLabel:   current.FullName,
			Actor:         actor,
			Reason:        reason,
			ChangedFields: []string{"is_active"},
			Before:        map[string]any{"is_active": current.IsActive},
			After:         map[string]any{"is_active": active},
		}); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, false, err
	}
	dto := s.toDTO(ctx, updated)
	return &dto, changed, nil
}

type ImageFile struct {
	Reader    io.ReadSeekCloser
	MIME      string
	UpdatedAt time.Time
}

func (s *Service) SetImage(ctx context.Context, id uuid.UUID, r io.Reader, actor audit.Actor) (*DTO, error) {
	if s.media == nil {
		return nil, fmt.Errorf("media service is not configured")
	}
	img, err := s.media.Validate(r)
	if err != nil {
		return nil, media.AsAppError(err)
	}
	key, err := media.NewKey("users", img.Extension)
	if err != nil {
		return nil, err
	}
	if err := s.media.Put(ctx, key, img); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var previousKey *string
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		existing, err := txRepo.GetByIDForUpdate(ctx, id)
		if err != nil {
			return mapUserError(err)
		}
		previousKey = existing.ImageKey
		if err := txRepo.Update(ctx, id, map[string]any{
			"image_key":        key,
			"image_mime":       img.MIME,
			"image_updated_at": now,
		}); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Entry{
			EntityType:    audit.EntityUser,
			EntityID:      id,
			Action:        audit.ActionUserProfileUpdate,
			EntityLabel:   existing.FullName,
			Actor:         actor,
			ChangedFields: []string{"image"},
			Before:        map[string]any{"image": timestampValue(existing.ImageUpdatedAt)},
			After:         map[string]any{"image": timestampValue(&now)},
		})
	})
	if err != nil {
		_ = s.media.Delete(ctx, key)
		return nil, err
	}
	if previousKey != nil && *previousKey != "" && *previousKey != key {
		_ = s.media.Delete(ctx, *previousKey)
	}
	return s.Get(ctx, id)
}

func (s *Service) DeleteImage(ctx context.Context, id uuid.UUID, actor audit.Actor) error {
	if s.media == nil {
		return fmt.Errorf("media service is not configured")
	}
	var previousKey *string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txRepo := s.repo.WithTx(tx)
		existing, err := txRepo.GetByIDForUpdate(ctx, id)
		if err != nil {
			return mapUserError(err)
		}
		if existing.ImageKey == nil || *existing.ImageKey == "" {
			return nil
		}
		previousKey = existing.ImageKey
		if err := txRepo.Update(ctx, id, map[string]any{
			"image_key":        nil,
			"image_mime":       nil,
			"image_updated_at": nil,
		}); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Entry{
			EntityType:    audit.EntityUser,
			EntityID:      id,
			Action:        audit.ActionUserProfileUpdate,
			EntityLabel:   existing.FullName,
			Actor:         actor,
			ChangedFields: []string{"image"},
			Before:        map[string]any{"image": timestampValue(existing.ImageUpdatedAt)},
			After:         map[string]any{"image": nil},
		})
	})
	if err != nil {
		return err
	}
	if previousKey != nil && *previousKey != "" {
		_ = s.media.Delete(ctx, *previousKey)
	}
	return nil
}

func (s *Service) GetImage(ctx context.Context, id uuid.UUID) (*ImageFile, error) {
	if s.media == nil {
		return nil, fmt.Errorf("media service is not configured")
	}
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapUserError(err)
	}
	if existing.ImageKey == nil || *existing.ImageKey == "" || existing.ImageMIME == nil || *existing.ImageMIME == "" {
		return nil, httpx.NotFound("image not found")
	}
	reader, err := s.media.Open(ctx, *existing.ImageKey)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return nil, httpx.NotFound("image not found")
		}
		return nil, err
	}
	updatedAt := time.Time{}
	if existing.ImageUpdatedAt != nil {
		updatedAt = *existing.ImageUpdatedAt
	}
	return &ImageFile{Reader: reader, MIME: *existing.ImageMIME, UpdatedAt: updatedAt}, nil
}

func timestampValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func (s *Service) History(ctx context.Context, id uuid.UUID, page, pageSize int) ([]audit.DTO, int64, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return nil, 0, mapUserError(err)
	}
	return s.audit.HistoryFor(ctx, audit.EntityUser, id, page, pageSize, false)
}

func (s *Service) toDTO(ctx context.Context, u *User) DTO {
	dto := ToDTO(u)
	if u.DepartmentID != nil {
		if d, err := s.repo.GetDepartment(ctx, *u.DepartmentID); err == nil {
			dto.Department = &DepartmentRef{ID: d.ID, Code: d.Code, Name: d.Name}
		}
	}
	return dto
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*DTO, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, mapUserError(err)
	}
	dto := s.toDTO(ctx, u)
	return &dto, nil
}

func mapUserError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return httpx.NotFound("user not found")
	}
	return err
}

func userSnapshot(u *User) map[string]any {
	departmentID := ""
	if u.DepartmentID != nil {
		departmentID = u.DepartmentID.String()
	}
	return map[string]any{
		"email":         u.Email,
		"full_name":     u.FullName,
		"role":          u.Role,
		"department_id": departmentID,
		"is_active":     u.IsActive,
	}
}
