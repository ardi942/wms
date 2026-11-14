CREATE TABLE products (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  sku text NOT NULL UNIQUE,
  name text NOT NULL,
  description text,
  category_id uuid NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
  unit_id uuid NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
  min_stock integer NOT NULL DEFAULT 0 CHECK (min_stock >= 0),
  unit_cost numeric(14,2) CHECK (unit_cost IS NULL OR unit_cost >= 0),
  is_active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT products_sku_uppercase CHECK (sku = upper(sku))
);
CREATE INDEX products_name_idx ON products (lower(name));
CREATE INDEX products_category_idx ON products (category_id);
CREATE INDEX products_active_idx ON products (is_active);
