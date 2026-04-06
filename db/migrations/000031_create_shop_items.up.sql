-- 000031_create_shop_items.up.sql
-- Items available for purchase with XP in the in-app shop.
-- UUID PK: user-facing resource, exposed as an identifier in API responses.

CREATE TABLE IF NOT EXISTS shop_items (
    id          UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100)   NOT NULL,
    description TEXT           NOT NULL,
    -- Controls how the frontend renders and applies the purchased item
    item_type   shop_item_type NOT NULL,
    xp_cost     INTEGER        NOT NULL,
    -- Type-specific configuration (shape varies per item_type)
    -- e.g. ui_theme: {"primary_color": "#..."}, streak_freeze: {"days": 1}
    metadata    JSONB          NULL,
    -- FALSE = item hidden from shop (discontinued or draft)
    is_active   BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_shop_items_xp_cost CHECK (xp_cost >= 0)
);

CREATE INDEX idx_shop_items_type      ON shop_items (item_type);
CREATE INDEX idx_shop_items_is_active ON shop_items (is_active);
