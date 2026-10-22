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
)

type GameService interface {
	Create(ctx context.Context, ownerID, name string) (game.Game, error)
	Find(ctx context.Context, id string) (game.Game, error)
	ListOwned(ctx context.Context, ownerID string) ([]game.Game, error)
	Rename(ctx context.Context, actorID, gameID, name string) (game.Game, error)
}

type PlayerSession interface {
	Current(ctx context.Context, accessToken string) (player.Player, error)
}

type gameBody struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Owner     playerBody `json:"owner"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type gameListBody struct {
	Games []gameBody `json:"games"`
}

type nameBody struct {
	Name string `json:"name"`
}

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

func (h *gameHandler) create(w http.ResponseWriter, r *http.Request) {
	current, ok := h.currentPlayer(w, r)
	if !ok {
		return
	}
	body, err := readName(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	created, err := h.games.Create(r.Context(), current.ID, body.Name)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toGameBody(created))
}

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
	raw := cookieValue(r, accessCookie)
	if raw == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return player.Player{}, false
	}
	current, err := h.sessions.Current(r.Context(), raw)
	if err != nil {
		writeServiceError(w, r, err)
		return player.Player{}, false
	}
	return current, true
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

func toGameBody(item game.Game) gameBody {
	return gameBody{
		ID:   item.ID,
		Name: item.Name,
		Owner: playerBody{
			ID:    item.OwnerID,
			Login: item.OwnerLogin,
		},
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
