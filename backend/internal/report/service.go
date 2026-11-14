package report

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/cache"
	"wms/backend/internal/inventory"
	"wms/backend/internal/user"
)

type Service struct {
	db          *gorm.DB
	inv         *inventory.Repository
	warehouseID uuid.UUID
	cache       *cache.Cache
}

func NewService(db *gorm.DB, inv *inventory.Repository, warehouseID uuid.UUID) *Service {
	return &Service{db: db, inv: inv, warehouseID: warehouseID}
}

func (s *Service) SetCache(c *cache.Cache) { s.cache = c }

// paginate clamps a filter's page/page_size and returns LIMIT/OFFSET values.
func paginate(f Filter) (limit, offset int) {
	page := f.Page
	if page < 1 {
		page = 1
	}
	size := f.PageSize
	if size <= 0 {
		size = 20
	}
	if size > 5000 {
		size = 5000
	}
	return size, (page - 1) * size
}

// dateActivity appends an EXISTS condition limiting rows to products with
// stock movements inside the [from, to] range (inclusive of the end day).
func dateActivity(where []string, args []any, from, to string) ([]string, []any) {
	if from == "" && to == "" {
		return where, args
	}
	cond := "EXISTS (SELECT 1 FROM stock_movements m WHERE m.product_id = p.id"
	if from != "" {
		cond += " AND m.created_at >= ?::date"
		args = append(args, from)
	}
	if to != "" {
		cond += " AND m.created_at < (?::date + interval '1 day')"
		args = append(args, to)
	}
	cond += ")"
	return append(where, cond), args
}

func (s *Service) Dashboard(ctx context.Context, actor *httpxIdentity) (any, error) {
	var cacheKey string
	if s.cache != nil {
		scope := "global"
		if actor.DepartmentID != nil {
			scope = actor.DepartmentID.String()
		}
		ver := s.cache.Version(ctx, cache.KeyDashboardVersion)
		cacheKey = fmt.Sprintf("wms:v1:dashboard:%d:%s:%s", ver, actor.Role, scope)
		var cached any
		if s.cache.GetJSON(ctx, cacheKey, &cached) {
			return cached, nil
		}
	}
	out, err := s.dashboardUncached(ctx, actor)
	if err != nil {
		return nil, err
	}
	if s.cache != nil && cacheKey != "" {
		s.cache.SetJSON(ctx, cacheKey, out, 30*time.Second)
	}
	return out, nil
}

func (s *Service) dashboardUncached(ctx context.Context, actor *httpxIdentity) (any, error) {
	switch actor.Role {
	case user.RoleSupervisor:
		return s.supervisorDashboard(ctx)
	case user.RoleWarehouseAdmin:
		return s.adminDashboard(ctx)
	case user.RoleDepartmentUser:
		return s.departmentDashboard(ctx, actor.ID, actor.DepartmentID)
	default:
		return nil, fmt.Errorf("unknown role")
	}
}

type httpxIdentity struct {
	ID           uuid.UUID
	Role         string
	DepartmentID *uuid.UUID
}

