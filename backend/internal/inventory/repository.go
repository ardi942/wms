package inventory

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db          *gorm.DB
	warehouseID uuid.UUID
}

func NewRepository(db *gorm.DB, warehouseID uuid.UUID) *Repository {
	return &Repository{db: db, warehouseID: warehouseID}
}

func (r *Repository) DB() *gorm.DB                  { return r.db }
func (r *Repository) WarehouseID() uuid.UUID        { return r.warehouseID }

type BalanceFilter struct {
	Search     string
	CategoryID *uuid.UUID
	LowStock   bool
	Page       int
	PageSize   int
	Role       string
}

func (r *Repository) ListBalances(ctx context.Context, f BalanceFilter) ([]InventoryDTO, int64, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if f.Search != "" {
		like := "%" + strings.TrimSpace(f.Search) + "%"
		where = append(where, "(p.name ILIKE ? OR p.sku ILIKE ?)")
		args = append(args, like, like)
	}
	if f.CategoryID != nil {
		where = append(where, "p.category_id = ?")
		args = append(args, *f.CategoryID)
	}
	if f.LowStock {
		where = append(where, "COALESCE(b.quantity,0) < p.min_stock AND p.min_stock > 0")
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	countSQL := `SELECT count(*) FROM products p
		LEFT JOIN inventory_balances b ON b.product_id = p.id AND b.warehouse_id = ?
		WHERE ` + whereSQL
	if err := r.db.WithContext(ctx).Raw(countSQL, append([]any{r.warehouseID}, args...)...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	sql := `SELECT p.id AS product_id, p.sku, p.name, c.name AS category,
		u.code AS unit, COALESCE(b.quantity,0) AS quantity, p.min_stock, p.unit_cost,
		(COALESCE(b.quantity,0) * COALESCE(p.unit_cost,0))::float8 AS stock_value
		FROM products p
		JOIN categories c ON c.id = p.category_id
		JOIN units u ON u.id = p.unit_id
		LEFT JOIN inventory_balances b ON b.product_id = p.id AND b.warehouse_id = ?
		WHERE ` + whereSQL + `
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT ? OFFSET ?`
	listArgs := append([]any{r.warehouseID}, args...)
	listArgs = append(listArgs, f.PageSize, (f.Page-1)*f.PageSize)
	var rows []InventoryDTO
	if err := r.db.WithContext(ctx).Raw(sql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	for i := range rows {
		rows[i].IsLowStock = rows[i].MinStock > 0 && rows[i].Quantity < rows[i].MinStock
	}
	return rows, total, nil
}

func (r *Repository) GetBalance(ctx context.Context, productID uuid.UUID) (*InventoryDTO, error) {
	sql := `SELECT p.id AS product_id, p.sku, p.name, c.name AS category,
		u.code AS unit, COALESCE(b.quantity,0) AS quantity, p.min_stock, p.unit_cost,
		(COALESCE(b.quantity,0) * COALESCE(p.unit_cost,0))::float8 AS stock_value
		FROM products p
		JOIN categories c ON c.id = p.category_id
		JOIN units u ON u.id = p.unit_id
		LEFT JOIN inventory_balances b ON b.product_id = p.id AND b.warehouse_id = ?
		WHERE p.id = ?`
	var out InventoryDTO
	if err := r.db.WithContext(ctx).Raw(sql, r.warehouseID, productID).Scan(&out).Error; err != nil {
		return nil, err
	}
	if out.ProductID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	out.IsLowStock = out.MinStock > 0 && out.Quantity < out.MinStock
	return &out, nil
}

type MovementFilter struct {
	ProductID     *uuid.UUID
	MovementType  string
	ReferenceType string
	DepartmentID  *uuid.UUID
	DateFrom      string
	DateTo        string
	Page          int
	PageSize      int
}

type movementRow struct {
	ID              uuid.UUID  `gorm:"column:id"`
	ProductID       uuid.UUID  `gorm:"column:product_id"`
	SKU             string     `gorm:"column:sku"`
	ProductName     string     `gorm:"column:product_name"`
	MovementType    string     `gorm:"column:movement_type"`
	QuantityDelta   int        `gorm:"column:quantity_delta"`
	BalanceAfter    int        `gorm:"column:balance_after"`
	ReferenceType   string     `gorm:"column:reference_type"`
	ReferenceID     uuid.UUID  `gorm:"column:reference_id"`
	ReferenceNumber string     `gorm:"column:reference_number"`
	DepartmentID    *uuid.UUID `gorm:"column:department_id"`
	DepartmentCode  *string    `gorm:"column:department_code"`
	DepartmentName  *string    `gorm:"column:department_name"`
	ActorID         uuid.UUID  `gorm:"column:actor_id"`
	ActorName       string     `gorm:"column:actor_name"`
	Supplier        *string    `gorm:"column:supplier"`
	Notes           *string    `gorm:"column:notes"`
	Reason          *string    `gorm:"column:reason"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (r *Repository) ListMovements(ctx context.Context, f MovementFilter) ([]MovementDTO, int64, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if f.ProductID != nil {
		where = append(where, "m.product_id = ?")
		args = append(args, *f.ProductID)
	}
	if f.MovementType != "" {
		where = append(where, "m.movement_type = ?")
		args = append(args, f.MovementType)
	}
	if f.ReferenceType != "" {
		where = append(where, "m.reference_type = ?")
		args = append(args, f.ReferenceType)
	}
	if f.DepartmentID != nil {
		where = append(where, "m.department_id = ?")
		args = append(args, *f.DepartmentID)
	}
	if f.DateFrom != "" {
		where = append(where, "m.created_at >= ?::date")
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		where = append(where, "m.created_at < (?::date + interval '1 day')")
		args = append(args, f.DateTo)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.WithContext(ctx).Raw(`SELECT count(*) FROM stock_movements m WHERE `+whereSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	sql := `SELECT m.id, m.product_id, p.sku, p.name AS product_name,
		m.movement_type, m.quantity_delta, m.balance_after,
		m.reference_type, m.reference_id, m.reference_number,
		m.department_id, d.code AS department_code, d.name AS department_name,
		m.actor_id, u.full_name AS actor_name,
		m.supplier, m.notes, m.reason, m.created_at
		FROM stock_movements m
		JOIN products p ON p.id = m.product_id
		JOIN users u ON u.id = m.actor_id
		LEFT JOIN departments d ON d.id = m.department_id
		WHERE ` + whereSQL + `
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), f.PageSize, (f.Page-1)*f.PageSize)
	var rows []movementRow
	if err := r.db.WithContext(ctx).Raw(sql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]MovementDTO, 0, len(rows))
	for i := range rows {
		out = append(out, toMovementDTO(&rows[i]))
	}
	return out, total, nil
}

