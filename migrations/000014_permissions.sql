-- +goose Up
-- Fine-grained grants for staff such as teachers. resource_type '' means every
-- resource; resource_id is text without a foreign key because it can point at
-- either a channel or a course, and those tables arrive in a later migration.
CREATE TABLE user_permissions (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    permission    text NOT NULL,
    resource_type text NOT NULL DEFAULT '',
    resource_id   text NOT NULL DEFAULT '',
    granted_by    uuid,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_user_permissions_grant UNIQUE (user_id, permission, resource_type, resource_id)
);

-- +goose Down
DROP TABLE IF EXISTS user_permissions;
