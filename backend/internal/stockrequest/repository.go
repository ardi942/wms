package stockrequest

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(tx *gorm.DB, req *Request, items []Item) error {
	if err := tx.Create(req).Error; err != nil {
		return err
	}
	if len(items) > 0 {
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) GetForUpdate(tx *gorm.DB, id uuid.UUID) (*Request, error) {
	var req Request
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *Repository) GetItemsForUpdate(tx *gorm.DB, requestID uuid.UUID) ([]Item, error) {
	var items []Item
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", requestID).Order("product_id ASC").Find(&items).Error
	return items, err
}

func (r *Repository) UpdateItemIssued(tx *gorm.DB, itemID uuid.UUID, issued int) error {
	return tx.Model(&Item{}).Where("id = ?", itemID).
		Updates(map[string]any{"issued_qty": issued, "updated_at": time.Now().UTC()}).Error
}

func (r *Repository) UpdateHeader(tx *gorm.DB, id uuid.UUID, fields map[string]any) error {
	fields["updated_at"] = time.Now().UTC()
	return tx.Model(&Request{}).Where("id = ?", id).Updates(fields).Error
}

func (r *Repository) Totals(tx *gorm.DB, requestID uuid.UUID) (requested, issued int, err error) {
	err = tx.Raw(
		`SELECT COALESCE(SUM(requested_qty),0), COALESCE(SUM(issued_qty),0)
		 FROM stock_request_items WHERE request_id = ?`, requestID,
	).Row().Scan(&requested, &issued)
	return
}

type Filter struct {
	Status       string
	DepartmentID *uuid.UUID
	DateFrom     string
	DateTo       string
	Search       string
	Page         int
	PageSize     int
}

func (r *Repository) List(ctx context.Context, f Filter) ([]ListDTO, int64, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if f.Status != "" {
		where = append(where, "r.status = ?")
		args = append(args, f.Status)
	}
	if f.DepartmentID != nil {
		where = append(where, "r.department_id = ?")
		args = append(args, *f.DepartmentID)
	}
	if f.DateFrom != "" {
		where = append(where, "r.created_at >= ?::date")
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		where = append(where, "r.created_at < (?::date + interval '1 day')")
		args = append(args, f.DateTo)
	}
	if f.Search != "" {
		like := "%" + strings.TrimSpace(f.Search) + "%"
		where = append(where, "(r.request_number ILIKE ? OR r.purpose ILIKE ?)")
		args = append(args, like, like)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.WithContext(ctx).Raw(`SELECT count(*) FROM stock_requests r WHERE `+whereSQL, args...).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	sql := `SELECT r.id, r.request_number, r.status, r.purpose, r.created_at, r.updated_at,
		r.department_id, d.code AS dept_code, d.name AS dept_name,
		r.requested_by, u.full_name AS requester_name,
		(SELECT count(*) FROM stock_request_items i WHERE i.request_id = r.id) AS item_count,
		(SELECT COALESCE(SUM(i.requested_qty),0) FROM stock_request_items i WHERE i.request_id = r.id) AS total_requested,
		(SELECT COALESCE(SUM(i.issued_qty),0) FROM stock_request_items i WHERE i.request_id = r.id) AS total_issued
		FROM stock_requests r
		JOIN departments d ON d.id = r.department_id
		JOIN users u ON u.id = r.requested_by
		WHERE ` + whereSQL + `
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT ? OFFSET ?`
	listArgs := append(append([]any{}, args...), f.PageSize, (f.Page-1)*f.PageSize)

	type row struct {
		ID             uuid.UUID `gorm:"column:id"`
		RequestNumber  string    `gorm:"column:request_number"`
		Status         string    `gorm:"column:status"`
		Purpose        string    `gorm:"column:purpose"`
		CreatedAt      time.Time `gorm:"column:created_at"`
		UpdatedAt      time.Time `gorm:"column:updated_at"`
		DepartmentID   uuid.UUID `gorm:"column:department_id"`
		DeptCode       string    `gorm:"column:dept_code"`
		DeptName       string    `gorm:"column:dept_name"`
		RequestedBy    uuid.UUID `gorm:"column:requested_by"`
		RequesterName  string    `gorm:"column:requester_name"`
		ItemCount      int       `gorm:"column:item_count"`
		TotalRequested int       `gorm:"column:total_requested"`
		TotalIssued    int       `gorm:"column:total_issued"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Raw(sql, listArgs...).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]ListDTO, 0, len(rows))
	for _, x := range rows {
		out = append(out, ListDTO{
			ID: x.ID, RequestNumber: x.RequestNumber, Status: x.Status,
			Department:  DepartmentRef{ID: x.DepartmentID, Code: x.DeptCode, Name: x.DeptName},
			RequestedBy: ActorRef{ID: x.RequestedBy, FullName: x.RequesterName},
			Purpose:     x.Purpose, ItemCount: x.ItemCount,
			TotalRequested: x.TotalRequested, TotalIssued: x.TotalIssued,
			CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt,
		})
	}
	return out, total, nil
}

func (r *Repository) GetDetail(ctx context.Context, id uuid.UUID) (*DTO, error) {
	type header struct {
		ID              uuid.UUID  `gorm:"column:id"`
		RequestNumber   string     `gorm:"column:request_number"`
		Status          string     `gorm:"column:status"`
		Purpose         string     `gorm:"column:purpose"`
		RejectionReason *string    `gorm:"column:rejection_reason"`
		ProcessedAt     *time.Time `gorm:"column:processed_at"`
		CreatedAt       time.Time  `gorm:"column:created_at"`
		UpdatedAt       time.Time  `gorm:"column:updated_at"`
		DepartmentID    uuid.UUID  `gorm:"column:department_id"`
		DeptCode        string     `gorm:"column:dept_code"`
		DeptName        string     `gorm:"column:dept_name"`
		RequestedBy     uuid.UUID  `gorm:"column:requested_by"`
		RequesterName   string     `gorm:"column:requester_name"`
		ProcessedBy     *uuid.UUID `gorm:"column:processed_by"`
		ProcessorName   *string    `gorm:"column:processor_name"`
	}
	var h header
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.id, r.request_number, r.status, r.purpose, r.rejection_reason,
			r.processed_at, r.created_at, r.updated_at,
			r.department_id, d.code AS dept_code, d.name AS dept_name,
			r.requested_by, u.full_name AS requester_name,
			r.processed_by, pu.full_name AS processor_name
		FROM stock_requests r
		JOIN departments d ON d.id = r.department_id
		JOIN users u ON u.id = r.requested_by
		LEFT JOIN users pu ON pu.id = r.processed_by
		WHERE r.id = ?`, id).Scan(&h).Error
	if err != nil {
		return nil, err
	}
	if h.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	type itemRow struct {
		ID           uuid.UUID `gorm:"column:id"`
		ProductID    uuid.UUID `gorm:"column:product_id"`
		SKU          string    `gorm:"column:sku"`
		ProductName  string    `gorm:"column:product_name"`
		Unit         string    `gorm:"column:unit"`
		RequestedQty int       `gorm:"column:requested_qty"`
		IssuedQty    int       `gorm:"column:issued_qty"`
	}
	var items []itemRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT i.id, i.product_id, p.sku, p.name AS product_name, un.code AS unit,
			i.requested_qty, i.issued_qty
		FROM stock_request_items i
		JOIN products p ON p.id = i.product_id
		JOIN units un ON un.id = p.unit_id
		WHERE i.request_id = ?
		ORDER BY p.name ASC, i.id ASC`, id).Scan(&items).Error; err != nil {
		return nil, err
	}

	dto := &DTO{
		ID: h.ID, RequestNumber: h.RequestNumber, Status: h.Status, Purpose: h.Purpose,
		Department:      DepartmentRef{ID: h.DepartmentID, Code: h.DeptCode, Name: h.DeptName},
		RequestedBy:     ActorRef{ID: h.RequestedBy, FullName: h.RequesterName},
		RejectionReason: h.RejectionReason, ProcessedAt: h.ProcessedAt,
		CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt,
		Items: make([]ItemDTO, 0, len(items)),
	}
	if h.ProcessedBy != nil && h.ProcessorName != nil {
		dto.ProcessedBy = &ActorRef{ID: *h.ProcessedBy, FullName: *h.ProcessorName}
	}
	for _, it := range items {
		dto.Items = append(dto.Items, ItemDTO{
			ID: it.ID, Product: ProductRef{ID: it.ProductID, SKU: it.SKU, Name: it.ProductName, Unit: it.Unit},
			RequestedQty: it.RequestedQty, IssuedQty: it.IssuedQty,
			RemainingQty: it.RequestedQty - it.IssuedQty,
		})
	}
	return dto, nil
}
