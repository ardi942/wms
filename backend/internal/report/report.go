package report

import (
	"time"

	"github.com/google/uuid"
)

type InventoryRow struct {
	SKU            string     `json:"sku"`
	Product        string     `json:"product"`
	Category       string     `json:"category"`
	Unit           string     `json:"unit"`
	Quantity       int        `json:"quantity"`
	MinStock       int        `json:"min_stock"`
	IsLowStock     bool       `json:"is_low_stock"`
	UnitCost       *float64   `json:"unit_cost"`
	StockValue     float64    `json:"stock_value"`
	LastMovementAt *time.Time `json:"last_movement_at"`
}

type UsageRow struct {
	Department   string     `json:"department"`
	SKU          string     `json:"sku"`
	Product      string     `json:"product"`
	IssuedUnits  int        `json:"issued_units"`
	ReturnedUnits int       `json:"returned_units"`
	NetUsed      int        `json:"net_used"`
	LastIssuedAt *time.Time `json:"last_issued_at"`
}

type LowStockRow struct {
	SKU            string     `json:"sku"`
	Product        string     `json:"product"`
	Category       string     `json:"category"`
	Quantity       int        `json:"quantity"`
	MinStock       int        `json:"min_stock"`
	Shortage       int        `json:"shortage"`
	LastMovementAt *time.Time `json:"last_movement_at"`
}

type Filter struct {
	Search       string
	CategoryID   *uuid.UUID
	LowStock     bool
	DepartmentID *uuid.UUID
	ProductID    *uuid.UUID
	MovementType string
	DateFrom     string
	DateTo       string
	Page         int
	PageSize     int
}

type DepartmentRef struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}
