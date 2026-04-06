-- 000002_create_users.up.sql
-- Central identity record. Root of the entire user data graph.

CREATE TABLE IF NOT EXISTS users (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username          VARCHAR(50)  NOT NULL,
    email             VARCHAR(255) NOT NULL,
    -- bcrypt hash — never store plaintext, never expose in API responses
    password          VARCHAR(255) NOT NULL,
    -- 'student' | 'admin' — checked by RBAC middleware on every admin request
    role              VARCHAR(20)  NOT NULL DEFAULT 'student',
    -- NULL = email not yet verified; set to NOW() on verification link click
    email_verified_at TIMESTAMPTZ  NULL,
    -- FALSE = account suspended by admin; checked on every login
    is_active         BOOLEAN      NOT NULL DEFAULT TRUE,
    -- Soft delete flag — row retained for audit/GDPR compliance
    is_deleted        BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_email    UNIQUE (email),
    CONSTRAINT chk_users_role    CHECK  (role IN ('student', 'admin'))
);

-- Used on every login query
CREATE INDEX idx_users_email      ON users (email);
-- Used on registration uniqueness check
CREATE INDEX idx_users_username   ON users (username);
-- Filter out deleted users in list queries
CREATE INDEX idx_users_is_deleted ON users (is_deleted);
