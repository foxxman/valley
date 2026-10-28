package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"game/backend/internal/game"
	"game/backend/internal/player"
	"strconv"
	"strings"
)

type GameService interface {
	Create(ctx context.Context, ownerID, name string, worldSeed *int64) (game.Game, error)
	Find(ctx context.Context, id string) (game.Game, error)
	ListOwned(ctx context.Context, ownerID string) ([]game.Game, error)
	Rename(ctx context.Context, actorID, gameID, name string) (game.Game, error)
}

type PlayerSession interface {
	Current(ctx context.Context, accessToken string) (player.Player, error)
}

type gameBody struct {
	ID        string     `json:"id" example:"3f1c2a0e-7b4d-4e1a-9c2f-6d8e0a1b2c3d"`
	Name      string     `json:"name" example:"Северная долина"`
	Owner     playerBody `json:"owner"`
	WorldSeed int64      `json:"worldSeed" example:"42"`
	CreatedAt time.Time  `json:"createdAt" example:"2026-01-02T15:04:05Z"`
	UpdatedAt time.Time  `json:"updatedAt" example:"2026-01-02T15:04:05Z"`
} //@name game

type gameListBody struct {
	Games []gameBody `json:"games"`
} //@name gameList

type createGameBody struct {
	Name string `json:"name" example:"Северная долина"`
	// Необязательный сид мира. Пустое значение — сервер выбирает его сам.
	WorldSeed json.RawMessage `json:"worldSeed,omitempty" swaggertype:"integer" example:"42"`
} //@name createGame

type nameBody struct {
	Name string `json:"name" example:"Северная долина"`
} //@name gameName

func RegisterGames(mux *http.ServeMux, games GameService, sessions PlayerSession) {
	h := &gameHandler{games: games, sessions: sessions}
	mux.HandleFunc("POST /api/games", h.create)
	mux.HandleFunc("GET /api/games", h.list)
	mux.HandleFunc("GET /api/games/{id}", h.find)
	mux.HandleFunc("PATCH /api/games/{id}", h.rename)
}

type gameHandler struct {
	games    GameService
	sessions PlayerSession
}

// create создаёт игру текущего игрока.
//
//	@Summary	Создать игру
//	@Tags		games
//	@Accept		json
//	@Produce	json
//	@Security	AccessCookie
//	@Param		body	body		createGameBody	true	"Название и необязательный сид"
//	@Success	201		{object}	gameBody
//	@Failure	400		{object}	errorBody	"invalid_body, invalid_name или invalid_seed"
//	@Failure	401		{object}	errorBody	"unauthorized"
//	@Failure	500		{object}	errorBody
//	@Router		/api/games [post]
func (h *gameHandler) create(w http.ResponseWriter, r *http.Request) {
	current, ok := h.currentPlayer(w, r)
	if !ok {
		return
	}
	body, err := readCreate(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	seed, err := parseOptionalSeed(body.WorldSeed)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_seed")
		return
	}
	created, err := h.games.Create(r.Context(), current.ID, body.Name, seed)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toGameBody(created))
}

// list возвращает игры текущего игрока.
//
//	@Summary	Список своих игр
//	@Tags		games
//	@Produce	json
//	@Security	AccessCookie
//	@Success	200	{object}	gameListBody
//	@Failure	401	{object}	errorBody	"unauthorized"
//	@Failure	500	{object}	errorBody
//	@Router		/api/games [get]
func (h *gameHandler) list(w http.ResponseWriter, r *http.Request) {
	current, ok := h.currentPlayer(w, r)
	if !ok {
		return
	}
	found, err := h.games.ListOwned(r.Context(), current.ID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	body := gameListBody{Games: make([]gameBody, 0, len(found))}
	for _, item := range found {
		body.Games = append(body.Games, toGameBody(item))
	}
	writeJSON(w, http.StatusOK, body)
}

// find возвращает игру по id.
//
//	@Summary	Игра по id
//	@Tags		games
//	@Produce	json
//	@Security	AccessCookie
//	@Param		id	path		string	true	"ID игры"
//	@Success	200	{object}	gameBody
//	@Failure	401	{object}	errorBody	"unauthorized"
//	@Failure	404	{object}	errorBody	"game_not_found"
//	@Failure	500	{object}	errorBody
//	@Router		/api/games/{id} [get]
func (h *gameHandler) find(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.currentPlayer(w, r); !ok {
		return
	}
	found, err := h.games.Find(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toGameBody(found))
}

// rename меняет название игры владельца.
//
//	@Summary	Переименовать игру
//	@Tags		games
//	@Accept		json
//	@Produce	json
//	@Security	AccessCookie
//	@Param		id		path		string		true	"ID игры"
//	@Param		body	body		nameBody	true	"Новое название"
//	@Success	200		{object}	gameBody
//	@Failure	400		{object}	errorBody	"invalid_body или invalid_name"
//	@Failure	401		{object}	errorBody	"unauthorized"
//	@Failure	403		{object}	errorBody	"forbidden"
//	@Failure	404		{object}	errorBody	"game_not_found"
//	@Failure	500		{object}	errorBody
//	@Router		/api/games/{id} [patch]
func (h *gameHandler) rename(w http.ResponseWriter, r *http.Request) {
	current, ok := h.currentPlayer(w, r)
	if !ok {
		return
	}
	body, err := readName(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	updated, err := h.games.Rename(r.Context(), current.ID, r.PathValue("id"), body.Name)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toGameBody(updated))
}

func (h *gameHandler) currentPlayer(w http.ResponseWriter, r *http.Request) (player.Player, bool) {
	return authenticate(w, r, h.sessions)
}

func readName(w http.ResponseWriter, r *http.Request) (nameBody, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var body nameBody
	if err := decoder.Decode(&body); err != nil {
		return nameBody{}, err
	}
	err := decoder.Decode(&struct{}{})
	if errors.Is(err, io.EOF) {
		return body, nil
	}
	if err == nil {
		return nameBody{}, errors.New("trailing json")
	}
	return nameBody{}, err
}

func readCreate(w http.ResponseWriter, r *http.Request) (createGameBody, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var body createGameBody
	if err := decoder.Decode(&body); err != nil {
		return createGameBody{}, err
	}
	err := decoder.Decode(&struct{}{})
	if errors.Is(err, io.EOF) {
		return body, nil
	}
	if err == nil {
		return createGameBody{}, errors.New("trailing json")
	}
	return createGameBody{}, err
}

func parseOptionalSeed(raw json.RawMessage) (*int64, error) {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return nil, nil
	}
	if strings.HasPrefix(text, `"`) {
		return nil, game.ErrInvalidSeed
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var number json.Number
	if err := decoder.Decode(&number); err != nil {
		return nil, game.ErrInvalidSeed
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, game.ErrInvalidSeed
	}
	literal := number.String()
	if literal == "" || strings.ContainsAny(literal, ".eE+") || strings.HasPrefix(literal, "-") {
		return nil, game.ErrInvalidSeed
	}
	value, err := strconv.ParseInt(literal, 10, 64)
	if err != nil || !game.ValidWorldSeed(value) {
		return nil, game.ErrInvalidSeed
	}
	return &value, nil
}

func toGameBody(item game.Game) gameBody {
	return gameBody{
		ID:        item.ID,
		Name:      item.Name,
		Owner:     playerBody{ID: item.OwnerID, Login: item.OwnerLogin},
		WorldSeed: item.WorldSeed,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
