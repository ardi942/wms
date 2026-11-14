package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wms/backend/internal/auth"
	"wms/backend/internal/inventory"
)

type seedProduct struct {
	SKU      string
	Name     string
	Category string
	Unit     string
	MinStock int
	Cost     float64
	Qty      int
}

var seedProducts = []seedProduct{
	{"IT-LAP-001", "Laptop 14\"", "IT Equipment", "PCS", 3, 950.00, 20},
	{"IT-MON-001", "Monitor 24\"", "IT Equipment", "PCS", 5, 180.00, 30},
	{"IT-KEY-001", "Keyboard USB", "IT Equipment", "PCS", 10, 25.00, 60},
	{"IT-MOU-001", "Mouse Wireless", "IT Equipment", "PCS", 10, 18.50, 45},
	{"OFF-PAP-001", "A4 Paper Ream", "Office Supplies", "BOX", 20, 4.75, 80},
	{"OFF-PEN-001", "Ballpoint Pen Box", "Office Supplies", "BOX", 15, 6.00, 50},
	{"CON-TON-001", "Printer Toner Black", "Consumables", "PCS", 4, 72.00, 2},
	{"CON-CLE-001", "Cleaning Fluid 1L", "Consumables", "LTR", 8, 3.20, 25},
	{"FUR-DSK-001", "Office Desk", "Furniture", "PCS", 2, 210.00, 5},
	{"FUR-CHR-001", "Office Chair", "Furniture", "PCS", 4, 95.00, 12},
}

type seedUser struct {
	Email        string
	FullName     string
	Password     string
	Role         string
	Department   string
}

var seedUsers = []seedUser{
	{"supervisor@wms.local", "Sam Supervisor", "Supervisor123!", "SUPERVISOR", ""},
	{"warehouse@wms.local", "Wendy Warehouse", "Warehouse123!", "WAREHOUSE_ADMIN", ""},
	{"staff@wms.local", "Ian IT Staff", "Staff123!", "DEPARTMENT_USER", "IT"},
	{"staff.fin@wms.local", "Fiona Finance", "Staff123!", "DEPARTMENT_USER", "FIN"},
}

// Run performs an idempotent demo seed. Existing rows with the same natural
// keys are left untouched.
func Run(db *gorm.DB, seedDemo bool) error {
	if !seedDemo {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		departments, err := upsertReference(tx)
		if err != nil {
			return err
		}
		if err := seedUsersInto(tx, departments); err != nil {
			return err
		}
		var adminID uuid.UUID
		if err := tx.Raw(`SELECT id FROM users WHERE email = ?`, "warehouse@wms.local").Row().Scan(&adminID); err != nil {
			return err
		}
		var staffID uuid.UUID
		if err := tx.Raw(`SELECT id FROM users WHERE email = ?`, "staff@wms.local").Row().Scan(&staffID); err != nil {
			return err
		}
		var warehouseID uuid.UUID
		if err := tx.Raw(`SELECT id FROM warehouses WHERE code = 'MAIN'`).Row().Scan(&warehouseID); err != nil {
			return err
		}
		if warehouseID == uuid.Nil || adminID == uuid.Nil || staffID == uuid.Nil {
			return fmt.Errorf("seed prerequisites missing")
		}
		inv := inventory.NewRepository(db, warehouseID)
		if err := seedProductsInto(tx, inv, adminID, departments); err != nil {
			return err
		}
		var requestCount int64
		if err := tx.Table("stock_requests").Count(&requestCount).Error; err != nil {
			return err
		}
		if requestCount == 0 {
			if err := seedDocuments(tx, inv, adminID, staffID, departments["IT"]); err != nil {
				return err
			}
		}
		return verifyLedger(tx, warehouseID)
	})
}

