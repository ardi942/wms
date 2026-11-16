package product

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type row struct {
	ID             uuid.UUID  `gorm:"column:id"`
	SKU            string     `gorm:"column:sku"`
	Name           string     `gorm:"column:name"`
	Description    *string    `gorm:"column:description"`
	CategoryID     uuid.UUID  `gorm:"column:category_id"`
	CategoryName   string     `gorm:"column:category_name"`
	UnitID         uuid.UUID  `gorm:"column:unit_id"`
	UnitCode       string     `gorm:"column:unit_code"`
	UnitName       string     `gorm:"column:unit_name"`
	MinStock       int        `gorm:"column:min_stock"`
	UnitCost       *float64   `gorm:"column:unit_cost"`
	IsActive       bool       `gorm:"column:is_active"`
	ImageKey       *string    `gorm:"column:image_key"`
	ImageMIME      *string    `gorm:"column:image_mime"`
	ImageUpdatedAt *time.Time `gorm:"column:image_updated_at"`
	Quantity       int        `gorm:"column:quantity"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

type Filter struct {
	Search     string
	CategoryID *uuid.UUID
	IsActive   *bool
	LowStock   bool
	Sort       string
	Page       int
	PageSize   int
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) WithTx(tx *gorm.DB) *Repository { return &Repository{db: tx} }

func (r *Repository) GetForUpdate(ctx context.Context, id uuid.UUID) (*Product, error) {
	var product Product
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

const selectCols = `
  p.id, p.sku, p.name, p.description,
  p.category_id, c.name AS category_name,
  p.unit_id, u.code AS unit_code, u.name AS unit_name,
  p.min_stock, p.unit_cost, p.is_active,
  p.image_key, p.image_mime, p.image_updated_at,
  COALESCE(b.quantity, 0) AS quantity,
  p.created_at, p.updated_at`

func orderClause(sort string) string {
	switch sort {
	case "name":
		return "p.name ASC, p.id ASC"
	case "-name":
		return "p.name DESC, p.id DESC"
	case "sku":
		return "p.sku ASC, p.id ASC"
	case "-sku":
		return "p.sku DESC, p.id DESC"
	case "created_at":
		return "p.created_at ASC, p.id ASC"
	default:
		return "p.created_at DESC, p.id DESC"
	}
}

func (r *Repository) List(ctx context.Context, f Filter, warehouseID uuid.UUID) ([]row, int64, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if f.Search != "" {
		where = append(where, "(p.name ILIKE ? OR p.sku ILIKE ?)")
		like := "%" + strings.TrimSpace(f.Search) + "%"
		args = append(args, like, like)
	}
	if f.CategoryID != nil {
		where = append(where, "p.category_id = ?")
		args = append(args, *f.CategoryID)
	}
	if f.IsActive != nil {
		where = append(where, "p.is_active = ?")
		args = append(args, *f.IsActive)
	}
	if f.LowStock {
		where = append(where, "COALESCE(b.quantity,0) < p.min_stock AND p.min_stock > 0")
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	countSQL := `SELECT count(*) FROM products p
		LEFT JOIN inventory_balances b ON b.product_id = p.id AND b.warehouse_id = ?
		WHERE ` + whereSQL
	if err := r.db.WithContext(ctx).Raw(countSQL, append([]any{warehouseID}, args...)...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	listSQL := `SELECT ` + selectCols + `
		FROM products p
		JOIN categories c ON c.id = p.category_id
		JOIN units u ON u.id = p.unit_id
		LEFT JOIN inventory_balances b ON b.product_id = p.id AND b.warehouse_id = ?
		WHERE ` + whereSQL + `
		ORDER BY ` + orderClause(f.Sort) + `
		LIMIT ? OFFSET ?`
	listArgs := append([]any{warehouseID}, args...)
	listArgs = append(listArgs, f.PageSize, (f.Page-1)*f.PageSize)

	var rows []row
	if err := r.db.WithContext(ctx).Raw(listSQL, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID, warehouseID uuid.UUID) (*row, error) {
	sql := `SELECT ` + selectCols + `
		FROM products p
		JOIN categories c ON c.id = p.category_id
		JOIN units u ON u.id = p.unit_id
		LEFT JOIN inventory_balances b ON b.product_id = p.id AND b.warehouse_id = ?
		WHERE p.id = ?`
	var out row
	if err := r.db.WithContext(ctx).Raw(sql, warehouseID, id).Scan(&out).Error; err != nil {
		return nil, err
	}
	if out.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &out, nil
}

func (r *Repository) ExistsSKU(ctx context.Context, sku string, except *uuid.UUID) (bool, error) {
	q := r.db.WithContext(ctx).Model(&Product{}).Where("sku = ?", sku)
	if except != nil {
		q = q.Where("id <> ?", *except)
	}
	var n int64
	err := q.Count(&n).Error
	return n > 0, err
}

func (r *Repository) RefExists(ctx context.Context, table string, id uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table(table).Where("id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *Repository) HasMovements(ctx context.Context, id uuid.UUID) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Table("stock_movements").Where("product_id = ?", id).Count(&n).Error
	return n > 0, err
}

func (r *Repository) Create(ctx context.Context, p *Product) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	fields["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).Model(&Product{}).Where("id = ?", id).Updates(fields).Error
}
