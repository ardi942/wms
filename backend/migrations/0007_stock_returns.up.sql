CREATE SEQUENCE return_number_seq START 1;

CREATE TABLE stock_returns (
  id uuid PRIMARY KEY,
  return_number text NOT NULL UNIQUE,
  department_id uuid NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
  requested_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status text NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING','CONFIRMED','REJECTED','CANCELLED')),
  reason text NOT NULL CHECK (length(btrim(reason)) > 0),
  rejection_reason text,
  processed_by uuid REFERENCES users(id) ON DELETE RESTRICT,
  processed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT stock_returns_rejected_reason CHECK (
    status <> 'REJECTED' OR rejection_reason IS NOT NULL
  ),
  CONSTRAINT stock_returns_processed CHECK (
    status <> 'CONFIRMED' OR (processed_by IS NOT NULL AND processed_at IS NOT NULL)
  )
);
CREATE INDEX stock_returns_department_status_idx ON stock_returns (department_id, status, created_at DESC);

CREATE TABLE stock_return_items (
  id uuid PRIMARY KEY,
  return_id uuid NOT NULL REFERENCES stock_returns(id) ON DELETE RESTRICT,
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  requested_qty integer NOT NULL CHECK (requested_qty > 0),
  accepted_qty integer NOT NULL DEFAULT 0 CHECK (accepted_qty >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT stock_return_items_qty CHECK (accepted_qty <= requested_qty),
  UNIQUE (return_id, product_id)
);
CREATE INDEX stock_return_items_product_idx ON stock_return_items (product_id);
