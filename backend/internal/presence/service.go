package presence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"
	"sync"

	"game/backend/internal/game"
	"game/backend/internal/realtime"
)

const roomPrefix = "game:"

var (
	ErrInvalidMessage = errors.New("invalid message")
	ErrNotInRoom      = errors.New("not in room")
)

// Games is the lookup the presence bus needs before letting someone into a world.
type Games interface {
	Find(ctx context.Context, id string) (game.Game, error)
}

type occupant struct {
	client *realtime.Client
	x      float64
	y      float64
	color  int
}

// Service tracks who is inside a game and forwards their positions.
// Positions live in memory and disappear when the connection closes.
type Service struct {
	hub       *realtime.Hub
	games     Games
	mu        sync.Mutex
	occupants map[string]map[string]*occupant
}

func New(hub *realtime.Hub, games Games) *Service {
	return &Service{
		hub:       hub,
		games:     games,
		occupants: map[string]map[string]*occupant{},
	}
}

func (s *Service) Register(router *realtime.Router) {
	router.Handle("subscribe", s.subscribe)
	router.Handle("position", s.position)
}

func (s *Service) subscribe(ctx context.Context, client *realtime.Client, env realtime.Envelope) error {
	gameID, err := gameID(env.Room)
	if err != nil {
		return s.fail(client, err)
	}
	if _, err := s.games.Find(ctx, gameID); err != nil {
		return s.fail(client, err)
	}
	point, err := decodePoint(env.Payload)
	if err != nil {
		return s.fail(client, err)
	}

	s.mu.Lock()
	others, color, fresh := s.admit(client, env.Room, point)
	s.mu.Unlock()

	s.send(client, "snapshot", env.Room, snapshotBody{Players: others, Color: color})
	kind := "moved"
	if fresh {
		kind = "joined"
	}
	s.publish(env.Room, kind, playerBody{
		PlayerID: client.PlayerID(),
		Login:    client.Login(),
		X:        point.x,
		Y:        point.y,
		Color:    color,
	}, client)
	return nil
}

func (s *Service) position(_ context.Context, client *realtime.Client, env realtime.Envelope) error {
	if _, err := gameID(env.Room); err != nil {
		return s.fail(client, err)
	}
	point, err := decodePoint(env.Payload)
	if err != nil {
		return s.fail(client, err)
	}

	s.mu.Lock()
	current := s.occupants[env.Room][client.PlayerID()]
	if current == nil || current.client != client {
		s.mu.Unlock()
		return s.fail(client, ErrNotInRoom)
	}
	current.x = point.x
	current.y = point.y
	body := playerBody{
		PlayerID: client.PlayerID(),
		Login:    client.Login(),
		X:        point.x,
		Y:        point.y,
		Color:    current.color,
	}
	s.mu.Unlock()

	s.publish(env.Room, "moved", body, client)
	return nil
}

// Disconnect drops the connection from every game and tells the others.
func (s *Service) Disconnect(client *realtime.Client) {
	s.mu.Lock()
	rooms := make([]string, 0, 1)
	for room, occupants := range s.occupants {
		current := occupants[client.PlayerID()]
		if current == nil || current.client != client {
			continue
		}
		delete(occupants, client.PlayerID())
		if len(occupants) == 0 {
			delete(s.occupants, room)
		}
		rooms = append(rooms, room)
	}
	s.mu.Unlock()

	client.LeaveAll()
	for _, room := range rooms {
		s.publish(room, "left", map[string]string{"playerId": client.PlayerID()}, nil)
	}
}

func (s *Service) admit(client *realtime.Client, room string, point point) ([]playerBody, int, bool) {
	occupants := s.occupants[room]
	if occupants == nil {
		occupants = map[string]*occupant{}
		s.occupants[room] = occupants
	}
	previous := occupants[client.PlayerID()]
	fresh := previous == nil
	if previous != nil && previous.client != client {
		previous.client.Leave(room)
		s.send(previous.client, "error", "", map[string]string{"code": "replaced"})
	}
	color := pickColor(colorsInUse(occupants, client.PlayerID()))
	if previous != nil && previous.client == client {
		color = previous.color
	}
	client.Join(room)
	occupants[client.PlayerID()] = &occupant{client: client, x: point.x, y: point.y, color: color}

	others := make([]playerBody, 0, len(occupants))
	for id, item := range occupants {
		if id == client.PlayerID() {
			continue
		}
		others = append(others, playerBody{
			PlayerID: id,
			Login:    item.client.Login(),
			X:        item.x,
			Y:        item.y,
			Color:    item.color,
		})
	}
	return others, color, fresh
}

func colorsInUse(occupants map[string]*occupant, except string) map[int]struct{} {
	used := make(map[int]struct{}, len(occupants))
	for id, item := range occupants {
		if id == except || item.color == 0 {
			continue
		}
		used[item.color] = struct{}{}
	}
	return used
}

func (s *Service) fail(client *realtime.Client, err error) error {
	code := "invalid_message"
	if errors.Is(err, ErrNotInRoom) {
		code = "not_in_room"
	}
	if errors.Is(err, game.ErrNotFound) {
		code = "game_not_found"
	}
	s.send(client, "error", "", map[string]string{"code": code})
	return nil
}

func (s *Service) publish(room, typ string, payload any, except *realtime.Client) {
	env, err := realtime.NewEnvelope(typ, room, payload)
	if err != nil {
		return
	}
	s.hub.Publish(room, env, except)
}

func (s *Service) send(client *realtime.Client, typ, room string, payload any) {
	env, err := realtime.NewEnvelope(typ, room, payload)
	if err != nil {
		return
	}
	client.Send(env)
}

type playerBody struct {
	PlayerID string  `json:"playerId"`
	Login    string  `json:"login"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Color    int     `json:"color"`
}

type snapshotBody struct {
	Players []playerBody `json:"players"`
	Color   int          `json:"color"`
}

type point struct {
	x float64
	y float64
}

func gameID(room string) (string, error) {
	id, ok := strings.CutPrefix(room, roomPrefix)
	if !ok || !game.ValidID(id) {
		return "", ErrInvalidMessage
	}
	return id, nil
}

func decodePoint(raw json.RawMessage) (point, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return point{}, ErrInvalidMessage
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var body struct {
		X *float64 `json:"x"`
		Y *float64 `json:"y"`
	}
	if err := decoder.Decode(&body); err != nil {
		return point{}, ErrInvalidMessage
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return point{}, ErrInvalidMessage
	}
	if body.X == nil || body.Y == nil || !validCoord(*body.X) || !validCoord(*body.Y) {
		return point{}, ErrInvalidMessage
	}
	return point{x: *body.X, y: *body.Y}, nil
}

func validCoord(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value) <= 1e7
}