func (s *Service) globalTotals(ctx context.Context) (products, units, lowCount, withoutCost int, value float64, err error) {
	err = s.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT count(*) FROM products WHERE is_active = true),
			(SELECT COALESCE(SUM(b.quantity),0) FROM inventory_balances b JOIN products p ON p.id=b.product_id WHERE p.is_active),
			(SELECT count(*) FROM products p LEFT JOIN inventory_balances b ON b.product_id=p.id AND b.warehouse_id=?
				WHERE p.is_active AND p.min_stock > 0 AND COALESCE(b.quantity,0) < p.min_stock),
			(SELECT count(*) FROM products WHERE is_active = true AND unit_cost IS NULL),
			(SELECT COALESCE(SUM(b.quantity * COALESCE(p.unit_cost,0)),0)::float8
				FROM inventory_balances b JOIN products p ON p.id=b.product_id WHERE p.is_active)`,
		s.warehouseID,
	).Row().Scan(&products, &units, &lowCount, &withoutCost, &value)
	return
}

func (s *Service) movementSummary(ctx context.Context, movementType string, since time.Time) (count, units int, err error) {
	err = s.db.WithContext(ctx).Raw(`
		SELECT count(*), COALESCE(SUM(ABS(quantity_delta)),0)
		FROM stock_movements WHERE movement_type = ? AND created_at >= ?`,
		movementType, since).Row().Scan(&count, &units)
	return
}

func (s *Service) pendingCount(ctx context.Context) (int, error) {
	var n int
	err := s.db.WithContext(ctx).Raw(`SELECT count(*) FROM stock_requests WHERE status = 'PENDING'`).Scan(&n).Error
	return n, err
}

func (s *Service) recentMovements(ctx context.Context, departmentID *uuid.UUID, limit int) ([]inventory.MovementDTO, error) {
	items, _, err := s.inv.ListMovements(ctx, inventory.MovementFilter{
		DepartmentID: departmentID, Page: 1, PageSize: limit,
	})
	return items, err
}

// RecentMovements returns a paginated, role-scoped movement list for the
// dashboard tables. It intentionally bypasses the 30s dashboard cache.
func (s *Service) RecentMovements(ctx context.Context, departmentID *uuid.UUID, page, pageSize int) ([]inventory.MovementDTO, int64, error) {
	return s.inv.ListMovements(ctx, inventory.MovementFilter{
		DepartmentID: departmentID,
		Page:         page,
		PageSize:     pageSize,
	})
}

func (s *Service) chart(ctx context.Context) ([]map[string]any, error) {
	type row struct {
		Date     string `gorm:"column:date"`
		InUnits  int    `gorm:"column:in_units"`
		OutUnits int    `gorm:"column:out_units"`
	}
	var rows []row
	err := s.db.WithContext(ctx).Raw(`
		SELECT to_char(d::date, 'YYYY-MM-DD') AS date,
			COALESCE(SUM(CASE WHEN m.movement_type='IN' THEN m.quantity_delta ELSE 0 END),0) AS in_units,
			COALESCE(SUM(CASE WHEN m.movement_type='OUT' THEN -m.quantity_delta ELSE 0 END),0) AS out_units
		FROM generate_series(current_date - interval '13 days', current_date, interval '1 day') d
		LEFT JOIN stock_movements m ON m.created_at >= d AND m.created_at < d + interval '1 day'
		GROUP BY d ORDER BY d`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{"date": r.Date, "in_units": r.InUnits, "out_units": r.OutUnits})
	}
	return out, nil
}

func (s *Service) supervisorDashboard(ctx context.Context) (any, error) {
	products, units, lowCount, withoutCost, value, err := s.globalTotals(ctx)
	if err != nil {
		return nil, err
	}
	pending, err := s.pendingCount(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	week := now.AddDate(0, 0, -7)
	inToday, inTodayUnits, err := s.movementSummary(ctx, inventory.TypeIn, today)
	if err != nil {
		return nil, err
	}
	_, inWeekUnits, err := s.movementSummary(ctx, inventory.TypeIn, week)
	if err != nil {
		return nil, err
	}
	outToday, outTodayUnits, err := s.movementSummary(ctx, inventory.TypeOut, today)
	if err != nil {
		return nil, err
	}
	_, outWeekUnits, err := s.movementSummary(ctx, inventory.TypeOut, week)
	if err != nil {
		return nil, err
	}
	_ = inToday
	_ = outToday
	recent, err := s.recentMovements(ctx, nil, 10)
	if err != nil {
		return nil, err
	}
	chart, err := s.chart(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"total_products":       products,
		"total_stock_units":    units,
		"inventory_value":      fmt.Sprintf("%.2f", value),
		"products_without_cost": withoutCost,
		"low_stock_count":      lowCount,
		"pending_requests":     pending,
		"incoming":             map[string]any{"today_units": inTodayUnits, "last_7d_units": inWeekUnits},
		"outgoing":             map[string]any{"today_units": outTodayUnits, "last_7d_units": outWeekUnits},
		"recent_movements":     recent,
		"movement_chart":       chart,
	}, nil
}

func (s *Service) adminDashboard(ctx context.Context) (any, error) {
	products, units, lowCount, withoutCost, value, err := s.globalTotals(ctx)
	if err != nil {
		return nil, err
	}
	_ = withoutCost
	_ = value
	pending, err := s.pendingCount(ctx)
	if err != nil {
		return nil, err
	}
	var lowStock []map[string]any
	type lowRow struct {
		ProductID uuid.UUID `gorm:"column:product_id"`
		SKU       string    `gorm:"column:sku"`
		Name      string    `gorm:"column:name"`
		Quantity  int       `gorm:"column:quantity"`
		MinStock  int       `gorm:"column:min_stock"`
	}
	var lowRows []lowRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT p.id AS product_id, p.sku, p.name, COALESCE(b.quantity,0) AS quantity, p.min_stock
		FROM products p
		LEFT JOIN inventory_balances b ON b.product_id=p.id AND b.warehouse_id=?
		WHERE p.is_active AND p.min_stock > 0 AND COALESCE(b.quantity,0) < p.min_stock
		ORDER BY (p.min_stock - COALESCE(b.quantity,0)) DESC, p.name ASC LIMIT 10`,
		s.warehouseID).Scan(&lowRows).Error; err != nil {
		return nil, err
	}
	for _, r := range lowRows {
		lowStock = append(lowStock, map[string]any{
			"product_id": r.ProductID, "sku": r.SKU, "name": r.Name,
			"quantity": r.Quantity, "min_stock": r.MinStock,
		})
	}
	if lowStock == nil {
		lowStock = []map[string]any{}
	}
	type previewRow struct {
		ID            uuid.UUID `gorm:"column:id"`
		RequestNumber string    `gorm:"column:request_number"`
		Department    string    `gorm:"column:department"`
		Purpose       string    `gorm:"column:purpose"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}
	var previewRows []previewRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT r.id, r.request_number, d.name AS department, r.purpose, r.created_at
		FROM stock_requests r JOIN departments d ON d.id=r.department_id
		WHERE r.status='PENDING' ORDER BY r.created_at ASC LIMIT 5`).Scan(&previewRows).Error; err != nil {
		return nil, err
	}
	preview := make([]map[string]any, 0, len(previewRows))
	for _, r := range previewRows {
		preview = append(preview, map[string]any{
			"id": r.ID, "request_number": r.RequestNumber, "department": r.Department,
			"purpose": r.Purpose, "created_at": r.CreatedAt,
		})
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	receiptsCount, receiptUnits, err := s.movementSummary(ctx, inventory.TypeIn, today)
	if err != nil {
		return nil, err
	}
	issuesCount, issueUnits, err := s.movementSummary(ctx, inventory.TypeOut, today)
	if err != nil {
		return nil, err
	}
	recent, err := s.recentMovements(ctx, nil, 10)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"total_products":          products,
		"total_stock_units":       units,
		"low_stock_count":         lowCount,
		"low_stock":               lowStock,
		"pending_requests":        pending,
		"pending_request_preview": preview,
		"today": map[string]any{
			"receipts_count": receiptsCount, "receipt_units": receiptUnits,
			"issues_count": issuesCount, "issue_units": issueUnits,
		},
		"recent_activities": recent,
	}, nil
}

