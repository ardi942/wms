package inventory

import (
	"time"

	"github.com/google/uuid"
)

const (
	TypeIn         = "IN"
	TypeOut        = "OUT"
	TypeAdjustment = "ADJUSTMENT"

	RefReceipt    = "RECEIPT"
	RefRequest    = "REQUEST"
	RefReturn     = "RETURN"
	RefAdjustment = "ADJUSTMENT"
)

type Balance struct {
	ProductID   uuid.UUID `gorm:"column:product_id;type:uuid"`
	WarehouseID uuid.UUID `gorm:"column:warehouse_id;type:uuid"`
	Quantity    int       `gorm:"column:quantity"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (Balance) TableName() string { return "inventory_balances" }

type Movement struct {
	ID              uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	ProductID       uuid.UUID  `gorm:"column:product_id;type:uuid"`
	WarehouseID     uuid.UUID  `gorm:"column:warehouse_id;type:uuid"`
	MovementType    string     `gorm:"column:movement_type"`
	QuantityDelta   int        `gorm:"column:quantity_delta"`
	BalanceAfter    int        `gorm:"column:balance_after"`
	ReferenceType   string     `gorm:"column:reference_type"`
	ReferenceID     uuid.UUID  `gorm:"column:reference_id;type:uuid"`
	ReferenceNumber string     `gorm:"column:reference_number"`
	DepartmentID    *uuid.UUID `gorm:"column:department_id;type:uuid"`
	ActorID         uuid.UUID  `gorm:"column:actor_id;type:uuid"`
	Supplier        *string    `gorm:"column:supplier"`
	Notes           *string    `gorm:"column:notes"`
	Reason          *string    `gorm:"column:reason"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (Movement) TableName() string { return "stock_movements" }

type ProductRef struct {
	ID   uuid.UUID `json:"id"`
	SKU  string    `json:"sku"`
	Name string    `json:"name"`
}

type DepartmentRef struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

type ActorRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
}

type MovementDTO struct {
	ID              uuid.UUID      `json:"id"`
	Product         ProductRef     `json:"product"`
	MovementType    string         `json:"movement_type"`
	QuantityDelta   int            `json:"quantity_delta"`
	BalanceAfter    int            `json:"balance_after"`
	ReferenceType   string         `json:"reference_type"`
	ReferenceID     uuid.UUID      `json:"reference_id"`
	ReferenceNumber string         `json:"reference_number"`
	Department      *DepartmentRef `json:"department"`
	Actor           ActorRef       `json:"actor"`
	Supplier        *string        `json:"supplier"`
	Notes           *string        `json:"notes"`
	Reason          *string        `json:"reason"`
	CreatedAt       time.Time      `json:"created_at"`
}

type InventoryDTO struct {
	ProductID    uuid.UUID  `json:"product_id"`
	SKU          string     `json:"sku"`
	Name         string     `json:"name"`
	Category     string     `json:"category"`
	Unit         string     `json:"unit"`
	Quantity     int        `json:"quantity"`
	MinStock     int        `json:"min_stock"`
	IsLowStock   bool       `json:"is_low_stock"`
	UnitCost     *float64   `json:"unit_cost"`
	StockValue   float64    `json:"stock_value"`
}

type ProductInfo struct {
	ID       uuid.UUID `gorm:"column:id"`
	SKU      string    `gorm:"column:sku"`
	Name     string    `gorm:"column:name"`
	MinStock int       `gorm:"column:min_stock"`
	IsActive bool      `gorm:"column:is_active"`
}
