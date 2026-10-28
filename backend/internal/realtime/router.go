package realtime

import (
	"context"
	"errors"
)

// ErrUnknownType is returned when no handler is registered for Envelope.Type.
var ErrUnknownType = errors.New("unknown message type")

// Handler handles one message type. Game movement and chat each register their own.
type Handler func(ctx context.Context, client *Client, env Envelope) error

// Router dispatches incoming envelopes by type.
type Router struct {
	handlers map[string]Handler
}

func NewRouter() *Router {
	return &Router{handlers: map[string]Handler{}}
}

func (r *Router) Handle(typ string, handler Handler) {
	r.handlers[typ] = handler
}

func (r *Router) Dispatch(ctx context.Context, client *Client, env Envelope) error {
	handler, ok := r.handlers[env.Type]
	if !ok {
		return ErrUnknownType
	}
	return handler(ctx, client, env)
}