func upsertReference(tx *gorm.DB) (map[string]uuid.UUID, error) {
	departments := map[string]string{"IT": "Information Technology", "FIN": "Finance", "OPS": "Operations"}
	out := map[string]uuid.UUID{}
	for code, name := range departments {
		if err := tx.Exec(`INSERT INTO departments (code, name) VALUES (?, ?) ON CONFLICT (code) DO NOTHING`, code, name).Error; err != nil {
			return nil, err
		}
		var id uuid.UUID
		if err := tx.Raw(`SELECT id FROM departments WHERE code = ?`, code).Row().Scan(&id); err != nil {
			return nil, err
		}
		out[code] = id
	}
	categories := []string{"IT Equipment", "Office Supplies", "Consumables", "Furniture"}
	for _, name := range categories {
		if err := tx.Exec(`INSERT INTO categories (name) VALUES (?) ON CONFLICT (name) DO NOTHING`, name).Error; err != nil {
			return nil, err
		}
	}
	units := [][2]string{{"PCS", "Pieces"}, {"BOX", "Box"}, {"KG", "Kilogram"}, {"LTR", "Liter"}}
	for _, u := range units {
		if err := tx.Exec(`INSERT INTO units (code, name) VALUES (?, ?) ON CONFLICT (code) DO NOTHING`, u[0], u[1]).Error; err != nil {
			return nil, err
		}
	}
	if err := tx.Exec(`INSERT INTO warehouses (code, name) VALUES ('MAIN', 'Main Warehouse') ON CONFLICT (code) DO NOTHING`).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func seedUsersInto(tx *gorm.DB, departments map[string]uuid.UUID) error {
	now := time.Now().UTC()
	for _, u := range seedUsers {
		var exists int64
		if err := tx.Table("users").Where("email = ?", u.Email).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		hash, err := auth.HashPassword(u.Password)
		if err != nil {
			return err
		}
		var deptID *uuid.UUID
		if u.Department != "" {
			id := departments[u.Department]
			deptID = &id
		}
		if err := tx.Exec(`INSERT INTO users (id, email, password_hash, full_name, role, department_id, is_active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, true, ?, ?)`,
			uuid.Must(uuid.NewV7()), u.Email, hash, u.FullName, u.Role, deptID, now, now).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedProductsInto(tx *gorm.DB, inv *inventory.Repository, adminID uuid.UUID, departments map[string]uuid.UUID) error {
	now := time.Now().UTC()
	for _, p := range seedProducts {
		var productID uuid.UUID
		if err := tx.Raw(`SELECT id FROM products WHERE sku = ?`, p.SKU).Row().Scan(&productID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if productID == uuid.Nil {
			var categoryID, unitID uuid.UUID
			if err := tx.Raw(`SELECT id FROM categories WHERE name = ?`, p.Category).Row().Scan(&categoryID); err != nil {
				return err
			}
			if err := tx.Raw(`SELECT id FROM units WHERE code = ?`, p.Unit).Row().Scan(&unitID); err != nil {
				return err
			}
			productID = uuid.Must(uuid.NewV7())
			if err := tx.Exec(`INSERT INTO products (id, sku, name, description, category_id, unit_id, min_stock, unit_cost, is_active, created_at, updated_at)
				VALUES (?, ?, ?, NULL, ?, ?, ?, ?, true, ?, ?)`,
				productID, p.SKU, p.Name, categoryID, unitID, p.MinStock, p.Cost, now, now).Error; err != nil {
				return err
			}
			ref := "SEED-INIT-" + p.SKU
			supplier := "Initial Stock"
			_, _, err := inv.ApplyMovement(tx, inventory.MovementInput{
				ProductID: productID, MovementType: inventory.TypeIn, QuantityDelta: p.Qty,
				ReferenceType: inventory.RefReceipt, ReferenceID: uuid.Must(uuid.NewV7()),
				ReferenceNumber: ref, ActorID: adminID, Supplier: &supplier, MinStock: p.MinStock,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func productIDBySKU(tx *gorm.DB, sku string) (uuid.UUID, int, error) {
	var id uuid.UUID
	var minStock int
	if err := tx.Raw(`SELECT id, min_stock FROM products WHERE sku = ?`, sku).Row().Scan(&id, &minStock); err != nil {
		return uuid.Nil, 0, err
	}
	return id, minStock, nil
}

func seedDocuments(tx *gorm.DB, inv *inventory.Repository, adminID, staffID, itDept uuid.UUID) error {
	now := time.Now().UTC()
	keyboard, kbMin, err := productIDBySKU(tx, "IT-KEY-001")
	if err != nil {
		return err
	}
	mouse, mouseMin, err := productIDBySKU(tx, "IT-MOU-001")
	if err != nil {
		return err
	}
	monitor, _, err := productIDBySKU(tx, "IT-MON-001")
	if err != nil {
		return err
	}
	chair, chairMin, err := productIDBySKU(tx, "FUR-CHR-001")
	if err != nil {
		return err
	}

	// Fulfilled request with OUT movements (keyboard x2, mouse x1).
	reqNO, err := inventory.NextDocumentNumber(tx, inventory.SeqRequest(), "REQ")
	if err != nil {
		return err
	}
	reqID := uuid.Must(uuid.NewV7())
	if err := tx.Exec(`INSERT INTO stock_requests (id, request_number, department_id, requested_by, status, purpose, processed_by, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'FULFILLED', ?, ?, ?, ?, ?)`,
		reqID, reqNO, itDept, staffID, "Initial IT equipment order", adminID, now, now, now).Error; err != nil {
		return err
	}
	if err := tx.Exec(`INSERT INTO stock_request_items (id, request_id, product_id, requested_qty, issued_qty, created_at, updated_at)
		VALUES (?, ?, ?, 2, 2, ?, ?), (?, ?, ?, 1, 1, ?, ?)`,
		uuid.Must(uuid.NewV7()), reqID, keyboard, now, now,
		uuid.Must(uuid.NewV7()), reqID, mouse, now, now).Error; err != nil {
		return err
	}
	lines := []struct {
		pid uuid.UUID
		qty int
		min int
	}{{keyboard, 2, kbMin}, {mouse, 1, mouseMin}}
	for _, l := range lines {
		if _, _, err := inv.ApplyMovement(tx, inventory.MovementInput{
			ProductID: l.pid, MovementType: inventory.TypeOut, QuantityDelta: -l.qty,
			ReferenceType: inventory.RefRequest, ReferenceID: reqID, ReferenceNumber: reqNO,
			DepartmentID: &itDept, ActorID: adminID, Notes: ptr("Initial IT equipment order"), MinStock: l.min,
		}); err != nil {
			return err
		}
	}

	pendingNO, err := inventory.NextDocumentNumber(tx, inventory.SeqRequest(), "REQ")
	if err != nil {
		return err
	}
	pendingID := uuid.Must(uuid.NewV7())
	if err := tx.Exec(`INSERT INTO stock_requests (id, request_number, department_id, requested_by, status, purpose, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'PENDING', ?, ?, ?)`,
		pendingID, pendingNO, itDept, staffID, "Second monitor for analyst", now, now).Error; err != nil {
		return err
	}
	if err := tx.Exec(`INSERT INTO stock_request_items (id, request_id, product_id, requested_qty, issued_qty, created_at, updated_at)
		VALUES (?, ?, ?, 1, 0, ?, ?)`,
		uuid.Must(uuid.NewV7()), pendingID, monitor, now, now).Error; err != nil {
		return err
	}

	rejectedNO, err := inventory.NextDocumentNumber(tx, inventory.SeqRequest(), "REQ")
	if err != nil {
		return err
	}
	rejectedID := uuid.Must(uuid.NewV7())
	if err := tx.Exec(`INSERT INTO stock_requests (id, request_number, department_id, requested_by, status, purpose, rejection_reason, processed_by, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'REJECTED', ?, ?, ?, ?, ?, ?)`,
		rejectedID, rejectedNO, itDept, staffID, "Extra office chair", "Not approved for this quarter", adminID, now, now, now).Error; err != nil {
		return err
	}
	if err := tx.Exec(`INSERT INTO stock_request_items (id, request_id, product_id, requested_qty, issued_qty, created_at, updated_at)
		VALUES (?, ?, ?, 1, 0, ?, ?)`,
		uuid.Must(uuid.NewV7()), rejectedID, chair, now, now).Error; err != nil {
		return err
	}

	// Confirmed return (keyboard x1 back to warehouse).
	retNO, err := inventory.NextDocumentNumber(tx, inventory.SeqReturn(), "RET")
	if err != nil {
		return err
	}
	retID := uuid.Must(uuid.NewV7())
	if err := tx.Exec(`INSERT INTO stock_returns (id, return_number, department_id, requested_by, status, reason, processed_by, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'CONFIRMED', ?, ?, ?, ?, ?)`,
		retID, retNO, itDept, staffID, "Unused keyboard returned", adminID, now, now, now).Error; err != nil {
		return err
	}
	if err := tx.Exec(`INSERT INTO stock_return_items (id, return_id, product_id, requested_qty, accepted_qty, created_at, updated_at)
		VALUES (?, ?, ?, 1, 1, ?, ?)`,
		uuid.Must(uuid.NewV7()), retID, keyboard, now, now).Error; err != nil {
		return err
	}
	if _, _, err := inv.ApplyMovement(tx, inventory.MovementInput{
		ProductID: keyboard, MovementType: inventory.TypeIn, QuantityDelta: 1,
		ReferenceType: inventory.RefReturn, ReferenceID: retID, ReferenceNumber: retNO,
		DepartmentID: &itDept, ActorID: adminID, Reason: ptr("Unused keyboard returned"), MinStock: kbMin,
	}); err != nil {
		return err
	}

	// Pending return.
	pendRetNO, err := inventory.NextDocumentNumber(tx, inventory.SeqReturn(), "RET")
	if err != nil {
		return err
	}
	pendRetID := uuid.Must(uuid.NewV7())
	if err := tx.Exec(`INSERT INTO stock_returns (id, return_number, department_id, requested_by, status, reason, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'PENDING', ?, ?, ?)`,
		pendRetID, pendRetNO, itDept, staffID, "Spare mouse no longer needed", now, now).Error; err != nil {
		return err
	}
	if err := tx.Exec(`INSERT INTO stock_return_items (id, return_id, product_id, requested_qty, accepted_qty, created_at, updated_at)
		VALUES (?, ?, ?, 1, 0, ?, ?)`,
		uuid.Must(uuid.NewV7()), pendRetID, mouse, now, now).Error; err != nil {
		return err
	}

	// Adjustment.
	adjNO, err := inventory.NextDocumentNumber(tx, inventory.SeqAdjustment(), "ADJ")
	if err != nil {
		return err
	}
	if _, _, err := inv.ApplyMovement(tx, inventory.MovementInput{
		ProductID: chair, MovementType: inventory.TypeAdjustment, QuantityDelta: 1,
		ReferenceType: inventory.RefAdjustment, ReferenceID: uuid.Must(uuid.NewV7()),
		ReferenceNumber: adjNO, ActorID: adminID, Reason: ptr("Physical count correction"), MinStock: chairMin,
	}); err != nil {
		return err
	}
	return nil
}

func verifyLedger(tx *gorm.DB, warehouseID uuid.UUID) error {
	var mismatches int
	err := tx.Raw(`
		SELECT count(*) FROM (
			SELECT p.id,
				COALESCE(SUM(m.quantity_delta),0) AS ledger_qty,
				COALESCE(MAX(b.quantity),0) AS balance_qty
			FROM products p
			LEFT JOIN stock_movements m ON m.product_id = p.id AND m.warehouse_id = ?
			LEFT JOIN inventory_balances b ON b.product_id = p.id AND b.warehouse_id = ?
			GROUP BY p.id
			HAVING COALESCE(SUM(m.quantity_delta),0) <> COALESCE(MAX(b.quantity),0)
		) x`, warehouseID, warehouseID).Scan(&mismatches).Error
	if err != nil {
		return err
	}
	if mismatches > 0 {
		return fmt.Errorf("seed ledger verification failed: %d product(s) out of balance", mismatches)
	}
	return nil
}

func ptr(v string) *string { return &v }
