CREATE SEQUENCE request_number_seq START 1;

CREATE TABLE stock_requests (
  id uuid PRIMARY KEY,
  request_number text NOT NULL UNIQUE,
  department_id uuid NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
  requested_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status text NOT NULL DEFAULT 'PENDING'
    CHECK (status IN ('PENDING','PARTIALLY_FULFILLED','FULFILLED','REJECTED','CANCELLED')),
  purpose text NOT NULL CHECK (length(btrim(purpose)) > 0),
  rejection_reason text,
  processed_by uuid REFERENCES users(id) ON DELETE RESTRICT,
  processed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT stock_requests_rejected_reason CHECK (
    status <> 'REJECTED' OR rejection_reason IS NOT NULL
  ),
  CONSTRAINT stock_requests_processed CHECK (
    status NOT IN ('PARTIALLY_FULFILLED','FULFILLED')
    OR (processed_by IS NOT NULL AND processed_at IS NOT NULL)
  )
);
CREATE INDEX stock_requests_department_status_idx ON stock_requests (department_id, status, created_at DESC);
CREATE INDEX stock_requests_status_created_idx ON stock_requests (status, created_at DESC);

CREATE TABLE stock_request_items (
  id uuid PRIMARY KEY,
  request_id uuid NOT NULL REFERENCES stock_requests(id) ON DELETE RESTRICT,
  product_id uuid NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
  requested_qty integer NOT NULL CHECK (requested_qty > 0),
  issued_qty integer NOT NULL DEFAULT 0 CHECK (issued_qty >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT stock_request_items_qty CHECK (issued_qty <= requested_qty),
  UNIQUE (request_id, product_id)
);
CREATE INDEX stock_request_items_product_idx ON stock_request_items (product_id);
