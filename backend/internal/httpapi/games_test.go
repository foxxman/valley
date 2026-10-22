package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"game/backend/internal/game"
	"game/backend/internal/player"
)

func TestGameHandlers(t *testing.T) {
	owner := player.Player{ID: "11111111-1111-1111-1111-111111111111", Login: "fox"}
	item := game.Game{
		ID:         "33333333-3333-3333-3333-333333333333",
		Name:       "Долина",
		OwnerID:    owner.ID,
		OwnerLogin: owner.Login,
		CreatedAt:  time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
	}
	svc := &fakeGames{created: item, found: item, owned: []game.Game{item}, renamed: item}
	sessions := &fakeAuth{current: owner}
	mux := http.NewServeMux()
	RegisterGames(mux, svc, sessions)
	cookies := map[string]string{accessCookie: "access-1"}

	created := request(t, mux, http.MethodPost, "/api/games", `{"name":"Долина"}`, cookies)
	if created.Code != http.StatusCreated || svc.gotOwner != owner.ID || svc.gotName != "Долина" {
		t.Fatalf("create %d owner %s name %s body %s", created.Code, svc.gotOwner, svc.gotName, created.Body.String())
	}
	assertGame(t, created.Body.Bytes(), item.ID, "Долина", owner.ID)

	listed := request(t, mux, http.MethodGet, "/api/games", "", cookies)
	if listed.Code != http.StatusOK || svc.gotOwner != owner.ID {
		t.Fatalf("list %d %s", listed.Code, listed.Body.String())
	}
	var list gameListBody
	if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Games) != 1 || list.Games[0].ID != item.ID {
		t.Fatalf("list %+v", list.Games)
	}

	found := request(t, mux, http.MethodGet, "/api/games/"+item.ID, "", cookies)
	if found.Code != http.StatusOK {
		t.Fatalf("find %d %s", found.Code, found.Body.String())
	}
	assertGame(t, found.Body.Bytes(), item.ID, "Долина", owner.ID)

	renamed := request(t, mux, http.MethodPatch, "/api/games/"+item.ID, `{"name":"Новая"}`, cookies)
	if renamed.Code != http.StatusOK || svc.gotActor != owner.ID || svc.gotName != "Новая" {
		t.Fatalf("rename %d actor %s name %s", renamed.Code, svc.gotActor, svc.gotName)
	}
}

func TestGameHandlerErrors(t *testing.T) {
	owner := player.Player{ID: "11111111-1111-1111-1111-111111111111", Login: "fox"}
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		cookie bool
		svc    *fakeGames
		status int
		code   string
	}{
		{
			name:   "no session",
			method: http.MethodGet,
			path:   "/api/games",
			svc:    &fakeGames{},
			status: http.StatusUnauthorized,
			code:   "unauthorized",
		},
		{
			name:   "blank name",
			method: http.MethodPost,
			path:   "/api/games",
			body:   `{"name":" "}`,
			cookie: true,
			svc:    &fakeGames{createErr: game.ErrInvalidName},
			status: http.StatusBadRequest,
			code:   "invalid_name",
		},
		{
			name:   "missing game",
			method: http.MethodGet,
			path:   "/api/games/33333333-3333-3333-3333-333333333333",
			cookie: true,
			svc:    &fakeGames{findErr: game.ErrNotFound},
			status: http.StatusNotFound,
			code:   "game_not_found",
		},
		{
			name:   "stranger rename",
			method: http.MethodPatch,
			path:   "/api/games/33333333-3333-3333-3333-333333333333",
			body:   `{"name":"Чужая"}`,
			cookie: true,
			svc:    &fakeGames{renameErr: game.ErrForbidden},
			status: http.StatusForbidden,
			code:   "forbidden",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			RegisterGames(mux, test.svc, &fakeAuth{current: owner})
			var cookies map[string]string
			if test.cookie {
				cookies = map[string]string{accessCookie: "access-1"}
			}
			rec := request(t, mux, test.method, test.path, test.body, cookies)
			if rec.Code != test.status {
				t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
			}
			assertErrorCode(t, rec.Body.Bytes(), test.code)
		})
	}
}

func assertGame(t *testing.T, body []byte, id, name, ownerID string) {
	t.Helper()
	var got gameBody
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != id || got.Name != name || got.Owner.ID != ownerID || got.Owner.Login == "" {
		t.Fatalf("game %+v", got)
	}
}

type fakeGames struct {
	created   game.Game
	createErr error
	found     game.Game
	findErr   error
	owned     []game.Game
	renamed   game.Game
	renameErr error

	gotOwner string
	gotActor string
	gotName  string
	gotID    string
}

func (f *fakeGames) Create(_ context.Context, ownerID, name string) (game.Game, error) {
	f.gotOwner = ownerID
	f.gotName = name
	return f.created, f.createErr
}

func (f *fakeGames) Find(_ context.Context, id string) (game.Game, error) {
	f.gotID = id
	return f.found, f.findErr
}

func (f *fakeGames) ListOwned(_ context.Context, ownerID string) ([]game.Game, error) {
	f.gotOwner = ownerID
	if f.owned == nil {
		return []game.Game{}, nil
	}
	return f.owned, nil
}

func (f *fakeGames) Rename(_ context.Context, actorID, gameID, name string) (game.Game, error) {
	f.gotActor = actorID
	f.gotID = gameID
	f.gotName = name
	return f.renamed, f.renameErr
}

var _ PlayerSession = (*fakeAuth)(nil)
var _ GameService = (*fakeGames)(nil)
var _ AuthService = (*fakeAuth)(nil)
