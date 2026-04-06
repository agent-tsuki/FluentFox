// Package shop — model.go.
// DB models and DTOs for the in-app shop (streak freezes, cosmetics, etc.).
package shop

import (
	"time"

	"github.com/google/uuid"
)

// ShopItem is the DB model for the shop_items table.
type ShopItem struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	ItemType    string    `db:"item_type"` // streak_freeze, avatar_frame, badge
	XPCost      int       `db:"xp_cost"`
	Available   bool      `db:"available"`
}

// UserInventory is the DB model for the user_inventory table.
type UserInventory struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	ItemID     uuid.UUID  `db:"item_id"`
	UsedAt     *time.Time `db:"used_at"`
	PurchasedAt time.Time `db:"purchased_at"`
}

// --- Request DTOs ---

// PurchaseRequest is the payload for POST /shop/purchase.
type PurchaseRequest struct {
	ItemID string `json:"item_id" validate:"required,uuid"`
}

// --- Response DTOs ---

// ShopItemResponse is the public representation of a shop item.
type ShopItemResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ItemType    string `json:"item_type"`
	XPCost      int    `json:"xp_cost"`
}

// PurchaseResponse is returned after a successful purchase.
type PurchaseResponse struct {
	Message    string `json:"message"`
	InventoryID string `json:"inventory_id"`
}

// InventoryResponse is the user's inventory of purchased items.
type InventoryResponse struct {
	ID       string  `json:"id"`
	ItemName string  `json:"item_name"`
	ItemType string  `json:"item_type"`
	UsedAt   *string `json:"used_at"`
}
