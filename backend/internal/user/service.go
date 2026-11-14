package user

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/httpx"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type ListFilter struct {
	Search   string
	IsActive *bool
	Page     int
	PageSize int
}

type CreateInput struct {
	Email        string    `json:"email" binding:"required,email"`
	FullName     string    `json:"full_name" binding:"required,min=2,max=200"`
	Password     string    `json:"password" binding:"required,min=8,max=72"`
	DepartmentID uuid.UUID `json:"department_id" binding:"required"`
}

type UpdateInput struct {
	FullName     *string    `json:"full_name" binding:"omitempty,min=2,max=200"`
	Password     *string    `json:"password" binding:"omitempty,min=8,max=72"`
	DepartmentID *uuid.UUID `json:"department_id"`
	IsActive     *bool      `json:"is_active"`
}

func (s *Service) List(ctx context.Context, f ListFilter) ([]DTO, int64, error) {
	users, total, err := s.repo.List(ctx, strings.TrimSpace(f.Search), f.IsActive, f.Page, f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]DTO, 0, len(users))
	for i := range users {
		out = append(out, s.toDTO(ctx, &users[i]))
	}
	return out, total, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*DTO, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, httpx.FieldError("email", "must be a valid email")
	}
	name := strings.TrimSpace(in.FullName)
	if len(name) < 2 {
		return nil, httpx.FieldError("full_name", "must be at least 2 characters")
	}
	exists, err := s.repo.EmailExists(ctx, email, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, httpx.Conflict("DUPLICATE", "email already in use", nil)
	}
	deptOK, err := s.repo.DepartmentExists(ctx, in.DepartmentID)
	if err != nil {
		return nil, err
	}
	if !deptOK {
		return nil, httpx.FieldError("department_id", "department does not exist")
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
		Role:         RoleDepartmentUser,
		DepartmentID: &in.DepartmentID,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	dto := s.toDTO(ctx, u)
	return &dto, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*DTO, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound("user not found")
		}
		return nil, err
	}
	if u.Role != RoleDepartmentUser {
		return nil, httpx.Forbidden("only department users can be managed here")
	}
	fields := map[string]any{}
	if in.FullName != nil {
		name := strings.TrimSpace(*in.FullName)
		if len(name) < 2 {
			return nil, httpx.FieldError("full_name", "must be at least 2 characters")
		}
		fields["full_name"] = name
	}
	if in.Password != nil {
		hash, err := HashPassword(*in.Password)
		if err != nil {
			return nil, err
		}
		fields["password_hash"] = hash
	}
	if in.DepartmentID != nil {
		ok, err := s.repo.DepartmentExists(ctx, *in.DepartmentID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, httpx.FieldError("department_id", "department does not exist")
		}
		fields["department_id"] = *in.DepartmentID
	}
	if in.IsActive != nil {
		fields["is_active"] = *in.IsActive
	}
	if len(fields) > 0 {
		if err := s.repo.Update(ctx, id, fields); err != nil {
			return nil, err
		}
	}
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := s.toDTO(ctx, updated)
	return &dto, nil
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.NotFound("user not found")
		}
		return nil, err
	}
	dto := s.toDTO(ctx, u)
	return &dto, nil
}