func (r *Repository) GetMovementDTO(ctx context.Context, id uuid.UUID) (*MovementDTO, error) {
	sql := `SELECT m.id, m.product_id, p.sku, p.name AS product_name,
		m.movement_type, m.quantity_delta, m.balance_after,
		m.reference_type, m.reference_id, m.reference_number,
		m.department_id, d.code AS department_code, d.name AS department_name,
		m.actor_id, u.full_name AS actor_name,
		m.supplier, m.notes, m.reason, m.created_at
		FROM stock_movements m
		JOIN products p ON p.id = m.product_id
		JOIN users u ON u.id = m.actor_id
		LEFT JOIN departments d ON d.id = m.department_id
		WHERE m.id = ?`
	var row movementRow
	if err := r.db.WithContext(ctx).Raw(sql, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	dto := toMovementDTO(&row)
	return &dto, nil
}

func toMovementDTO(r *movementRow) MovementDTO {
	dto := MovementDTO{
		ID:              r.ID,
		Product:         ProductRef{ID: r.ProductID, SKU: r.SKU, Name: r.ProductName},
		MovementType:    r.MovementType,
		QuantityDelta:   r.QuantityDelta,
		BalanceAfter:    r.BalanceAfter,
		ReferenceType:   r.ReferenceType,
		ReferenceID:     r.ReferenceID,
		ReferenceNumber: r.ReferenceNumber,
		Actor:           ActorRef{ID: r.ActorID, FullName: r.ActorName},
		Supplier:        r.Supplier,
		Notes:           r.Notes,
		Reason:          r.Reason,
		CreatedAt:       r.CreatedAt,
	}
	if r.DepartmentID != nil && r.DepartmentCode != nil && r.DepartmentName != nil {
		dto.Department = &DepartmentRef{ID: *r.DepartmentID, Code: *r.DepartmentCode, Name: *r.DepartmentName}
	}
	return dto
}
