// Package shop — handler.go.
// HTTP handlers for shop endpoints.
package shop

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fluentfox/api/internal/middleware"
	"github.com/fluentfox/api/pkg/response"
	"github.com/fluentfox/api/pkg/validator"
	"go.uber.org/zap"
)

// Handler holds shop handler dependencies.
type Handler struct {
	svc       *Service
	validator *validator.Validator
	log       *zap.Logger
}

// NewHandler constructs a shop Handler.
func NewHandler(svc *Service, v *validator.Validator, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validator: v, log: log}
}

// ListItems handles GET /shop.
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListItems(r.Context())
	if err != nil {
		h.log.Error("shop: list items", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

// Purchase handles POST /shop/purchase.
func (h *Handler) Purchase(w http.ResponseWriter, r *http.Request) {
	var req PurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		response.UnprocessableEntity(w, err.Error())
		return
	}

	userID := middleware.ContextUserID(r.Context())
	result, err := h.svc.Purchase(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrInsufficientXP) {
			response.BadRequest(w, "not enough XP to purchase this item")
			return
		}
		if errors.Is(err, ErrItemUnavailable) {
			response.BadRequest(w, "this item is not currently available")
			return
		}
		h.log.Error("shop: purchase", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// GetInventory handles GET /shop/inventory.
func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ContextUserID(r.Context())
	inv, err := h.svc.GetInventory(r.Context(), userID)
	if err != nil {
		h.log.Error("shop: get inventory", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, inv)
}
