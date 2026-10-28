package realtime

import "sync"

// Hub keeps connections in rooms and delivers envelopes to them.
type Hub struct {
	mu    sync.Mutex
	rooms map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{rooms: map[string]map[*Client]struct{}{}}
}

// Client is one authenticated connection.
// Events is consumed by the socket writer; callers deliver with Send.
type Client struct {
	playerID string
	login    string
	events   chan Envelope
	hub      *Hub

	mu     sync.Mutex
	rooms  map[string]struct{}
	closed bool
}

func (h *Hub) Add(playerID, login string) *Client {
	return &Client{
		playerID: playerID,
		login:    login,
		events:   make(chan Envelope, 32),
		hub:      h,
		rooms:    map[string]struct{}{},
	}
}

func (c *Client) PlayerID() string { return c.playerID }

func (c *Client) Login() string { return c.login }

func (c *Client) Events() <-chan Envelope { return c.events }

func (c *Client) Send(env Envelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.events <- env:
	default:
	}
}

func (c *Client) closeSend() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.events)
}

func (c *Client) Join(room string) {
	if room == "" {
		return
	}
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	if _, ok := c.rooms[room]; ok {
		return
	}
	members := c.hub.rooms[room]
	if members == nil {
		members = map[*Client]struct{}{}
		c.hub.rooms[room] = members
	}
	members[c] = struct{}{}
	c.rooms[room] = struct{}{}
}

func (c *Client) Leave(room string) {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	c.leaveLocked(room)
}

func (c *Client) LeaveAll() {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	for room := range c.rooms {
		c.leaveLocked(room)
	}
}

func (c *Client) leaveLocked(room string) {
	delete(c.rooms, room)
	members := c.hub.rooms[room]
	delete(members, c)
	if len(members) == 0 {
		delete(c.hub.rooms, room)
	}
}

func (c *Client) Rooms() []string {
	c.hub.mu.Lock()
	defer c.hub.mu.Unlock()
	rooms := make([]string, 0, len(c.rooms))
	for room := range c.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// Publish delivers env to everyone in the room except except, which may be nil.
func (h *Hub) Publish(room string, env Envelope, except *Client) {
	h.mu.Lock()
	members := make([]*Client, 0, len(h.rooms[room]))
	for member := range h.rooms[room] {
		if member != except {
			members = append(members, member)
		}
	}
	h.mu.Unlock()
	for _, member := range members {
		member.Send(env)
	}
}
