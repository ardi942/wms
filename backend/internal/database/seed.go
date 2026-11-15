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
	SKU         string
	Name        string
	Description string
	Category    string
	Unit        string
	MinStock    int
	Cost        float64
	Qty         int
}

var seedProducts = []seedProduct{
	{"OFF-PAP-001", "A4 Copy Paper (Ream)", "80 gsm white paper for daily printing and photocopying.", "Office Supplies", "REAM", 10, 55000, 80},
	{"OFF-PEN-001", "Ballpoint Pen (Box of 12)", "Blue ink ballpoint pens for general office use.", "Office Supplies", "BOX", 10, 60000, 50},
	{"OFF-NBK-001", "Notebook A5 (Pack of 5)", "Lined A5 notebooks for meetings and daily notes.", "Office Supplies", "PACK", 8, 45000, 40},
	{"OFF-FLD-001", "Document Folder (Box of 50)", "Buff paper folders for filing and archiving.", "Office Supplies", "BOX", 5, 75000, 25},
	{"OFF-STP-001", "Stapler & Staples Set", "Medium-duty stapler with a box of staples.", "Office Supplies", "PCS", 4, 85000, 12},
	{"OFF-MRK-001", "Whiteboard Marker (Box of 12)", "Assorted colours for whiteboards and flipcharts.", "Office Supplies", "BOX", 6, 90000, 18},
	{"IT-LAP-001", "Laptop 14-inch", "Standard business laptop for office and remote work.", "IT Equipment", "PCS", 2, 12500000, 20},
	{"IT-MON-001", "Monitor 24-inch", "Full HD monitor for desk work and presentations.", "IT Equipment", "PCS", 3, 2750000, 30},
	{"IT-KEY-001", "USB Keyboard", "Standard USB keyboard for desktop workstations.", "IT Equipment", "PCS", 5, 250000, 60},
	{"IT-MOU-001", "Wireless Mouse", "Wireless mouse for laptops and desktops.", "IT Equipment", "PCS", 5, 150000, 45},
	{"IT-PRN-001", "Laser Printer", "Monochrome office laser printer.", "IT Equipment", "PCS", 1, 4200000, 4},
	{"CON-TON-001", "Printer Toner (Black)", "Black toner cartridge for office laser printers.", "Consumables", "PCS", 3, 1250000, 2},
	{"CON-CLE-001", "Floor Cleaner 1L", "Concentrated floor cleaner for daily cleaning.", "Consumables", "LTR", 8, 35000, 25},
	{"CON-TIS-001", "Facial Tissue (Pack of 24)", "Soft tissue packs for desks and meeting rooms.", "Consumables", "PACK", 6, 60000, 30},
	{"CON-GAL-001", "Drinking Water Gallon 19L", "Refillable drinking water for pantry dispensers.", "Consumables", "GAL", 10, 22000, 40},
	{"CON-COF-001", "Coffee Sachets (Box of 25)", "Instant coffee sachets for pantry and guests.", "Consumables", "BOX", 10, 45000, 35},
	{"FUR-DSK-001", "Office Desk", "Standard 120 cm office desk.", "Furniture", "PCS", 2, 2100000, 5},
	{"FUR-CHR-001", "Ergonomic Office Chair", "Adjustable office chair with back support.", "Furniture", "PCS", 3, 1450000, 12},
	{"FUR-CAB-001", "Steel Filing Cabinet", "Three-drawer locking cabinet for documents.", "Furniture", "PCS", 2, 1850000, 3},
	{"FAC-LMP-001", "LED Tube Lamp 18W", "Energy-saving 120 cm tube for office ceilings.", "Facility & Safety", "PCS", 10, 65000, 24},
	{"FAC-EXT-001", "Extension Cord 5m", "Five-metre extension cord with four outlets.", "Facility & Safety", "PCS", 5, 120000, 8},
	{"FAC-FAK-001", "First Aid Kit", "Basic first aid kit for office and warehouse areas.", "Facility & Safety", "PCS", 2, 350000, 1},
	{"FAC-FEX-001", "Fire Extinguisher 3kg", "Dry powder extinguisher for office and warehouse floors.", "Facility & Safety", "PCS", 1, 450000, 3},
}

