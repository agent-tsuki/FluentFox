-- 000036_create_admin_audit_log.up.sql
-- Append-only audit trail of all admin actions.
-- One row per admin operation. Never updated, never deleted.
-- append-only log: created_at only, no updated_at.

CREATE TABLE IF NOT EXISTS admin_audit_log (
    id          UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    -- The admin who performed the action
    -- ON DELETE RESTRICT: cannot delete a user who has audit log entries
    actor_id    UUID           NOT NULL,
    -- Action verb, e.g. 'create', 'update', 'delete', 'publish'
    action      VARCHAR(100)   NOT NULL,
    -- The resource type that was acted upon
    resource    admin_resource NOT NULL,
    -- The ID of the affected row (TEXT to accommodate both UUID and SERIAL integer IDs)
    resource_id TEXT           NULL,
    -- Arbitrary JSON payload: before/after values, context, etc.
    metadata    JSONB          NULL,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_admin_audit_log_actor FOREIGN KEY (actor_id) REFERENCES users (id) ON DELETE RESTRICT
);

CREATE INDEX idx_admin_audit_log_actor_id  ON admin_audit_log (actor_id, created_at);
CREATE INDEX idx_admin_audit_log_resource  ON admin_audit_log (resource, created_at);
