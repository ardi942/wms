package inventory

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/httpx"
)

type MovementInput struct {
	ProductID       uuid.UUID
	MovementType    string
	QuantityDelta   int
	ReferenceType   string
	ReferenceID     uuid.UUID
	ReferenceNumber string
	DepartmentID    *uuid.UUID
	ActorID         uuid.UUID
	Supplier        *string
	Notes           *string
	Reason          *string
	MinStock        int
}

// ApplyMovement inserts a movement and updates the balance atomically. The
// caller must invoke it inside a transaction. The balance row is locked for
// the remainder of that transaction, so callers processing multiple products
// must invoke it in ascending product_id order.
func (r *Repository) ApplyMovement(tx *gorm.DB, in MovementInput) (*Movement, bool, error) {
	if in.QuantityDelta == 0 {
		return nil, false, httpx.Validation("validation failed", map[string]string{"quantity": "must not be zero"})
	}
	if err := tx.Exec(
		`INSERT INTO inventory_balances (product_id, warehouse_id, quantity)
		 VALUES (?, ?, 0) ON CONFLICT (product_id, warehouse_id) DO NOTHING`,
		in.ProductID, r.warehouseID,
	).Error; err != nil {
		return nil, false, err
	}
	var old int
	if err := tx.Raw(
		`SELECT quantity FROM inventory_balances
		 WHERE product_id = ? AND warehouse_id = ? FOR UPDATE`,
		in.ProductID, r.warehouseID,
	).Scan(&old).Error; err != nil {
		return nil, false, err
	}
	newBalance := old + in.QuantityDelta
	if newBalance < 0 {
		return nil, false, httpx.Conflict("INSUFFICIENT_STOCK", "insufficient stock for this operation", map[string]any{
			"available": old,
			"requested": -in.QuantityDelta,
		})
	}
	now := time.Now().UTC()
	movement := &Movement{
		ID:              uuid.Must(uuid.NewV7()),
		ProductID:       in.ProductID,
		WarehouseID:     r.warehouseID,
		MovementType:    in.MovementType,
		QuantityDelta:   in.QuantityDelta,
		BalanceAfter:    newBalance,
		ReferenceType:   in.ReferenceType,
		ReferenceID:     in.ReferenceID,
		ReferenceNumber: in.ReferenceNumber,
		DepartmentID:    in.DepartmentID,
		ActorID:         in.ActorID,
		Supplier:        in.Supplier,
		Notes:           in.Notes,
		Reason:          in.Reason,
		CreatedAt:       now,
	}
	if err := tx.Create(movement).Error; err != nil {
		return nil, false, err
	}
	if err := tx.Exec(
		`UPDATE inventory_balances SET quantity = ?, updated_at = now()
		 WHERE product_id = ? AND warehouse_id = ?`,
		newBalance, in.ProductID, r.warehouseID,
	).Error; err != nil {
		return nil, false, err
	}
	lowStock := in.QuantityDelta < 0 && in.MinStock > 0 && old >= in.MinStock && newBalance < in.MinStock
	return movement, lowStock, nil
}

func (r *Repository) GetProductInfo(tx *gorm.DB, productID uuid.UUID) (*ProductInfo, error) {
	var p ProductInfo
	if err := tx.Raw(`SELECT id, sku, name, min_stock, is_active FROM products WHERE id = ?`, productID).Scan(&p).Error; err != nil {
		return nil, err
	}
	if p.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}

func (r *Repository) NetIssued(tx *gorm.DB, departmentID, productID uuid.UUID) (int, error) {
	var issued, returned int
	if err := tx.Raw(
		`SELECT COALESCE(SUM(-quantity_delta), 0) FROM stock_movements
		 WHERE department_id = ? AND product_id = ? AND movement_type = 'OUT'`,
		departmentID, productID,
	).Scan(&issued).Error; err != nil {
		return 0, err
	}
	if err := tx.Raw(
		`SELECT COALESCE(SUM(quantity_delta), 0) FROM stock_movements
		 WHERE department_id = ? AND product_id = ? AND movement_type = 'IN' AND reference_type = 'RETURN'`,
		departmentID, productID,
	).Scan(&returned).Error; err != nil {
		return 0, err
	}
	return issued - returned, nil
}

var seqPattern = regexp.MustCompile(`^[a-z_]+$`)

func NextDocumentNumber(tx *gorm.DB, sequence, prefix string) (string, error) {
	if !seqPattern.MatchString(sequence) {
		return "", fmt.Errorf("invalid sequence name")
	}
	var n int64
	if err := tx.Raw(fmt.Sprintf("SELECT nextval('%s')", sequence)).Scan(&n).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%06d", prefix, time.Now().UTC().Format("20060102"), n), nil
}

func SeqRequest() string    { return "request_number_seq" }
func SeqReturn() string     { return "return_number_seq" }
func SeqAdjustment() string { return "adjustment_number_seq" }