func (s *Service) departmentDashboard(ctx context.Context, userID uuid.UUID, departmentID *uuid.UUID) (any, error) {
	if departmentID == nil {
		return nil, fmt.Errorf("department user has no department")
	}
	var available int
	if err := s.db.WithContext(ctx).Raw(`SELECT count(*) FROM products WHERE is_active = true`).Scan(&available).Error; err != nil {
		return nil, err
	}
	var pending, partial, fulfilled int
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT count(*) FROM stock_requests WHERE department_id=? AND status='PENDING'),
			(SELECT count(*) FROM stock_requests WHERE department_id=? AND status='PARTIALLY_FULFILLED'),
			(SELECT count(*) FROM stock_requests WHERE department_id=? AND status='FULFILLED')`,
		*departmentID, *departmentID, *departmentID).Row().Scan(&pending, &partial, &fulfilled); err != nil {
		return nil, err
	}
	from := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")
	usage, _, err := s.DepartmentUsage(ctx, Filter{DepartmentID: departmentID, DateFrom: from, Page: 1, PageSize: 5})
	if err != nil {
		return nil, err
	}
	recent, err := s.recentMovements(ctx, departmentID, 10)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"available_products":              available,
		"my_pending_requests":             pending,
		"my_partially_fulfilled_requests": partial,
		"my_fulfilled_requests":           fulfilled,
		"usage_last_30d":                  usage,
		"recent_department_movements":     recent,
	}, nil
}

func (s *Service) Inventory(ctx context.Context, f Filter) ([]InventoryRow, int64, error) {
	where := []string{"p.is_active = true"}
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
	where, args = dateActivity(where, args, f.DateFrom, f.DateTo)
	whereSQL := strings.Join(where, " AND ")
	limit, offset := paginate(f)

	var total int64
	countSQL := `SELECT count(*) FROM products p
		JOIN categories c ON c.id=p.category_id
		JOIN units u ON u.id=p.unit_id
		LEFT JOIN inventory_balances b ON b.product_id=p.id AND b.warehouse_id=?
		WHERE ` + whereSQL
	if err := s.db.WithContext(ctx).Raw(countSQL, append([]any{s.warehouseID}, args...)...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	sql := `SELECT p.sku, p.name AS product, c.name AS category, u.code AS unit,
		COALESCE(b.quantity,0) AS quantity, p.min_stock, p.unit_cost,
		(COALESCE(b.quantity,0) * COALESCE(p.unit_cost,0))::float8 AS stock_value,
		(SELECT MAX(m.created_at) FROM stock_movements m WHERE m.product_id = p.id) AS last_movement_at
		FROM products p
		JOIN categories c ON c.id=p.category_id
		JOIN units u ON u.id=p.unit_id
		LEFT JOIN inventory_balances b ON b.product_id=p.id AND b.warehouse_id=?
		WHERE ` + whereSQL + `
		ORDER BY p.name ASC, p.id ASC LIMIT ? OFFSET ?`
	listArgs := append([]any{s.warehouseID}, args...)
	listArgs = append(listArgs, limit, offset)
	var rows []InventoryRow
	if err := s.db.WithContext(ctx).Raw(sql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	for i := range rows {
		rows[i].IsLowStock = rows[i].MinStock > 0 && rows[i].Quantity < rows[i].MinStock
	}
	if rows == nil {
		rows = []InventoryRow{}
	}
	return rows, total, nil
}

func (s *Service) LowStock(ctx context.Context, f Filter) ([]LowStockRow, int64, error) {
	where := []string{"p.is_active AND p.min_stock > 0 AND COALESCE(b.quantity,0) < p.min_stock"}
	args := []any{}
	if f.CategoryID != nil {
		where = append(where, "p.category_id = ?")
		args = append(args, *f.CategoryID)
	}
	where, args = dateActivity(where, args, f.DateFrom, f.DateTo)
	whereSQL := strings.Join(where, " AND ")
	limit, offset := paginate(f)

	var total int64
	countSQL := `SELECT count(*) FROM products p
		JOIN categories c ON c.id=p.category_id
		LEFT JOIN inventory_balances b ON b.product_id=p.id AND b.warehouse_id=?
		WHERE ` + whereSQL
	if err := s.db.WithContext(ctx).Raw(countSQL, append([]any{s.warehouseID}, args...)...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	sql := `SELECT p.sku, p.name AS product, c.name AS category,
		COALESCE(b.quantity,0) AS quantity, p.min_stock,
		(p.min_stock - COALESCE(b.quantity,0)) AS shortage,
		(SELECT MAX(m.created_at) FROM stock_movements m WHERE m.product_id = p.id) AS last_movement_at
		FROM products p
		JOIN categories c ON c.id=p.category_id
		LEFT JOIN inventory_balances b ON b.product_id=p.id AND b.warehouse_id=?
		WHERE ` + whereSQL + `
		ORDER BY shortage DESC, p.name ASC LIMIT ? OFFSET ?`
	listArgs := append([]any{s.warehouseID}, args...)
	listArgs = append(listArgs, limit, offset)
	var rows []LowStockRow
	if err := s.db.WithContext(ctx).Raw(sql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []LowStockRow{}
	}
	return rows, total, nil
}

func (s *Service) Movements(ctx context.Context, f Filter) ([]inventory.MovementDTO, int64, error) {
	limit, _ := paginate(f)
	page := f.Page
	if page < 1 {
		page = 1
	}
	from, to := f.DateFrom, f.DateTo
	if from == "" && to == "" {
		from = time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")
	}
	items, total, err := s.inv.ListMovements(ctx, inventory.MovementFilter{
		ProductID: f.ProductID, MovementType: f.MovementType, DepartmentID: f.DepartmentID,
		DateFrom: from, DateTo: to, Page: page, PageSize: limit,
	})
	return items, total, err
}

func (s *Service) DepartmentUsage(ctx context.Context, f Filter) ([]UsageRow, int64, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if f.DepartmentID != nil {
		where = append(where, "m.department_id = ?")
		args = append(args, *f.DepartmentID)
	}
	if f.ProductID != nil {
		where = append(where, "m.product_id = ?")
		args = append(args, *f.ProductID)
	}
	from, to := f.DateFrom, f.DateTo
	if from == "" && to == "" {
		from = time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if from != "" {
		where = append(where, "m.created_at >= ?::date")
		args = append(args, from)
	}
	if to != "" {
		where = append(where, "m.created_at < (?::date + interval '1 day')")
		args = append(args, to)
	}
	whereSQL := strings.Join(where, " AND ")
	limit, offset := paginate(f)

	var total int64
	countSQL := `SELECT count(*) FROM (
		SELECT 1 FROM stock_movements m
		JOIN departments d ON d.id = m.department_id
		JOIN products p ON p.id = m.product_id
		WHERE ` + whereSQL + `
		GROUP BY d.id, p.id) t`
	if err := s.db.WithContext(ctx).Raw(countSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	sql := `SELECT d.name AS department, p.sku, p.name AS product,
		COALESCE(SUM(CASE WHEN m.movement_type='OUT' THEN -m.quantity_delta ELSE 0 END),0) AS issued_units,
		COALESCE(SUM(CASE WHEN m.movement_type='IN' AND m.reference_type='RETURN' THEN m.quantity_delta ELSE 0 END),0) AS returned_units,
		COALESCE(SUM(CASE WHEN m.movement_type='OUT' THEN -m.quantity_delta
			WHEN m.movement_type='IN' AND m.reference_type='RETURN' THEN -m.quantity_delta ELSE 0 END),0) AS net_used,
		MAX(CASE WHEN m.movement_type='OUT' THEN m.created_at END) AS last_issued_at
		FROM stock_movements m
		JOIN departments d ON d.id = m.department_id
		JOIN products p ON p.id = m.product_id
		WHERE ` + whereSQL + `
		GROUP BY d.id, d.name, p.id, p.sku, p.name
		ORDER BY net_used DESC, p.name ASC LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), limit, offset)
	var rows []UsageRow
	if err := s.db.WithContext(ctx).Raw(sql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []UsageRow{}
	}
	return rows, total, nil
}
