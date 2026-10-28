package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"game/backend/internal/gather"
	"game/backend/internal/inventory"
	"game/backend/internal/mapstate"
	"game/backend/internal/player"
	"game/backend/internal/realtime"
)

func TestMapHandler(t *testing.T) {
	owner := player.Player{ID: "11111111-1111-1111-1111-111111111111", Login: "fox"}
	gameID := "33333333-3333-3333-3333-333333333333"
	cookies := map[string]string{accessCookie: "access-1"}
	flower := "flower-0"

	t.Run("empty journal", func(t *testing.T) {
		journal := &fakeMap{}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, &fakePublish{})
		rec := request(t, mux, http.MethodGet, "/api/games/"+gameID+"/map", "", cookies)
		if rec.Code != http.StatusOK || journal.gameID != gameID {
			t.Fatalf("status %d game %s body %s", rec.Code, journal.gameID, rec.Body.String())
		}
		var body mapBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Cells == nil || len(body.Cells) != 0 {
			t.Fatalf("body %+v", body)
		}
	})

	t.Run("cleared flora", func(t *testing.T) {
		journal := &fakeMap{cells: []mapstate.Cell{{
			X: 1, Y: 2,
			Layers: map[string]*string{mapstate.LayerFlora: nil, mapstate.LayerTerrain: &flower},
		}}}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, &fakePublish{})
		rec := request(t, mux, http.MethodGet, "/api/games/"+gameID+"/map", "", cookies)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d %s", rec.Code, rec.Body.String())
		}
		var body mapBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Cells) != 1 || body.Cells[0].X != 1 || body.Cells[0].Y != 2 {
			t.Fatalf("body %+v", body)
		}
		if body.Cells[0].Layers[mapstate.LayerFlora] != nil {
			t.Fatal("flora was not null")
		}
		if body.Cells[0].Layers[mapstate.LayerTerrain] == nil || *body.Cells[0].Layers[mapstate.LayerTerrain] != flower {
			t.Fatalf("terrain %+v", body.Cells[0].Layers)
		}
	})

	t.Run("gather publishes map", func(t *testing.T) {
		journal := &fakeMap{result: gather.Result{
			Inventory: inventory.Inventory{
				Capacity: inventory.Capacity,
				Slots: []inventory.Slot{{
					Index: 0, Quantity: 1,
					Item: inventory.Item{ID: "44444444-4444-4444-4444-444444444444", Slug: "flower-1", Name: "Цветок II", IconKey: "flower-1", MaxStack: 99},
				}},
			},
			Cell: mapstate.Cell{X: 4, Y: 5, Layers: map[string]*string{mapstate.LayerFlora: nil}},
		}}
		publish := &fakePublish{}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, publish)
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/gather", `{"x":4,"y":5,"kind":"flower-1"}`, cookies)
		if rec.Code != http.StatusOK || journal.playerID != owner.ID || journal.x != 4 || journal.kind != "flower-1" {
			t.Fatalf("status %d player %s x %d kind %s body %s", rec.Code, journal.playerID, journal.x, journal.kind, rec.Body.String())
		}
		var body gatherBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Capacity != inventory.Capacity || len(body.Slots) != 1 || body.Slots[0].Item.Slug != "flower-1" {
			t.Fatalf("body %+v", body)
		}
		if body.Cell.Layers["flora"] != nil {
			t.Fatal("response flora was not null")
		}
		if publish.calls != 1 || publish.room != "game:"+gameID || publish.except != nil || publish.env.Type != "map" {
			t.Fatalf("publish %+v", publish)
		}
		var cell cellBody
		if err := json.Unmarshal(publish.env.Payload, &cell); err != nil {
			t.Fatal(err)
		}
		if cell.X != 4 || cell.Y != 5 || cell.Layers["flora"] != nil {
			t.Fatalf("event %+v", cell)
		}
	})

	t.Run("already taken does not publish", func(t *testing.T) {
		journal := &fakeMap{err: gather.ErrAlreadyTaken}
		publish := &fakePublish{}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, publish)
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/gather", `{"x":1,"y":1,"kind":"flower-0"}`, cookies)
		if rec.Code != http.StatusConflict || publish.calls != 0 {
			t.Fatalf("status %d publishes %d", rec.Code, publish.calls)
		}
		assertErrorCode(t, rec.Body.Bytes(), "already_taken")
	})

	t.Run("bag full", func(t *testing.T) {
		mux := http.NewServeMux()
		RegisterMap(mux, &fakeMap{err: gather.ErrBagFull}, &fakeAuth{current: owner}, &fakePublish{})
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/gather", `{"x":1,"y":1,"kind":"flower-0"}`, cookies)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status %d %s", rec.Code, rec.Body.String())
		}
		assertErrorCode(t, rec.Body.Bytes(), "bag_full")
	})

	t.Run("not in reach", func(t *testing.T) {
		mux := http.NewServeMux()
		RegisterMap(mux, &fakeMap{err: gather.ErrNotInReach}, &fakeAuth{current: owner}, &fakePublish{})
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/gather", `{"x":1,"y":1,"kind":"flower-0"}`, cookies)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d %s", rec.Code, rec.Body.String())
		}
		assertErrorCode(t, rec.Body.Bytes(), "not_in_reach")
	})

	t.Run("invalid kind", func(t *testing.T) {
		mux := http.NewServeMux()
		RegisterMap(mux, &fakeMap{err: gather.ErrInvalidKind}, &fakeAuth{current: owner}, &fakePublish{})
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/gather", `{"x":1,"y":1,"kind":"lily"}`, cookies)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status %d %s", rec.Code, rec.Body.String())
		}
		assertErrorCode(t, rec.Body.Bytes(), "invalid_kind")
	})

	t.Run("invalid body", func(t *testing.T) {
		journal := &fakeMap{}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, &fakePublish{})
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/gather", `{"x":1,"extra":true}`, cookies)
		if rec.Code != http.StatusBadRequest || journal.gathers != 0 {
			t.Fatalf("status %d gathers %d", rec.Code, journal.gathers)
		}
		assertErrorCode(t, rec.Body.Bytes(), "invalid_body")
	})

	t.Run("strike publishes durability", func(t *testing.T) {
		remaining := "7"
		journal := &fakeMap{strikeResult: gather.StrikeResult{
			Cell: mapstate.Cell{X: 4, Y: 5, Layers: map[string]*string{mapstate.LayerDurability: &remaining}},
		}}
		publish := &fakePublish{}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, publish)
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/strike", `{"x":4,"y":5,"kind":"tree"}`, cookies)
		if rec.Code != http.StatusOK || journal.strikes != 1 || journal.kind != "tree" {
			t.Fatalf("status %d strikes %d kind %s body %s", rec.Code, journal.strikes, journal.kind, rec.Body.String())
		}
		var body strikeBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Broken || body.Capacity != 0 || body.Slots != nil {
			t.Fatalf("body %+v", body)
		}
		if body.Cell.Layers["durability"] == nil || *body.Cell.Layers["durability"] != "7" {
			t.Fatalf("cell %+v", body.Cell)
		}
		if publish.calls != 1 || publish.env.Type != "map" || publish.room != "game:"+gameID {
			t.Fatalf("publish %+v", publish)
		}
	})

	t.Run("broken tree returns wood", func(t *testing.T) {
		journal := &fakeMap{strikeResult: gather.StrikeResult{
			Broken: true,
			Inventory: inventory.Inventory{
				Capacity: inventory.Capacity,
				Slots: []inventory.Slot{{
					Index: 0, Quantity: 4,
					Item: inventory.Item{ID: "44444444-4444-4444-4444-444444444444", Slug: "wood", Name: "Древесина", IconKey: "wood", MaxStack: 99},
				}},
			},
			Cell: mapstate.Cell{X: 1, Y: 1, Layers: map[string]*string{
				mapstate.LayerTree:       nil,
				mapstate.LayerDurability: nil,
			}},
		}}
		publish := &fakePublish{}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, publish)
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/strike", `{"x":1,"y":1,"kind":"tree"}`, cookies)
		if rec.Code != http.StatusOK || publish.calls != 1 {
			t.Fatalf("status %d publishes %d body %s", rec.Code, publish.calls, rec.Body.String())
		}
		var body strikeBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if !body.Broken || body.Capacity != inventory.Capacity || len(body.Slots) != 1 || body.Slots[0].Quantity != 4 || body.Slots[0].Item.Slug != "wood" {
			t.Fatalf("body %+v", body)
		}
		if body.Cell.Layers["tree"] != nil || body.Cell.Layers["durability"] != nil {
			t.Fatalf("cell %+v", body.Cell.Layers)
		}
	})

	t.Run("strike already taken does not publish", func(t *testing.T) {
		journal := &fakeMap{err: gather.ErrAlreadyTaken}
		publish := &fakePublish{}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, publish)
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/strike", `{"x":1,"y":1,"kind":"tree"}`, cookies)
		if rec.Code != http.StatusConflict || publish.calls != 0 || journal.strikes != 1 {
			t.Fatalf("status %d publishes %d strikes %d", rec.Code, publish.calls, journal.strikes)
		}
		assertErrorCode(t, rec.Body.Bytes(), "already_taken")
	})

	t.Run("strike invalid body", func(t *testing.T) {
		journal := &fakeMap{}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, &fakePublish{})
		rec := request(t, mux, http.MethodPost, "/api/games/"+gameID+"/actions/strike", `{"x":1,"extra":true}`, cookies)
		if rec.Code != http.StatusBadRequest || journal.strikes != 0 {
			t.Fatalf("status %d strikes %d", rec.Code, journal.strikes)
		}
		assertErrorCode(t, rec.Body.Bytes(), "invalid_body")
	})

	t.Run("unauthorized", func(t *testing.T) {
		journal := &fakeMap{}
		mux := http.NewServeMux()
		RegisterMap(mux, journal, &fakeAuth{current: owner}, &fakePublish{})
		rec := request(t, mux, http.MethodGet, "/api/games/"+gameID+"/map", "", nil)
		if rec.Code != http.StatusUnauthorized || journal.lists != 0 {
			t.Fatalf("status %d lists %d", rec.Code, journal.lists)
		}
		assertErrorCode(t, rec.Body.Bytes(), "unauthorized")
	})
}

