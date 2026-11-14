package product

import (
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	SKU         string    `gorm:"column:sku"`
	Name        string    `gorm:"column:name"`
	Description *string   `gorm:"column:description"`
	CategoryID  uuid.UUID `gorm:"column:category_id;type:uuid"`
	UnitID      uuid.UUID `gorm:"column:unit_id;type:uuid"`
	MinStock    int       `gorm:"column:min_stock"`
	UnitCost    *float64  `gorm:"column:unit_cost"`
	IsActive    bool      `gorm:"column:is_active"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (Product) TableName() string { return "products" }

type CategoryRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type UnitRef struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

type DTO struct {
	ID           uuid.UUID    `json:"id"`
	SKU          string       `json:"sku"`
	Name         string       `json:"name"`
	Description  *string      `json:"description"`
	Category     CategoryRef  `json:"category"`
	Unit         UnitRef      `json:"unit"`
	MinStock     int          `json:"min_stock"`
	UnitCost     *float64     `json:"unit_cost"`
	IsActive     bool         `json:"is_active"`
	CurrentStock int          `json:"current_stock"`
	IsLowStock   bool         `json:"is_low_stock"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}