type seedUser struct {
	Email      string
	FullName   string
	Password   string
	Role       string
	Department string
}

var seedUsers = []seedUser{
	{"supervisor@wms.local", "Sari Wulandari", "Supervisor123!", "SUPERVISOR", ""},
	{"warehouse@wms.local", "Wahyu Pratama", "Warehouse123!", "WAREHOUSE_ADMIN", ""},
	{"staff@wms.local", "Budi Santoso", "Staff123!", "DEPARTMENT_USER", "IT"},
	{"staff.fin@wms.local", "Siti Rahmawati", "Staff123!", "DEPARTMENT_USER", "FIN"},
	{"staff.ops@wms.local", "Andi Saputra", "Staff123!", "DEPARTMENT_USER", "OPS"},
	{"staff.ga@wms.local", "Dewi Lestari", "Staff123!", "DEPARTMENT_USER", "GA"},
}

type seedProductRow struct {
	ID       uuid.UUID
	MinStock int
}

var categorySuppliers = map[string]string{
	"Office Supplies":   "PT Sinar Jaya Office Supplies",
	"IT Equipment":      "CV Mitra Teknologi",
	"Consumables":       "PT Bersih Sejahtera",
	"Furniture":         "PT Karya Furnitur Nusantara",
	"Facility & Safety": "Toko Sumber Makmur",
}

// Run performs an idempotent demo seed. Existing rows with the same natural
// keys are updated in place; documents are only seeded when no requests exist.
func Run(db *gorm.DB, seedDemo bool) error {
	if !seedDemo {
		return nil
	}
	now := time.Now().UTC()
	return db.Transaction(func(tx *gorm.DB) error {
		departments, err := upsertReference(tx)
		if err != nil {
			return err
		}
		if err := seedUsersInto(tx, departments); err != nil {
			return err
		}
		var adminID uuid.UUID
		if adminID, err = lookupID(tx, `SELECT id FROM users WHERE email = ?`, "warehouse@wms.local"); err != nil {
			return err
		}
		warehouseID, err := lookupID(tx, `SELECT id FROM warehouses WHERE code = 'MAIN'`)
		if err != nil {
			return err
		}
		if warehouseID == uuid.Nil || adminID == uuid.Nil {
			return fmt.Errorf("seed prerequisites missing")
		}
		deptUsers := map[string]uuid.UUID{}
		for dept, email := range map[string]string{
			"IT":  "staff@wms.local",
			"FIN": "staff.fin@wms.local",
			"OPS": "staff.ops@wms.local",
			"GA":  "staff.ga@wms.local",
		} {
			var id uuid.UUID
			var err error
			if id, err = lookupID(tx, `SELECT id FROM users WHERE email = ?`, email); err != nil {
				return err
			}
			if id == uuid.Nil {
				return fmt.Errorf("seed user %s missing", email)
			}
			deptUsers[dept] = id
		}
		inv := inventory.NewRepository(tx, warehouseID)
		products, err := seedProductsInto(tx, inv, adminID, now)
		if err != nil {
			return err
		}
		var requestCount int64
		if err := tx.Table("stock_requests").Count(&requestCount).Error; err != nil {
			return err
		}
		if requestCount == 0 {
			if err := seedDocuments(tx, inv, adminID, departments, deptUsers, products, now); err != nil {
				return err
			}
		}
		return verifyLedger(tx, warehouseID)
	})
}

