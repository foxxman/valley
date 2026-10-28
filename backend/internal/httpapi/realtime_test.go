package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"game/backend/internal/auth"
	"game/backend/internal/game"
	"game/backend/internal/player"
	"game/backend/internal/presence"
	"game/backend/internal/realtime"

	"github.com/coder/websocket"
)

func TestRealtimeRequiresSession(t *testing.T) {
	server := startRealtime(t, map[string]player.Player{}, sampleGameID)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, response, err := websocket.Dial(ctx, wsURL(server), nil)
	if err == nil {
		conn.Close(websocket.StatusNormalClosure, "")
		t.Fatal("expected unauthorized")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("status %d err %v", status, err)
	}
}

func TestRealtimeSharesPosition(t *testing.T) {
	fox := player.Player{ID: "11111111-1111-1111-1111-111111111111", Login: "fox"}
	owl := player.Player{ID: "22222222-2222-2222-2222-222222222222", Login: "owl"}
	server := startRealtime(t, map[string]player.Player{"fox-token": fox, "owl-token": owl}, sampleGameID)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	foxConn := dial(t, ctx, server, "fox-token")
	defer foxConn.Close(websocket.StatusNormalClosure, "")
	owlConn := dial(t, ctx, server, "owl-token")
	defer owlConn.Close(websocket.StatusNormalClosure, "")

	writeText(t, ctx, foxConn, `{"type":"subscribe","room":"game:`+sampleGameID+`","payload":{"x":10,"y":20}}`)
	foxSnapshot := readJSON(t, ctx, foxConn)
	if !strings.Contains(foxSnapshot, `"type":"snapshot"`) || !strings.Contains(foxSnapshot, `"players":[]`) {
		t.Fatalf("snapshot %s", foxSnapshot)
	}

	writeText(t, ctx, owlConn, `{"type":"subscribe","room":"game:`+sampleGameID+`","payload":{"x":30,"y":40}}`)
	joined := readJSON(t, ctx, foxConn)
	if !strings.Contains(joined, `"type":"joined"`) || !strings.Contains(joined, `"login":"owl"`) {
		t.Fatalf("joined %s", joined)
	}
	owlSnapshot := readJSON(t, ctx, owlConn)
	if !strings.Contains(owlSnapshot, `"login":"fox"`) {
		t.Fatalf("owl snapshot %s", owlSnapshot)
	}

	writeText(t, ctx, owlConn, `{"type":"position","room":"game:`+sampleGameID+`","payload":{"x":31,"y":41}}`)
	moved := readJSON(t, ctx, foxConn)
	if !strings.Contains(moved, `"type":"moved"`) || !strings.Contains(moved, `"x":31`) {
		t.Fatalf("moved %s", moved)
	}
}

const sampleGameID = "33333333-3333-3333-3333-333333333333"

func startRealtime(t *testing.T, sessions map[string]player.Player, id string) *httptest.Server {
	t.Helper()
	hub := realtime.NewHub()
	router := realtime.NewRouter()
	world := presence.New(hub, storedGames{id: id})
	world.Register(router)
	mux := http.NewServeMux()
	RegisterRealtime(mux, tokenSessions{byToken: sessions}, hub, router, world.Disconnect)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httptest.NewServer(LogRequests(logger, mux))
}

type tokenSessions struct {
	byToken map[string]player.Player
}

func (s tokenSessions) Current(_ context.Context, token string) (player.Player, error) {
	current, ok := s.byToken[token]
	if !ok {
		return player.Player{}, auth.ErrUnauthorized
	}
	return current, nil
}

type storedGames struct {
	id string
}

func (s storedGames) Find(_ context.Context, id string) (game.Game, error) {
	if id != s.id {
		return game.Game{}, game.ErrNotFound
	}
	return game.Game{ID: id}, nil
}

func wsURL(server *httptest.Server) string {
	return "ws" + strings.TrimPrefix(server.URL, "http") + "/api/ws"
}

func dial(t *testing.T, ctx context.Context, server *httptest.Server, token string) *websocket.Conn {
	t.Helper()
	conn, response, err := websocket.Dial(ctx, wsURL(server), &websocket.DialOptions{
		HTTPHeader: http.Header{"Cookie": []string{accessCookie + "=" + token}},
	})
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("dial %d %v", status, err)
	}
	return conn
}

func writeText(t *testing.T, ctx context.Context, conn *websocket.Conn, body string) {
	t.Helper()
	if err := conn.Write(ctx, websocket.MessageText, []byte(body)); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, ctx context.Context, conn *websocket.Conn) string {
	t.Helper()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
