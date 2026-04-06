-- 000033_create_admin_roles.up.sql
-- Named admin roles (e.g. "content_editor", "support_agent").
-- Permissions are defined per-role per-resource in admin_role_permissions.

CREATE TABLE IF NOT EXISTS admin_roles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    description TEXT         NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_admin_roles_name UNIQUE (name)
);