type fakeMap struct {
	cells        []mapstate.Cell
	result       gather.Result
	strikeResult gather.StrikeResult
	err          error
	gameID       string
	playerID     string
	x            int
	y            int
	kind         string
	lists        int
	gathers      int
	strikes      int
}

func (f *fakeMap) List(_ context.Context, gameID string) ([]mapstate.Cell, error) {
	f.lists++
	f.gameID = gameID
	if f.err != nil {
		return nil, f.err
	}
	if f.cells == nil {
		return []mapstate.Cell{}, nil
	}
	return f.cells, nil
}

func (f *fakeMap) Gather(_ context.Context, gameID, playerID string, x, y int, kind string) (gather.Result, error) {
	f.gathers++
	f.gameID = gameID
	f.playerID = playerID
	f.x = x
	f.y = y
	f.kind = kind
	if f.err != nil {
		return gather.Result{}, f.err
	}
	return f.result, nil
}

func (f *fakeMap) Strike(_ context.Context, gameID, playerID string, x, y int, kind string) (gather.StrikeResult, error) {
	f.strikes++
	f.gameID = gameID
	f.playerID = playerID
	f.x = x
	f.y = y
	f.kind = kind
	if f.err != nil {
		return gather.StrikeResult{}, f.err
	}
	return f.strikeResult, nil
}

type fakePublish struct {
	room   string
	env    realtime.Envelope
	except *realtime.Client
	calls  int
}

func (f *fakePublish) Publish(room string, env realtime.Envelope, except *realtime.Client) {
	f.calls++
	f.room = room
	f.env = env
	f.except = except
}
