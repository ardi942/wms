package stockreturn

import (
	"time"

	"github.com/google/uuid"
	"wms/backend/internal/user"
)

const (
	StatusPending   = "PENDING"
	StatusConfirmed = "CONFIRMED"
	StatusRejected  = "REJECTED"
	StatusCanceled  = "CANCELLED"
)

type Return struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	ReturnNumber    string     `gorm:"column:return_number"`
	DepartmentID    uuid.UUID  `gorm:"column:department_id;type:uuid"`
	RequestedBy     uuid.UUID  `gorm:"column:requested_by;type:uuid"`
	Status          string     `gorm:"column:status"`
	Reason          string     `gorm:"column:reason"`
	RejectionReason *string    `gorm:"column:rejection_reason"`
	ProcessedBy     *uuid.UUID `gorm:"column:processed_by;type:uuid"`
	ProcessedAt     *time.Time `gorm:"column:processed_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (Return) TableName() string { return "stock_returns" }

type Item struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	ReturnID     uuid.UUID `gorm:"column:return_id;type:uuid"`
	ProductID    uuid.UUID `gorm:"column:product_id;type:uuid"`
	RequestedQty int       `gorm:"column:requested_qty"`
	AcceptedQty  int       `gorm:"column:accepted_qty"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (Item) TableName() string { return "stock_return_items" }

type DepartmentRef struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

type ActorRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type ProductRef struct {
	ID   uuid.UUID `json:"id"`
	SKU  string    `json:"sku"`
	Name string    `json:"name"`
	Unit string    `json:"unit"`
}

type ItemDTO struct {
	ID           uuid.UUID  `json:"id"`
	Product      ProductRef `json:"product"`
	RequestedQty int        `json:"requested_qty"`
	AcceptedQty  int        `json:"accepted_qty"`
	RemainingQty int        `json:"remaining_qty"`
}

type DTO struct {
	ID              uuid.UUID     `json:"id"`
	ReturnNumber    string        `json:"return_number"`
	Status          string        `json:"status"`
	Department      DepartmentRef `json:"department"`
	RequestedBy     ActorRef      `json:"requested_by"`
	Reason          string        `json:"reason"`
	RejectionReason *string       `json:"rejection_reason"`
	ProcessedBy     *ActorRef     `json:"processed_by"`
	ProcessedAt     *time.Time    `json:"processed_at"`
	Items           []ItemDTO     `json:"items"`
	AllowedActions  []string      `json:"allowed_actions"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type ListDTO struct {
	ID             uuid.UUID     `json:"id"`
	ReturnNumber   string        `json:"return_number"`
	Status         string        `json:"status"`
	Department     DepartmentRef `json:"department"`
	RequestedBy    ActorRef      `json:"requested_by"`
	Reason         string        `json:"reason"`
	ItemCount      int           `json:"item_count"`
	TotalRequested int           `json:"total_requested"`
	TotalAccepted  int           `json:"total_accepted"`
	AllowedActions []string      `json:"allowed_actions"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func allowedActions(status, role string, actorDept *uuid.UUID, deptID uuid.UUID) []string {
	actions := []string{}
	switch role {
	case user.RoleWarehouseAdmin:
		if status == StatusPending {
			actions = append(actions, "confirm", "reject")
		}
	case user.RoleDepartmentUser:
		if status == StatusPending && actorDept != nil && *actorDept == deptID {
			actions = append(actions, "cancel")
		}
	}
	return actions
}
