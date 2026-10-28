package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"game/backend/internal/gather"
	"game/backend/internal/mapstate"
	"game/backend/internal/presence"
	"game/backend/internal/realtime"
)

type MapService interface {
	List(ctx context.Context, gameID string) ([]mapstate.Cell, error)
	Gather(ctx context.Context, gameID, playerID string, x, y int, kind string) (gather.Result, error)
	Strike(ctx context.Context, gameID, playerID string, x, y int, kind string) (gather.StrikeResult, error)
}

type mapPublisher interface {
	Publish(room string, env realtime.Envelope, except *realtime.Client)
}

type mapBody struct {
	Cells []cellBody `json:"cells"`
} //@name mapJournal

type cellBody struct {
	X      int                `json:"x" example:"1"`
	Y      int                `json:"y" example:"2"`
	Layers map[string]*string `json:"layers"`
} //@name mapCell

type gatherBody struct {
	Capacity int                 `json:"capacity" example:"20"`
	Slots    []inventorySlotBody `json:"slots"`
	Cell     cellBody            `json:"cell"`
} //@name gatherResult

type gatherRequest struct {
	X    int    `json:"x" example:"4"`
	Y    int    `json:"y" example:"5"`
	Kind string `json:"kind" example:"flower-0"`
} //@name gatherRequest

type strikeRequest struct {
	X    int    `json:"x" example:"4"`
	Y    int    `json:"y" example:"5"`
	Kind string `json:"kind" example:"tree"`
} //@name strikeRequest

type strikeBody struct {
	Broken   bool                `json:"broken" example:"false"`
	Capacity int                 `json:"capacity,omitempty" example:"20"`
	Slots    []inventorySlotBody `json:"slots,omitempty"`
	Cell     cellBody            `json:"cell"`
} //@name strikeResult

func RegisterMap(mux *http.ServeMux, journal MapService, sessions PlayerSession, publish mapPublisher) {
	h := &mapHandler{journal: journal, sessions: sessions, publish: publish}
	mux.HandleFunc("GET /api/games/{id}/map", h.list)
	mux.HandleFunc("POST /api/games/{id}/actions/gather", h.gather)
	mux.HandleFunc("POST /api/games/{id}/actions/strike", h.strike)
}

type mapHandler struct {
	journal  MapService
	sessions PlayerSession
	publish  mapPublisher
}

// list возвращает журнал текущих изменений карты.
//
//	@Summary	Журнал карты
//	@Tags		map
//	@Produce	json
//	@Security	AccessCookie
//	@Param		id	path		string	true	"ID игры"
//	@Success	200	{object}	mapBody
//	@Failure	401	{object}	errorBody	"unauthorized"
//	@Failure	404	{object}	errorBody	"game_not_found"
//	@Failure	500	{object}	errorBody
//	@Router		/api/games/{id}/map [get]
func (h *mapHandler) list(w http.ResponseWriter, r *http.Request) {
	if _, ok := authenticate(w, r, h.sessions); !ok {
		return
	}
	cells, err := h.journal.List(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toMapBody(cells))
}

// gather срывает цветок в досягаемости и кладёт его в рюкзак.
//
//	@Summary	Собрать цветок
//	@Tags		map
//	@Accept		json
//	@Produce	json
//	@Security	AccessCookie
//	@Param		id		path		string			true	"ID игры"
//	@Param		body	body		gatherRequest	true	"Клетка и вид цветка"
//	@Success	200		{object}	gatherBody
//	@Failure	400		{object}	errorBody	"invalid_body, invalid_kind или not_in_reach"
//	@Failure	401		{object}	errorBody	"unauthorized"
//	@Failure	404		{object}	errorBody	"game_not_found"
//	@Failure	409		{object}	errorBody	"already_taken или bag_full"
//	@Failure	500		{object}	errorBody
//	@Router		/api/games/{id}/actions/gather [post]
func (h *mapHandler) gather(w http.ResponseWriter, r *http.Request) {
	current, ok := authenticate(w, r, h.sessions)
	if !ok {
		return
	}
	body, err := readGather(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	gameID := r.PathValue("id")
	result, err := h.journal.Gather(r.Context(), gameID, current.ID, body.X, body.Y, body.Kind)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	cell := toCellBody(result.Cell)
	room := presence.Room(gameID)
	if env, err := realtime.NewEnvelope("map", room, cell); err == nil {
		h.publish.Publish(room, env, nil)
	}
	bag := toInventoryBody(result.Inventory)
	writeJSON(w, http.StatusOK, gatherBody{
		Capacity: bag.Capacity,
		Slots:    bag.Slots,
		Cell:     cell,
	})
}

// strike наносит один удар рукой по разрушаемому объекту в досягаемости.
//
//	@Summary	Ударить по объекту
//	@Tags		map
//	@Accept		json
//	@Produce	json
//	@Security	AccessCookie
//	@Param		id		path		string			true	"ID игры"
//	@Param		body	body		strikeRequest	true	"Клетка и вид объекта"
//	@Success	200		{object}	strikeBody
//	@Failure	400		{object}	errorBody	"invalid_body, invalid_kind или not_in_reach"
//	@Failure	401		{object}	errorBody	"unauthorized"
//	@Failure	404		{object}	errorBody	"game_not_found"
//	@Failure	409		{object}	errorBody	"already_taken или bag_full"
//	@Failure	500		{object}	errorBody
//	@Router		/api/games/{id}/actions/strike [post]
func (h *mapHandler) strike(w http.ResponseWriter, r *http.Request) {
	current, ok := authenticate(w, r, h.sessions)
	if !ok {
		return
	}
	var body strikeRequest
	if err := decodeBody(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	gameID := r.PathValue("id")
	result, err := h.journal.Strike(r.Context(), gameID, current.ID, body.X, body.Y, body.Kind)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	cell := toCellBody(result.Cell)
	room := presence.Room(gameID)
	if env, err := realtime.NewEnvelope("map", room, cell); err == nil {
		h.publish.Publish(room, env, nil)
	}
	response := strikeBody{Broken: result.Broken, Cell: cell}
	if result.Broken {
		bag := toInventoryBody(result.Inventory)
		response.Capacity = bag.Capacity
		response.Slots = bag.Slots
	}
	writeJSON(w, http.StatusOK, response)
}

func readGather(w http.ResponseWriter, r *http.Request) (gatherRequest, error) {
	var body gatherRequest
	if err := decodeBody(w, r, &body); err != nil {
		return gatherRequest{}, err
	}
	return body, nil
}

func decodeBody(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	err := decoder.Decode(&struct{}{})
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("trailing json")
	}
	return err
}

func toMapBody(cells []mapstate.Cell) mapBody {
	body := mapBody{Cells: make([]cellBody, 0, len(cells))}
	for _, cell := range cells {
		body.Cells = append(body.Cells, toCellBody(cell))
	}
	return body
}

func toCellBody(cell mapstate.Cell) cellBody {
	layers := make(map[string]*string, len(cell.Layers))
	for key, value := range cell.Layers {
		layers[key] = value
	}
	return cellBody{X: cell.X, Y: cell.Y, Layers: layers}
}
