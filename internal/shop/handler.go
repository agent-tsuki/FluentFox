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

// ListItems godoc
// @Summary      List shop items
// @Description  Returns all currently available items in the XP shop.
// @Tags         shop
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} ShopItemResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /shop [get]
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListItems(r.Context())
	if err != nil {
		h.log.Error("shop: list items", zap.Error(err))
		response.InternalServerError(w)
		return
	}
	response.JSON(w, http.StatusOK, items)
}

// Purchase godoc
// @Summary      Purchase a shop item
// @Description  Spends XP to purchase an item. Returns inventory ID of the new item.
// @Tags         shop
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body PurchaseRequest true "Item ID to purchase"
// @Success      200 {object} PurchaseResponse
// @Failure      400 {object} response.ErrorResponse "Insufficient XP or item unavailable"
// @Failure      401 {object} response.ErrorResponse
// @Failure      422 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /shop/purchase [post]
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

// GetInventory godoc
// @Summary      Get user inventory
// @Description  Returns all items the authenticated user has purchased.
// @Tags         shop
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} InventoryResponse
// @Failure      401 {object} response.ErrorResponse
// @Failure      500 {object} response.ErrorResponse
// @Router       /shop/inventory [get]
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
