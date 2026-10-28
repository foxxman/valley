package realtime

import (
	"context"
	"testing"
	"time"
)

func TestHubDeliversToRoom(t *testing.T) {
	hub := NewHub()
	fox := hub.Add("fox", "fox")
	owl := hub.Add("owl", "owl")
	fox.Join("game:1")
	owl.Join("game:1")

	env, err := NewEnvelope("ping", "game:1", map[string]int{"n": 1})
	if err != nil {
		t.Fatal(err)
	}
	hub.Publish("game:1", env, fox)

	select {
	case got := <-owl.Events():
		if got.Type != "ping" || got.Room != "game:1" {
			t.Fatalf("got %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("owl got nothing")
	}
	select {
	case got := <-fox.Events():
		t.Fatalf("sender received %+v", got)
	default:
	}

	owl.Leave("game:1")
	hub.Publish("game:1", env, nil)
	select {
	case got := <-owl.Events():
		t.Fatalf("left member received %+v", got)
	default:
	}
}

func TestRouterUnknownType(t *testing.T) {
	router := NewRouter()
	err := router.Dispatch(context.Background(), NewHub().Add("fox", "fox"), Envelope{Type: "nope"})
	if err != ErrUnknownType {
		t.Fatalf("err %v", err)
	}
}
