package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

const (
	readLimit    = 4096
	writeTimeout = 5 * time.Second
	pingInterval = 20 * time.Second
)

// Serve upgrades the request and runs the connection until it closes.
// onDisconnect runs before the send channel is closed.
func Serve(
	w http.ResponseWriter,
	r *http.Request,
	playerID, login string,
	hub *Hub,
	router *Router,
	onDisconnect func(*Client),
) error {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// The dev UI is proxied from another port, so the browser Origin
		// does not match this host. The session cookie authorizes the socket.
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionDisabled,
	})
	if err != nil {
		return err
	}
	conn.SetReadLimit(readLimit)

	client := hub.Add(playerID, login)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer func() {
		if onDisconnect != nil {
			onDisconnect(client)
		}
		client.closeSend()
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	go func() {
		writeLoop(ctx, conn, client.events)
		cancel()
	}()
	readLoop(ctx, conn, client, router)
	return nil
}

func readLoop(ctx context.Context, conn *websocket.Conn, client *Client, router *Router) {
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			client.Send(errorEnvelope("invalid_message"))
			continue
		}
		if err := router.Dispatch(ctx, client, env); err != nil {
			code := "internal"
			if errors.Is(err, ErrUnknownType) {
				code = "unknown_type"
			} else {
				slog.Error("realtime dispatch", "err", err, "type", env.Type)
			}
			client.Send(errorEnvelope(code))
		}
	}
}

func writeLoop(ctx context.Context, conn *websocket.Conn, events <-chan Envelope) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		case env, ok := <-events:
			if !ok {
				return
			}
			body, err := json.Marshal(env)
			if err != nil {
				continue
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err = conn.Write(writeCtx, websocket.MessageText, body)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func errorEnvelope(code string) Envelope {
	env, err := NewEnvelope("error", "", map[string]string{"code": code})
	if err != nil {
		return Envelope{Type: "error"}
	}
	return env
}
