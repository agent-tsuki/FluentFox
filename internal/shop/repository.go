// Package shop — repository.go.
// Owns all SQL for shop_items and user_inventory.
package shop

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles shop-related DB queries.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a shop Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ListAvailable returns all available shop items.
func (r *Repository) ListAvailable(ctx context.Context) ([]*ShopItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, item_type, xp_cost, available
		 FROM shop_items WHERE available = true ORDER BY xp_cost`)
	if err != nil {
		return nil, fmt.Errorf("shop repository: list items: %w", err)
	}
	defer rows.Close()

	var items []*ShopItem
	for rows.Next() {
		item := &ShopItem{}
		if err := rows.Scan(&item.ID, &item.Name, &item.Description,
			&item.ItemType, &item.XPCost, &item.Available); err != nil {
			return nil, fmt.Errorf("shop repository: scan item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetByID fetches a shop item by primary key.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*ShopItem, error) {
	item := &ShopItem{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, item_type, xp_cost, available FROM shop_items WHERE id = $1`, id,
	).Scan(&item.ID, &item.Name, &item.Description, &item.ItemType, &item.XPCost, &item.Available)
	if err != nil {
		return nil, fmt.Errorf("shop repository: get item: %w", err)
	}
	return item, nil
}

// AddToInventory inserts a purchased item into the user's inventory.
func (r *Repository) AddToInventory(ctx context.Context, userID, itemID uuid.UUID) (*UserInventory, error) {
	inv := &UserInventory{}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO user_inventory (id, user_id, item_id, purchased_at)
		 VALUES ($1, $2, $3, NOW())
		 RETURNING id, user_id, item_id, used_at, purchased_at`,
		uuid.New(), userID, itemID,
	).Scan(&inv.ID, &inv.UserID, &inv.ItemID, &inv.UsedAt, &inv.PurchasedAt)
	if err != nil {
		return nil, fmt.Errorf("shop repository: add to inventory: %w", err)
	}
	return inv, nil
}

// GetInventory returns the user's purchased items with item details.
func (r *Repository) GetInventory(ctx context.Context, userID uuid.UUID) ([]*InventoryResponse, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ui.id, si.name, si.item_type, ui.used_at
		 FROM user_inventory ui
		 JOIN shop_items si ON si.id = ui.item_id
		 WHERE ui.user_id = $1
		 ORDER BY ui.purchased_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("shop repository: get inventory: %w", err)
	}
	defer rows.Close()

	var result []*InventoryResponse
	for rows.Next() {
		inv := &InventoryResponse{}
		var usedAt *string
		if err := rows.Scan(&inv.ID, &inv.ItemName, &inv.ItemType, &usedAt); err != nil {
			return nil, fmt.Errorf("shop repository: scan inventory: %w", err)
		}
		inv.UsedAt = usedAt
		result = append(result, inv)
	}
	return result, rows.Err()
}
