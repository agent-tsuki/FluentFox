-- 000032_create_user_inventory.up.sql
-- Items purchased by users from the shop. One row per purchase.
-- A user can own multiple copies of the same item (e.g. multiple streak freezes).

CREATE TABLE IF NOT EXISTS user_inventory (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL,
    item_id      UUID        NOT NULL,
    -- XP deducted at time of purchase (snapshot in case item price changes)
    xp_spent     INTEGER     NOT NULL,
    -- For consumable items: timestamp when the item was used. NULL = unused.
    used_at      TIMESTAMPTZ NULL,
    purchased_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_user_inventory_xp_spent CHECK       (xp_spent >= 0),
    CONSTRAINT fk_user_inventory_users      FOREIGN KEY (user_id) REFERENCES users      (id) ON DELETE CASCADE,
    -- RESTRICT: prevents deleting a shop item that users have purchased
    CONSTRAINT fk_user_inventory_shop_items FOREIGN KEY (item_id) REFERENCES shop_items (id) ON DELETE RESTRICT
);

CREATE INDEX idx_user_inventory_user_id ON user_inventory (user_id);
CREATE INDEX idx_user_inventory_item_id ON user_inventory (item_id);
