package httpapi

import (
	"context"
	"net/http"

	"game/backend/internal/inventory"
)

type InventoryService interface {
	List(ctx context.Context, gameID, playerID string) (inventory.Inventory, error)
}

type inventoryBody struct {
	Capacity int                 `json:"capacity" example:"20"`
	Slots    []inventorySlotBody `json:"slots"`
} //@name inventory

type inventorySlotBody struct {
	Index    int      `json:"index" example:"0"`
	Quantity int      `json:"quantity" example:"1"`
	Item     itemBody `json:"item"`
} //@name inventorySlot

type itemBody struct {
	ID          string `json:"id" example:"44444444-4444-4444-4444-444444444444"`
	Slug        string `json:"slug" example:"wood"`
	Name        string `json:"name" example:"Древесина"`
	IconKey     string `json:"iconKey" example:"wood"`
	Description string `json:"description" example:""`
	MaxStack    int    `json:"maxStack" example:"99"`
} //@name item

func RegisterInventory(mux *http.ServeMux, bags InventoryService, sessions PlayerSession) {
	h := &inventoryHandler{bags: bags, sessions: sessions}
	mux.HandleFunc("GET /api/games/{id}/inventory", h.list)
}

type inventoryHandler struct {
	bags     InventoryService
	sessions PlayerSession
}

// list возвращает инвентарь текущего игрока в игре.
//
//	@Summary	Инвентарь игрока
//	@Tags		inventory
//	@Produce	json
//	@Security	AccessCookie
//	@Param		id	path		string	true	"ID игры"
//	@Success	200	{object}	inventoryBody
//	@Failure	401	{object}	errorBody	"unauthorized"
//	@Failure	404	{object}	errorBody	"game_not_found"
//	@Failure	500	{object}	errorBody
//	@Router		/api/games/{id}/inventory [get]
func (h *inventoryHandler) list(w http.ResponseWriter, r *http.Request) {
	current, ok := authenticate(w, r, h.sessions)
	if !ok {
		return
	}
	bag, err := h.bags.List(r.Context(), r.PathValue("id"), current.ID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toInventoryBody(bag))
}

func toInventoryBody(bag inventory.Inventory) inventoryBody {
	body := inventoryBody{
		Capacity: bag.Capacity,
		Slots:    make([]inventorySlotBody, 0, len(bag.Slots)),
	}
	for _, slot := range bag.Slots {
		body.Slots = append(body.Slots, inventorySlotBody{
			Index:    slot.Index,
			Quantity: slot.Quantity,
			Item: itemBody{
				ID:          slot.Item.ID,
				Slug:        slot.Item.Slug,
				Name:        slot.Item.Name,
				IconKey:     slot.Item.IconKey,
				Description: slot.Item.Description,
				MaxStack:    slot.Item.MaxStack,
			},
		})
	}
	return body
}
