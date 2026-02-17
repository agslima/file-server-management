-- Queryable audit trail + append-only guardrails.

CREATE TABLE IF NOT EXISTS audit_events (
  id BIGSERIAL PRIMARY KEY,
  event_type TEXT NOT NULL,
  task_id TEXT,
  correlation_id TEXT,
  message TEXT,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_created_at ON audit_events(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_events_task_id ON audit_events(task_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_correlation_id ON audit_events(correlation_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);

CREATE OR REPLACE FUNCTION audit_events_append_only_guard()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION 'audit_events is append-only; % is not allowed', TG_OP;
END;
$$;

DROP TRIGGER IF EXISTS trg_audit_events_block_update ON audit_events;
CREATE TRIGGER trg_audit_events_block_update
BEFORE UPDATE ON audit_events
FOR EACH ROW
EXECUTE FUNCTION audit_events_append_only_guard();

DROP TRIGGER IF EXISTS trg_audit_events_block_delete ON audit_events;
CREATE TRIGGER trg_audit_events_block_delete
BEFORE DELETE ON audit_events
FOR EACH ROW
EXECUTE FUNCTION audit_events_append_only_guard();

REVOKE UPDATE, DELETE ON audit_events FROM PUBLIC;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fileengine') THEN
    REVOKE UPDATE, DELETE ON audit_events FROM fileengine;
  END IF;
END;
$$;
