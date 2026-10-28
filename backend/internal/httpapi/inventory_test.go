package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"game/backend/internal/game"
	"game/backend/internal/inventory"
	"game/backend/internal/player"
)

func TestInventoryHandler(t *testing.T) {
	owner := player.Player{ID: "11111111-1111-1111-1111-111111111111", Login: "fox"}
	gameID := "33333333-3333-3333-3333-333333333333"
	cookies := map[string]string{accessCookie: "access-1"}
	item := inventory.Slot{
		Index:    4,
		Quantity: 2,
		Item: inventory.Item{
			ID:          "44444444-4444-4444-4444-444444444444",
			Slug:        "wood",
			Name:        "Дерево",
			IconKey:     "item-wood",
			Description: "Сухие ветки",
			MaxStack:    20,
		},
	}

	t.Run("empty", func(t *testing.T) {
		bags := &fakeInventory{}
		mux := http.NewServeMux()
		RegisterInventory(mux, bags, &fakeAuth{current: owner})
		rec := request(t, mux, http.MethodGet, "/api/games/"+gameID+"/inventory", "", cookies)
		if rec.Code != http.StatusOK || bags.gameID != gameID || bags.playerID != owner.ID {
			t.Fatalf("status %d game %s player %s body %s", rec.Code, bags.gameID, bags.playerID, rec.Body.String())
		}
		var body inventoryBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Capacity != inventory.Capacity || len(body.Slots) != 0 {
			t.Fatalf("body %+v", body)
		}
	})

	t.Run("slot", func(t *testing.T) {
		bags := &fakeInventory{bag: inventory.Inventory{Capacity: inventory.Capacity, Slots: []inventory.Slot{item}}}
		mux := http.NewServeMux()
		RegisterInventory(mux, bags, &fakeAuth{current: owner})
		rec := request(t, mux, http.MethodGet, "/api/games/"+gameID+"/inventory", "", cookies)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d %s", rec.Code, rec.Body.String())
		}
		var body inventoryBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Slots) != 1 || body.Slots[0].Index != 4 || body.Slots[0].Quantity != 2 || body.Slots[0].Item.Slug != "wood" {
			t.Fatalf("body %+v", body)
		}
	})

	t.Run("missing game", func(t *testing.T) {
		mux := http.NewServeMux()
		RegisterInventory(mux, &fakeInventory{err: game.ErrNotFound}, &fakeAuth{current: owner})
		rec := request(t, mux, http.MethodGet, "/api/games/"+gameID+"/inventory", "", cookies)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status %d %s", rec.Code, rec.Body.String())
		}
		assertErrorCode(t, rec.Body.Bytes(), "game_not_found")
	})

	t.Run("unauthorized", func(t *testing.T) {
		bags := &fakeInventory{}
		mux := http.NewServeMux()
		RegisterInventory(mux, bags, &fakeAuth{current: owner})
		rec := request(t, mux, http.MethodGet, "/api/games/"+gameID+"/inventory", "", nil)
		if rec.Code != http.StatusUnauthorized || bags.lists != 0 {
			t.Fatalf("status %d lists %d", rec.Code, bags.lists)
		}
		assertErrorCode(t, rec.Body.Bytes(), "unauthorized")
	})
}

type fakeInventory struct {
	bag      inventory.Inventory
	err      error
	gameID   string
	playerID string
	lists    int
}

func (f *fakeInventory) List(_ context.Context, gameID, playerID string) (inventory.Inventory, error) {
	f.lists++
	f.gameID = gameID
	f.playerID = playerID
	if f.err != nil {
		return inventory.Inventory{}, f.err
	}
	if f.bag.Slots == nil {
		f.bag.Slots = []inventory.Slot{}
	}
	if f.bag.Capacity == 0 {
		f.bag.Capacity = inventory.Capacity
	}
	return f.bag, nil
}
