package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"wms/backend/internal/httpx"
	"wms/backend/internal/user"
)

type Service struct {
	users *user.Repository
	jwt   *JWT
}

func NewService(users *user.Repository, jwt *JWT) *Service {
	return &Service{users: users, jwt: jwt}
}

type LoginResult struct {
	Token string   `json:"token"`
	User  user.DTO `json:"user"`
}

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	u, err := s.users.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Unauthorized("invalid email or password")
		}
		return nil, err
	}
	if !u.IsActive || !VerifyPassword(u.PasswordHash, password) {
		return nil, httpx.Unauthorized("invalid email or password")
	}
	token, err := s.jwt.Sign(u)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_ = s.users.SetLastLogin(ctx, u.ID, now)
	u.LastLoginAt = &now
	dto := user.ToDTO(u)
	if u.DepartmentID != nil {
		if d, err := s.users.GetDepartment(ctx, *u.DepartmentID); err == nil {
			dto.Department = &user.DepartmentRef{ID: d.ID, Code: d.Code, Name: d.Name}
		}
	}
	return &LoginResult{Token: token, User: dto}, nil
}

func (s *Service) CurrentUser(ctx context.Context, id string) (*user.DTO, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return nil, httpx.Unauthorized("invalid token subject")
	}
	u, err := s.users.GetByID(ctx, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httpx.Unauthorized("user no longer exists")
		}
		return nil, err
	}
	if !u.IsActive {
		return nil, httpx.Unauthorized("user is inactive")
	}
	dto := user.ToDTO(u)
	if u.DepartmentID != nil {
		if d, err := s.users.GetDepartment(ctx, *u.DepartmentID); err == nil {
			dto.Department = &user.DepartmentRef{ID: d.ID, Code: d.Code, Name: d.Name}
		}
	}
	return &dto, nil
}
