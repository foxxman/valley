package presence

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"game/backend/internal/game"
	"game/backend/internal/realtime"
)

const sampleGameID = "33333333-3333-3333-3333-333333333333"

func TestPresenceTwoPlayers(t *testing.T) {
	svc, fox, owl := newRoom(t)

	subscribe(t, svc, fox, 10, 20)
	foxSnapshot := decodeSnapshot(t, take(t, fox))
	if len(foxSnapshot.Players) != 0 || foxSnapshot.Color == 0 {
		t.Fatalf("snapshot %+v", foxSnapshot)
	}

	subscribe(t, svc, owl, 30, 40)
	joinedEnv := take(t, fox)
	joined := decodePlayer(t, joinedEnv)
	if joinedEnv.Type != "joined" || joined.Login != "owl" || joined.Color == 0 || joined.Color == foxSnapshot.Color {
		t.Fatalf("joined %+v", joined)
	}
	owlSnapshot := decodeSnapshot(t, take(t, owl))
	if len(owlSnapshot.Players) != 1 || owlSnapshot.Players[0].PlayerID != fox.PlayerID() {
		t.Fatalf("owl snapshot %+v", owlSnapshot.Players)
	}
	if owlSnapshot.Color != joined.Color || owlSnapshot.Players[0].Color != foxSnapshot.Color {
		t.Fatalf("colors self %d joined %d fox %d seen %d", owlSnapshot.Color, joined.Color, foxSnapshot.Color, owlSnapshot.Players[0].Color)
	}

	if err := svc.position(context.Background(), owl, pointMessage(35, 45)); err != nil {
		t.Fatal(err)
	}
	moved := decodePlayer(t, take(t, fox))
	if moved.X != 35 || moved.Y != 45 || moved.PlayerID != owl.PlayerID() || moved.Color != owlSnapshot.Color {
		t.Fatalf("moved %+v", moved)
	}
	select {
	case env := <-owl.Events():
		t.Fatalf("echoed to sender %+v", env)
	default:
	}

	svc.Disconnect(owl)
	left := take(t, fox)
	if left.Type != "left" || decodeLeft(t, left) != owl.PlayerID() {
		t.Fatalf("left %+v", left)
	}
}

func TestPresenceRejectsUnknownGameAndBadPoint(t *testing.T) {
	svc, fox, _ := newRoom(t)
	env := pointMessage(1, 2)
	env.Room = "game:44444444-4444-4444-4444-444444444444"
	if err := svc.subscribe(context.Background(), fox, env); err != nil {
		t.Fatal(err)
	}
	if code := decodeCode(t, take(t, fox)); code != "game_not_found" {
		t.Fatalf("code %s", code)
	}

	env = pointMessage(1, 2)
	env.Room = "nope"
	if err := svc.subscribe(context.Background(), fox, env); err != nil {
		t.Fatal(err)
	}
	if code := decodeCode(t, take(t, fox)); code != "invalid_message" {
		t.Fatalf("code %s", code)
	}

	env = realtime.Envelope{Type: "subscribe", Room: room()}
	if err := svc.subscribe(context.Background(), fox, env); err != nil {
		t.Fatal(err)
	}
	if code := decodeCode(t, take(t, fox)); code != "invalid_message" {
		t.Fatalf("code %s", code)
	}
}

func TestPresenceReplacesSamePlayer(t *testing.T) {
	svc, fox, owl := newRoom(t)
	subscribe(t, svc, fox, 1, 1)
	take(t, fox)
	subscribe(t, svc, owl, 2, 2)
	take(t, fox)
	take(t, owl)

	again := svc.hub.Add(fox.PlayerID(), fox.Login())
	subscribe(t, svc, again, 8, 9)
	if code := decodeCode(t, take(t, fox)); code != "replaced" {
		t.Fatalf("code %s", code)
	}
	moved := decodePlayer(t, take(t, owl))
	againSnapshot := decodeSnapshot(t, take(t, again))
	if moved.X != 8 || moved.PlayerID != fox.PlayerID() || moved.Color == 0 || moved.Color != againSnapshot.Color {
		t.Fatalf("moved %+v snapshot %+v", moved, againSnapshot)
	}

	if err := svc.position(context.Background(), fox, pointMessage(3, 3)); err != nil {
		t.Fatal(err)
	}
	if code := decodeCode(t, take(t, fox)); code != "not_in_room" {
		t.Fatalf("code %s", code)
	}

	svc.Disconnect(fox)
	if err := svc.position(context.Background(), again, pointMessage(9, 9)); err != nil {
		t.Fatal(err)
	}
	movedEnv := take(t, owl)
	if movedEnv.Type != "moved" {
		t.Fatalf("type %s payload %s", movedEnv.Type, movedEnv.Payload)
	}
	moved = decodePlayer(t, movedEnv)
	if moved.X != 9 {
		t.Fatalf("still in room %+v", moved)
	}
}

type storedGame struct{}

func (storedGame) Find(_ context.Context, id string) (game.Game, error) {
	if id != sampleGameID {
		return game.Game{}, game.ErrNotFound
	}
	return game.Game{ID: id, Name: "Долина", WorldSeed: 42}, nil
}

func newRoom(t *testing.T) (*Service, *realtime.Client, *realtime.Client) {
	t.Helper()
	hub := realtime.NewHub()
	svc := New(hub, storedGame{})
	return svc, hub.Add("fox-id", "fox"), hub.Add("owl-id", "owl")
}

func room() string {
	return roomPrefix + sampleGameID
}

func subscribe(t *testing.T, svc *Service, client *realtime.Client, x, y float64) {
	t.Helper()
	if err := svc.subscribe(context.Background(), client, pointMessage(x, y)); err != nil {
		t.Fatal(err)
	}
}

func pointMessage(x, y float64) realtime.Envelope {
	env, err := realtime.NewEnvelope("subscribe", room(), map[string]float64{"x": x, "y": y})
	if err != nil {
		panic(err)
	}
	return env
}

func take(t *testing.T, client *realtime.Client) realtime.Envelope {
	t.Helper()
	select {
	case env := <-client.Events():
		return env
	case <-time.After(time.Second):
		t.Fatal("no message")
	}
	return realtime.Envelope{}
}

func decodeSnapshot(t *testing.T, env realtime.Envelope) snapshotBody {
	t.Helper()
	var body snapshotBody
	if err := json.Unmarshal(env.Payload, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func decodePlayer(t *testing.T, env realtime.Envelope) playerBody {
	t.Helper()
	var body playerBody
	if err := json.Unmarshal(env.Payload, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func decodeLeft(t *testing.T, env realtime.Envelope) string {
	t.Helper()
	var body struct {
		PlayerID string `json:"playerId"`
	}
	if err := json.Unmarshal(env.Payload, &body); err != nil {
		t.Fatal(err)
	}
	return body.PlayerID
}

func decodeCode(t *testing.T, env realtime.Envelope) string {
	t.Helper()
	if env.Type != "error" {
		t.Fatalf("type %s", env.Type)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(env.Payload, &body); err != nil {
		t.Fatal(err)
	}
	return body.Code
}
