CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email citext NOT NULL UNIQUE,
  password_hash text NOT NULL,
  full_name text NOT NULL,
  role text NOT NULL CHECK (role IN ('SUPERVISOR','WAREHOUSE_ADMIN','DEPARTMENT_USER')),
  department_id uuid REFERENCES departments(id) ON DELETE RESTRICT,
  is_active boolean NOT NULL DEFAULT true,
  last_login_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT users_department_required CHECK (
    (role = 'DEPARTMENT_USER' AND department_id IS NOT NULL)
    OR (role IN ('SUPERVISOR','WAREHOUSE_ADMIN') AND department_id IS NULL)
  )
);
