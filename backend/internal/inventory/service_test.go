package inventory

import (
	"context"
	"errors"
	"testing"

	"game/backend/internal/game"
)

const (
	gameID   = "33333333-3333-3333-3333-333333333333"
	playerID = "11111111-1111-1111-1111-111111111111"
)

func TestListMissingGame(t *testing.T) {
	slots := &fakeSlots{}
	_, err := NewService(slots, &fakeGames{err: game.ErrNotFound}).List(context.Background(), gameID, playerID)
	if err != game.ErrNotFound {
		t.Fatalf("err %v", err)
	}
	if slots.lists != 0 {
		t.Fatalf("lists %d", slots.lists)
	}
}

func TestListEmptyBag(t *testing.T) {
	bag, err := NewService(&fakeSlots{}, &fakeGames{}).List(context.Background(), gameID, playerID)
	if err != nil {
		t.Fatal(err)
	}
	if bag.Capacity != Capacity || bag.Slots == nil || len(bag.Slots) != 0 {
		t.Fatalf("bag %+v", bag)
	}
}

func TestListReturnsSlots(t *testing.T) {
	stored := []Slot{{
		Index:    3,
		Quantity: 2,
		Item:     Item{ID: "44444444-4444-4444-4444-444444444444", Slug: "wood", Name: "Дерево", IconKey: "item-wood", MaxStack: 20},
	}}
	slots := &fakeSlots{slots: stored}
	bag, err := NewService(slots, &fakeGames{}).List(context.Background(), gameID, playerID)
	if err != nil {
		t.Fatal(err)
	}
	if slots.gameID != gameID || slots.playerID != playerID {
		t.Fatalf("query game %s player %s", slots.gameID, slots.playerID)
	}
	if bag.Capacity != Capacity || len(bag.Slots) != 1 || bag.Slots[0].Index != 3 || bag.Slots[0].Item.Slug != "wood" {
		t.Fatalf("bag %+v", bag)
	}
}

type fakeGames struct {
	err error
}

func (f *fakeGames) Find(context.Context, string) (game.Game, error) {
	if f.err != nil {
		return game.Game{}, f.err
	}
	return game.Game{ID: gameID}, nil
}

type fakeSlots struct {
	slots    []Slot
	err      error
	gameID   string
	playerID string
	lists    int
}

func (f *fakeSlots) List(_ context.Context, gameID, playerID string) ([]Slot, error) {
	f.lists++
	f.gameID = gameID
	f.playerID = playerID
	if f.err != nil {
		return nil, f.err
	}
	return f.slots, nil
}

func TestListSlotError(t *testing.T) {
	boom := errors.New("db down")
	_, err := NewService(&fakeSlots{err: boom}, &fakeGames{}).List(context.Background(), gameID, playerID)
	if !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
}
