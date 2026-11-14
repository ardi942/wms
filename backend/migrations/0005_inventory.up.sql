CREATE TABLE inventory_balances (
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  warehouse_id uuid NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
  quantity integer NOT NULL DEFAULT 0 CHECK (quantity >= 0),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (product_id, warehouse_id)
);
CREATE INDEX inventory_balances_qty_idx ON inventory_balances (warehouse_id, quantity);

CREATE TABLE stock_movements (
  id uuid PRIMARY KEY,
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  warehouse_id uuid NOT NULL REFERENCES warehouses(id) ON DELETE RESTRICT,
  movement_type text NOT NULL CHECK (movement_type IN ('IN','OUT','ADJUSTMENT')),
  quantity_delta integer NOT NULL CHECK (quantity_delta <> 0),
  balance_after integer NOT NULL CHECK (balance_after >= 0),
  reference_type text NOT NULL CHECK (reference_type IN ('RECEIPT','REQUEST','RETURN','ADJUSTMENT')),
  reference_id uuid NOT NULL,
  reference_number text NOT NULL,
  department_id uuid REFERENCES departments(id) ON DELETE RESTRICT,
  actor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  supplier text,
  notes text,
  reason text,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT stock_movements_sign CHECK (
    (movement_type = 'IN' AND quantity_delta > 0)
    OR (movement_type = 'OUT' AND quantity_delta < 0)
    OR (movement_type = 'ADJUSTMENT' AND quantity_delta <> 0)
  )
);
CREATE INDEX stock_movements_product_created_idx ON stock_movements (product_id, created_at DESC);
CREATE INDEX stock_movements_created_idx ON stock_movements (created_at DESC);
CREATE INDEX stock_movements_department_created_idx ON stock_movements (department_id, created_at DESC);
CREATE INDEX stock_movements_reference_idx ON stock_movements (reference_type, reference_id);

CREATE OR REPLACE FUNCTION prevent_stock_movement_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'stock_movements is append-only';
END;
$$;

CREATE TRIGGER stock_movements_immutable
BEFORE UPDATE OR DELETE ON stock_movements
FOR EACH ROW EXECUTE FUNCTION prevent_stock_movement_mutation();