// Reset removes all demo rows and restarts document sequences. TRUNCATE is used
// because the immutable movement trigger rejects DELETE on stock_movements. The
// MAIN warehouse row is reinserted with its previous ID so running API
// processes that cached the warehouse ID keep working.
func Reset(db *gorm.DB) error {
	warehouseID, err := lookupID(db, `SELECT id FROM warehouses WHERE code = 'MAIN'`)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`TRUNCATE TABLE
			stock_return_items, stock_returns,
			stock_request_items, stock_requests,
			stock_movements, inventory_balances,
			notifications, outbox_events, processed_events, idempotency_keys,
			audit_logs,
			users, products, warehouses, units, categories, departments
			RESTART IDENTITY CASCADE`).Error; err != nil {
			return err
		}
		if warehouseID != uuid.Nil {
			if err := tx.Exec(`INSERT INTO warehouses (id, code, name) VALUES (?, 'MAIN', 'Main Warehouse')`, warehouseID).Error; err != nil {
				return err
			}
		}
		for _, seq := range []string{"request_number_seq", "return_number_seq", "adjustment_number_seq"} {
			if err := tx.Exec(fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH 1", seq)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func upsertReference(tx *gorm.DB) (map[string]uuid.UUID, error) {
	departments := map[string]string{
		"IT":  "Information Technology",
		"FIN": "Finance",
		"OPS": "Operations",
		"GA":  "General Affairs",
	}
	out := map[string]uuid.UUID{}
	for code, name := range departments {
		if err := tx.Exec(`INSERT INTO departments (code, name) VALUES (?, ?) ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name`, code, name).Error; err != nil {
			return nil, err
		}
		var id uuid.UUID
		var err error
		if id, err = lookupID(tx, `SELECT id FROM departments WHERE code = ?`, code); err != nil {
			return nil, err
		}
		out[code] = id
	}
	categories := []string{"Office Supplies", "IT Equipment", "Consumables", "Furniture", "Facility & Safety"}
	for _, name := range categories {
		if err := tx.Exec(`INSERT INTO categories (name) VALUES (?) ON CONFLICT (name) DO NOTHING`, name).Error; err != nil {
			return nil, err
		}
	}
	units := [][2]string{
		{"PCS", "Pieces"}, {"BOX", "Box"}, {"REAM", "Ream"}, {"PACK", "Pack"},
		{"KG", "Kilogram"}, {"LTR", "Liter"}, {"GAL", "Gallon"},
	}
	for _, u := range units {
		if err := tx.Exec(`INSERT INTO units (code, name) VALUES (?, ?) ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name`, u[0], u[1]).Error; err != nil {
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
			VALUES (?, ?, ?, ?, ?, ?, true, ?, ?)
			ON CONFLICT (email) DO UPDATE SET
				full_name = EXCLUDED.full_name,
				role = EXCLUDED.role,
				department_id = EXCLUDED.department_id,
				updated_at = EXCLUDED.updated_at`,
			uuid.Must(uuid.NewV7()), u.Email, hash, u.FullName, u.Role, deptID, now, now).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedProductsInto(tx *gorm.DB, inv *inventory.Repository, adminID uuid.UUID, now time.Time) (map[string]seedProductRow, error) {
	out := map[string]seedProductRow{}
	for i, p := range seedProducts {
		var categoryID, unitID uuid.UUID
		var err error
		if categoryID, err = lookupID(tx, `SELECT id FROM categories WHERE name = ?`, p.Category); err != nil {
			return nil, err
		}
		if unitID, err = lookupID(tx, `SELECT id FROM units WHERE code = ?`, p.Unit); err != nil {
			return nil, err
		}
		if categoryID == uuid.Nil || unitID == uuid.Nil {
			return nil, fmt.Errorf("seed reference missing for %s", p.SKU)
		}
		productID, err := lookupID(tx, `SELECT id FROM products WHERE sku = ?`, p.SKU)
		if err != nil {
			return nil, err
		}
		if productID == uuid.Nil {
			productID = uuid.Must(uuid.NewV7())
			if err := tx.Exec(`INSERT INTO products (id, sku, name, description, category_id, unit_id, min_stock, unit_cost, is_active, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, true, ?, ?)`,
				productID, p.SKU, p.Name, p.Description, categoryID, unitID, p.MinStock, p.Cost, now, now).Error; err != nil {
				return nil, err
			}
		} else {
			if err := tx.Exec(`UPDATE products SET name = ?, description = ?, category_id = ?, unit_id = ?, min_stock = ?, unit_cost = ?, updated_at = ? WHERE id = ?`,
				p.Name, p.Description, categoryID, unitID, p.MinStock, p.Cost, now, productID).Error; err != nil {
				return nil, err
			}
		}
		out[p.SKU] = seedProductRow{ID: productID, MinStock: p.MinStock}

		var movementCount int64
		if err := tx.Table("stock_movements").Where("product_id = ?", productID).Count(&movementCount).Error; err != nil {
			return nil, err
		}
		if movementCount > 0 {
			continue
		}
		occurred := dayAt(now, 30-(i%3), 8+(i%5), (i*7)%60)
		ref := fmt.Sprintf("PO/2026/%04d", i+1)
		supplier := categorySuppliers[p.Category]
		if _, _, err := inv.ApplyMovement(tx, inventory.MovementInput{
			ProductID:       productID,
			MovementType:    inventory.TypeIn,
			QuantityDelta:   p.Qty,
			ReferenceType:   inventory.RefReceipt,
			ReferenceID:     uuid.Must(uuid.NewV7()),
			ReferenceNumber: ref,
			ActorID:         adminID,
			Supplier:        &supplier,
			Notes:           ptr("Initial stock for " + p.Category),
			MinStock:        p.MinStock,
			OccurredAt:      occurred,
		}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

type seedLine struct {
	SKU       string
	Requested int
	Issued    int
}

func seedDocuments(tx *gorm.DB, inv *inventory.Repository, adminID uuid.UUID, departments, deptUsers map[string]uuid.UUID, products map[string]seedProductRow, now time.Time) error {
	if err := seedRequest(tx, inv, products, adminID, deptUsers["IT"], departments["IT"],
		dayAt(now, 26, 9, 0), "FULFILLED", "Peripheral setup for two new analysts", "",
		[]seedLine{{"IT-KEY-001", 2, 2}, {"IT-MOU-001", 1, 1}}); err != nil {
		return err
	}
	if err := seedRequest(tx, inv, products, adminID, deptUsers["FIN"], departments["FIN"],
		dayAt(now, 23, 9, 15), "FULFILLED", "Monthly office supplies for Finance", "",
		[]seedLine{{"OFF-PAP-001", 3, 3}, {"OFF-PEN-001", 2, 2}}); err != nil {
		return err
	}
	if err := seedRequest(tx, inv, products, adminID, deptUsers["OPS"], departments["OPS"],
		dayAt(now, 20, 8, 45), "PARTIALLY_FULFILLED", "Monthly cleaning supplies for the operations floor", "",
		[]seedLine{{"CON-CLE-001", 8, 5}, {"CON-TIS-001", 4, 2}}); err != nil {
		return err
	}
	if err := seedReturn(tx, inv, products, adminID, deptUsers["IT"], departments["IT"],
		dayAt(now, 17, 10, 0), "CONFIRMED", "Spare keyboard returned after desk move",
		[]seedLine{{"IT-KEY-001", 1, 1}}); err != nil {
		return err
	}
	if err := seedRequest(tx, inv, products, adminID, deptUsers["GA"], departments["GA"],
		dayAt(now, 15, 9, 30), "FULFILLED", "Meeting room lighting repair", "",
		[]seedLine{{"FAC-LMP-001", 6, 6}, {"FAC-EXT-001", 2, 2}}); err != nil {
		return err
	}
	if err := seedRequest(tx, inv, products, adminID, deptUsers["FIN"], departments["FIN"],
		dayAt(now, 12, 9, 0), "REJECTED", "Additional filing cabinet for Finance archive", "Not in this quarter's budget",
		[]seedLine{{"FUR-CAB-001", 1, 0}}); err != nil {
		return err
	}
	if err := seedAdjustment(tx, inv, products, adminID, dayAt(now, 10, 14, 0),
		"CON-TON-001", -1, "Damaged toner cartridge removed from stock"); err != nil {
		return err
	}
	if err := seedReturn(tx, inv, products, adminID, deptUsers["OPS"], departments["OPS"],
		dayAt(now, 8, 9, 0), "PENDING", "Unused tissue from last order",
		[]seedLine{{"CON-TIS-001", 1, 0}}); err != nil {
		return err
	}
	if err := seedRequest(tx, inv, products, adminID, deptUsers["IT"], departments["IT"],
		dayAt(now, 6, 9, 0), "PENDING", "Second monitor for new analyst", "",
		[]seedLine{{"IT-MON-001", 1, 0}}); err != nil {
		return err
	}
	if err := seedRequest(tx, inv, products, adminID, deptUsers["FIN"], departments["FIN"],
		dayAt(now, 4, 9, 0), "FULFILLED", "Pantry restock for Finance floor", "",
		[]seedLine{{"CON-COF-001", 1, 1}, {"CON-GAL-001", 2, 2}}); err != nil {
		return err
	}
	if err := seedAdjustment(tx, inv, products, adminID, dayAt(now, 2, 15, 0),
		"FUR-CHR-001", 1, "Stocktake correction after physical count"); err != nil {
		return err
	}
	return seedRequest(tx, inv, products, adminID, deptUsers["GA"], departments["GA"],
		dayAt(now, 1, 8, 0), "CANCELLED", "Duplicate request; item already ordered", "",
		[]seedLine{{"FAC-FEX-001", 1, 0}})
}

func seedRequest(tx *gorm.DB, inv *inventory.Repository, products map[string]seedProductRow, adminID, requestedBy, deptID uuid.UUID, at time.Time, status, purpose, rejection string, lines []seedLine) error {
	number, err := inventory.NextDocumentNumberAt(tx, inventory.SeqRequest(), "REQ", at)
	if err != nil {
		return err
	}
	id := uuid.Must(uuid.NewV7())
	var processedBy *uuid.UUID
	var processedAt *time.Time
	if status != "PENDING" && status != "CANCELLED" {
		processedBy = &adminID
		p := at.Add(3 * time.Hour)
		processedAt = &p
	}
	var rejectionReason *string
	if rejection != "" {
		rejectionReason = &rejection
	}
	if err := tx.Exec(`INSERT INTO stock_requests
		(id, request_number, department_id, requested_by, status, purpose, rejection_reason, processed_by, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, number, deptID, requestedBy, status, purpose, rejectionReason, processedBy, processedAt, at, at).Error; err != nil {
		return err
	}
	for _, line := range lines {
		product, ok := products[line.SKU]
		if !ok {
			return fmt.Errorf("seed product %s missing", line.SKU)
		}
		if err := tx.Exec(`INSERT INTO stock_request_items (id, request_id, product_id, requested_qty, issued_qty, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			uuid.Must(uuid.NewV7()), id, product.ID, line.Requested, line.Issued, at, at).Error; err != nil {
			return err
		}
	}
	if status != "FULFILLED" && status != "PARTIALLY_FULFILLED" {
		return nil
	}
	issuedAt := at.Add(3 * time.Hour)
	for _, line := range lines {
		if line.Issued == 0 {
			continue
		}
		product := products[line.SKU]
		if _, _, err := inv.ApplyMovement(tx, inventory.MovementInput{
			ProductID:       product.ID,
			MovementType:    inventory.TypeOut,
			QuantityDelta:   -line.Issued,
			ReferenceType:   inventory.RefRequest,
			ReferenceID:     id,
			ReferenceNumber: number,
			DepartmentID:    &deptID,
			ActorID:         adminID,
			Notes:           ptr(purpose),
			MinStock:        product.MinStock,
			OccurredAt:      issuedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedReturn(tx *gorm.DB, inv *inventory.Repository, products map[string]seedProductRow, adminID, requestedBy, deptID uuid.UUID, at time.Time, status, reason string, lines []seedLine) error {
	number, err := inventory.NextDocumentNumberAt(tx, inventory.SeqReturn(), "RET", at)
	if err != nil {
		return err
	}
	id := uuid.Must(uuid.NewV7())
	var processedBy *uuid.UUID
	var processedAt *time.Time
	if status == "CONFIRMED" || status == "REJECTED" {
		processedBy = &adminID
		p := at.Add(2 * time.Hour)
		processedAt = &p
	}
	if err := tx.Exec(`INSERT INTO stock_returns
		(id, return_number, department_id, requested_by, status, reason, processed_by, processed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, number, deptID, requestedBy, status, reason, processedBy, processedAt, at, at).Error; err != nil {
		return err
	}
	for _, line := range lines {
		product, ok := products[line.SKU]
		if !ok {
			return fmt.Errorf("seed product %s missing", line.SKU)
		}
		if err := tx.Exec(`INSERT INTO stock_return_items (id, return_id, product_id, requested_qty, accepted_qty, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			uuid.Must(uuid.NewV7()), id, product.ID, line.Requested, line.Issued, at, at).Error; err != nil {
			return err
		}
	}
	if status != "CONFIRMED" {
		return nil
	}
	acceptedAt := at.Add(2 * time.Hour)
	for _, line := range lines {
		if line.Issued == 0 {
			continue
		}
		product := products[line.SKU]
		if _, _, err := inv.ApplyMovement(tx, inventory.MovementInput{
			ProductID:       product.ID,
			MovementType:    inventory.TypeIn,
			QuantityDelta:   line.Issued,
			ReferenceType:   inventory.RefReturn,
			ReferenceID:     id,
			ReferenceNumber: number,
			DepartmentID:    &deptID,
			ActorID:         adminID,
			Reason:          ptr(reason),
			MinStock:        product.MinStock,
			OccurredAt:      acceptedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func seedAdjustment(tx *gorm.DB, inv *inventory.Repository, products map[string]seedProductRow, adminID uuid.UUID, at time.Time, sku string, delta int, reason string) error {
	number, err := inventory.NextDocumentNumberAt(tx, inventory.SeqAdjustment(), "ADJ", at)
	if err != nil {
		return err
	}
	product, ok := products[sku]
	if !ok {
		return fmt.Errorf("seed product %s missing", sku)
	}
	_, _, err = inv.ApplyMovement(tx, inventory.MovementInput{
		ProductID:       product.ID,
		MovementType:    inventory.TypeAdjustment,
		QuantityDelta:   delta,
		ReferenceType:   inventory.RefAdjustment,
		ReferenceID:     uuid.Must(uuid.NewV7()),
		ReferenceNumber: number,
		ActorID:         adminID,
		Reason:          ptr(reason),
		MinStock:        product.MinStock,
		OccurredAt:      at,
	})
	return err
}

func dayAt(now time.Time, daysAgo, hour, minute int) time.Time {
	d := now.AddDate(0, 0, -daysAgo)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, 0, 0, time.UTC)
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
	var lowStock int
	if err := tx.Raw(`
		SELECT count(*) FROM products p
		JOIN inventory_balances b ON b.product_id = p.id
		WHERE b.warehouse_id = ? AND b.quantity < p.min_stock`, warehouseID).Scan(&lowStock).Error; err != nil {
		return err
	}
	if lowStock < 2 {
		return fmt.Errorf("seed verification failed: expected at least 2 low-stock products, got %d", lowStock)
	}
	var badReturns int
	if err := tx.Raw(`
		SELECT count(*)
		FROM stock_return_items ri
		JOIN stock_returns r ON r.id = ri.return_id
		WHERE r.status = 'CONFIRMED' AND ri.accepted_qty > 0
		  AND ri.accepted_qty > (
			SELECT COALESCE(SUM(-m.quantity_delta), 0)
			FROM stock_movements m
			WHERE m.department_id = r.department_id
			  AND m.product_id = ri.product_id
			  AND m.movement_type = 'OUT'
			  AND m.created_at <= r.created_at
		  ) - (
			SELECT COALESCE(SUM(ri2.accepted_qty), 0)
			FROM stock_return_items ri2
			JOIN stock_returns r2 ON r2.id = ri2.return_id
			WHERE r2.department_id = r.department_id
			  AND ri2.product_id = ri.product_id
			  AND r2.status = 'CONFIRMED'
			  AND r2.id <> r.id
			  AND r2.created_at <= r.created_at
		  )`).Scan(&badReturns).Error; err != nil {
		return err
	}
	if badReturns > 0 {
		return fmt.Errorf("seed verification failed: %d return(s) exceed net issued quantity", badReturns)
	}
	return nil
}

func ptr(v string) *string { return &v }

// lookupID returns uuid.Nil when the query finds no row.
func lookupID(tx *gorm.DB, query string, args ...any) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.Raw(query, args...).Row().Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil
	}
	return id, err
}
