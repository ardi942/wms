package user

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleSupervisor     = "SUPERVISOR"
	RoleWarehouseAdmin = "WAREHOUSE_ADMIN"
	RoleDepartmentUser = "DEPARTMENT_USER"
)

type User struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email          string     `gorm:"column:email"`
	PasswordHash   string     `gorm:"column:password_hash"`
	FullName       string     `gorm:"column:full_name"`
	Role           string     `gorm:"column:role"`
	DepartmentID   *uuid.UUID `gorm:"column:department_id;type:uuid"`
	IsActive       bool       `gorm:"column:is_active"`
	LastLoginAt    *time.Time `gorm:"column:last_login_at"`
	ImageKey       *string    `gorm:"column:image_key"`
	ImageMIME      *string    `gorm:"column:image_mime"`
	ImageUpdatedAt *time.Time `gorm:"column:image_updated_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (User) TableName() string { return "users" }

type DepartmentRef struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

type DTO struct {
	ID             uuid.UUID      `json:"id"`
	Email          string         `json:"email"`
	FullName       string         `json:"full_name"`
	Role           string         `json:"role"`
	Department     *DepartmentRef `json:"department"`
	IsActive       bool           `json:"is_active"`
	ImageURL       *string        `json:"image_url"`
	ImageUpdatedAt *time.Time     `json:"image_updated_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func ToDTO(u *User) DTO {
	return DTO{
		ID:             u.ID,
		Email:          u.Email,
		FullName:       u.FullName,
		Role:           u.Role,
		IsActive:       u.IsActive,
		ImageURL:       userImageURL(u.ID, u.ImageKey),
		ImageUpdatedAt: u.ImageUpdatedAt,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}

func userImageURL(id uuid.UUID, key *string) *string {
	if key == nil || *key == "" {
		return nil
	}
	url := "/users/" + id.String() + "/image"
	return &url
}
