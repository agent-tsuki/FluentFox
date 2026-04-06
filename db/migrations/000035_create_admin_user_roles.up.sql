-- 000035_create_admin_user_roles.up.sql
-- Junction table assigning admin roles to users.
-- A user can hold multiple roles simultaneously.

CREATE TABLE IF NOT EXISTS admin_user_roles (
    user_id    UUID        NOT NULL,
    role_id    UUID        NOT NULL,
    -- The admin user who granted this role assignment
    granted_by UUID        NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, role_id),
    CONSTRAINT fk_admin_user_roles_users       FOREIGN KEY (user_id)    REFERENCES users       (id) ON DELETE CASCADE,
    CONSTRAINT fk_admin_user_roles_roles       FOREIGN KEY (role_id)    REFERENCES admin_roles (id) ON DELETE CASCADE,
    CONSTRAINT fk_admin_user_roles_granted_by  FOREIGN KEY (granted_by) REFERENCES users       (id) ON DELETE SET NULL
);

CREATE INDEX idx_admin_user_roles_user_id ON admin_user_roles (user_id);
CREATE INDEX idx_admin_user_roles_role_id ON admin_user_roles (role_id);
