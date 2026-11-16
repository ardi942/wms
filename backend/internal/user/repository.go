package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) WithTx(tx *gorm.DB) *Repository { return &Repository{db: tx} }

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) Create(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	fields["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(fields).Error
}

func (r *Repository) SetLastLogin(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("last_login_at", at).Error
}

func (r *Repository) List(ctx context.Context, search, role string, isActive *bool, page, pageSize int) ([]User, int64, error) {
	q := r.db.WithContext(ctx).Model(&User{})
	if role != "" {
		q = q.Where("role = ?", role)
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("email ILIKE ? OR full_name ILIKE ?", like, like)
	}
	if isActive != nil {
		q = q.Where("is_active = ?", *isActive)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	err := q.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error
	return users, total, err
}

type DepartmentRow struct {
	ID   uuid.UUID `gorm:"column:id"`
	Code string    `gorm:"column:code"`
	Name string    `gorm:"column:name"`
}

func (r *Repository) GetDepartment(ctx context.Context, id uuid.UUID) (*DepartmentRow, error) {
	var d DepartmentRow
	err := r.db.WithContext(ctx).Raw("SELECT id, code, name FROM departments WHERE id = ?", id).Scan(&d).Error
	if err != nil {
		return nil, err
	}
	if d.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &d, nil
}

func (r *Repository) DepartmentExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("departments").Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *Repository) EmailExists(ctx context.Context, email string, exceptID *uuid.UUID) (bool, error) {
	q := r.db.WithContext(ctx).Table("users").Where("email = ?", email)
	if exceptID != nil {
		q = q.Where("id <> ?", *exceptID)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

func (r *Repository) LockStatusChanges(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`SELECT pg_advisory_xact_lock(hashtext('user.status'))`).Error
}

func (r *Repository) LockActiveIDsByRole(ctx context.Context, role string) ([]uuid.UUID, error) {
	var users []User
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("role = ? AND is_active = true", role).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(users))
	for i := range users {
		ids = append(ids, users[i].ID)
	}
	return ids, nil
}

func (r *Repository) ActiveUserIDsByRoles(ctx context.Context, roles ...string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Model(&User{}).
		Where("role IN ? AND is_active = true", roles).
		Pluck("id", &ids).Error
	return ids, err
}
