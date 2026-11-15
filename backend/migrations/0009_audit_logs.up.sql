CREATE TABLE audit_logs (
  id uuid PRIMARY KEY,
  entity_type text NOT NULL CHECK (entity_type IN ('USER', 'PRODUCT', 'CATEGORY', 'UNIT')),
  entity_id uuid NOT NULL,
  action text NOT NULL CHECK (action IN (
    'USER_CREATED',
    'USER_ACTIVATED',
    'USER_DEACTIVATED',
    'USER_PROFILE_UPDATED',
    'USER_PASSWORD_CHANGED',
    'PRODUCT_COST_UPDATED',
    'PRODUCT_ACTIVATED',
    'PRODUCT_DEACTIVATED',
    'PRODUCT_UPDATED'
  )),
  actor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  actor_name text NOT NULL,
  actor_email text NOT NULL,
  actor_role text NOT NULL,
  reason text,
  changed_fields text[] NOT NULL DEFAULT '{}',
  before_data jsonb,
  after_data jsonb,
  request_id text,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT audit_reason_length CHECK (
    reason IS NULL OR char_length(btrim(reason)) BETWEEN 5 AND 500
  ),
  CONSTRAINT audit_status_reason_required CHECK (
    action NOT IN ('USER_ACTIVATED', 'USER_DEACTIVATED')
    OR (
      reason IS NOT NULL
      AND char_length(btrim(reason)) BETWEEN 5 AND 500
    )
  )
);

CREATE INDEX audit_logs_created_at_idx ON audit_logs (created_at DESC);
CREATE INDEX audit_logs_entity_idx ON audit_logs (entity_type, entity_id, created_at DESC);
CREATE INDEX audit_logs_actor_idx ON audit_logs (actor_id, created_at DESC);
CREATE INDEX audit_logs_action_created_idx ON audit_logs (action, created_at DESC);
CREATE INDEX audit_logs_request_idx ON audit_logs (request_id) WHERE request_id IS NOT NULL;

CREATE OR REPLACE FUNCTION prevent_audit_log_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'audit_logs is append-only';
END;
$$;

CREATE TRIGGER audit_logs_immutable
BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_mutation();
