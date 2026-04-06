-- 000034_create_admin_role_permissions.up.sql
-- Grants a specific permission level on a specific resource to a role.
-- Composite PK enforces one permission entry per role per resource.

CREATE TABLE IF NOT EXISTS admin_role_permissions (
    role_id    UUID                   NOT NULL,
    resource   admin_resource         NOT NULL,
    permission admin_permission_level NOT NULL,

    PRIMARY KEY (role_id, resource),
    CONSTRAINT fk_admin_role_permissions_roles FOREIGN KEY (role_id) REFERENCES admin_roles (id) ON DELETE CASCADE
);
