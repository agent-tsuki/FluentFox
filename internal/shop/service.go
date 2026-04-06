// Package shop — service.go.
// Business logic: purchasing items (deducting XP), inventory management.
package shop

import (
	"context"
	"errors"
	"fmt"

	"github.com/fluentfox/api/internal/xp"
	"github.com/google/uuid"
)

// ErrInsufficientXP is returned when the user cannot afford an item.
var ErrInsufficientXP = errors.New("shop service: insufficient XP")

// ErrItemUnavailable is returned when the item is not for sale.
var ErrItemUnavailable = errors.New("shop service: item not available")

// Service handles shop business logic.
type Service struct {
	repo    *Repository
	xpSvc   *xp.Service
}

// NewService constructs a shop Service.
func NewService(repo *Repository, xpSvc *xp.Service) *Service {
	return &Service{repo: repo, xpSvc: xpSvc}
}

// ListItems returns all available shop items.
func (s *Service) ListItems(ctx context.Context) ([]*ShopItemResponse, error) {
	items, err := s.repo.ListAvailable(ctx)
	if err != nil {
		return nil, fmt.Errorf("shop service: list items: %w", err)
	}

	resp := make([]*ShopItemResponse, len(items))
	for i, item := range items {
		resp[i] = &ShopItemResponse{
			ID:          item.ID.String(),
			Name:        item.Name,
			Description: item.Description,
			ItemType:    item.ItemType,
			XPCost:      item.XPCost,
		}
	}
	return resp, nil
}

// Purchase deducts XP and adds the item to the user's inventory.
func (s *Service) Purchase(ctx context.Context, userID uuid.UUID, req PurchaseRequest) (*PurchaseResponse, error) {
	itemID, err := uuid.Parse(req.ItemID)
	if err != nil {
		return nil, fmt.Errorf("shop service: invalid item_id: %w", err)
	}

	item, err := s.repo.GetByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("shop service: item not found: %w", err)
	}

	if !item.Available {
		return nil, ErrItemUnavailable
	}

	userXP, err := s.xpSvc.GetXP(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("shop service: get user xp: %w", err)
	}

	if userXP.Total < item.XPCost {
		return nil, ErrInsufficientXP
	}

	// Deduct XP (negative award).
	if err := s.xpSvc.Award(ctx, userID, -item.XPCost, "shop_purchase", &itemID); err != nil {
		return nil, fmt.Errorf("shop service: deduct xp: %w", err)
	}

	inv, err := s.repo.AddToInventory(ctx, userID, itemID)
	if err != nil {
		return nil, fmt.Errorf("shop service: add to inventory: %w", err)
	}

	return &PurchaseResponse{
		Message:     fmt.Sprintf("%s purchased successfully", item.Name),
		InventoryID: inv.ID.String(),
	}, nil
}

// GetInventory returns the user's owned items.
func (s *Service) GetInventory(ctx context.Context, userID uuid.UUID) ([]*InventoryResponse, error) {
	return s.repo.GetInventory(ctx, userID)
}
