package stockrequest

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusPending            = "PENDING"
	StatusPartiallyFulfilled = "PARTIALLY_FULFILLED"
	StatusFulfilled          = "FULFILLED"
	StatusRejected           = "REJECTED"
	StatusCanceled           = "CANCELLED"
)

type Request struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	RequestNumber   string     `gorm:"column:request_number"`
	DepartmentID    uuid.UUID  `gorm:"column:department_id;type:uuid"`
	RequestedBy     uuid.UUID  `gorm:"column:requested_by;type:uuid"`
	Status          string     `gorm:"column:status"`
	Purpose         string     `gorm:"column:purpose"`
	RejectionReason *string    `gorm:"column:rejection_reason"`
	ProcessedBy     *uuid.UUID `gorm:"column:processed_by;type:uuid"`
	ProcessedAt     *time.Time `gorm:"column:processed_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (Request) TableName() string { return "stock_requests" }

type Item struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	RequestID    uuid.UUID `gorm:"column:request_id;type:uuid"`
	ProductID    uuid.UUID `gorm:"column:product_id;type:uuid"`
	RequestedQty int       `gorm:"column:requested_qty"`
	IssuedQty    int       `gorm:"column:issued_qty"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (Item) TableName() string { return "stock_request_items" }

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
	IssuedQty    int        `json:"issued_qty"`
	RemainingQty int        `json:"remaining_qty"`
}

type DTO struct {
	ID              uuid.UUID      `json:"id"`
	RequestNumber   string         `json:"request_number"`
	Status          string         `json:"status"`
	Department      DepartmentRef  `json:"department"`
	RequestedBy     ActorRef       `json:"requested_by"`
	Purpose         string         `json:"purpose"`
	RejectionReason *string        `json:"rejection_reason"`
	ProcessedBy     *ActorRef      `json:"processed_by"`
	ProcessedAt     *time.Time     `json:"processed_at"`
	Items           []ItemDTO      `json:"items"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type ListDTO struct {
	ID             uuid.UUID     `json:"id"`
	RequestNumber  string        `json:"request_number"`
	Status         string        `json:"status"`
	Department     DepartmentRef `json:"department"`
	RequestedBy    ActorRef      `json:"requested_by"`
	Purpose        string        `json:"purpose"`
	ItemCount      int           `json:"item_count"`
	TotalRequested int           `json:"total_requested"`
	TotalIssued    int           `json:"total_issued"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}
