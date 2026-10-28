package inventory

import (
	"context"

	"game/backend/internal/game"
)

type Slots interface {
	List(ctx context.Context, gameID, playerID string) ([]Slot, error)
}

type Games interface {
	Find(ctx context.Context, id string) (game.Game, error)
}

type Service struct {
	slots Slots
	games Games
}

func NewService(slots Slots, games Games) *Service {
	return &Service{slots: slots, games: games}
}

func (s *Service) List(ctx context.Context, gameID, playerID string) (Inventory, error) {
	if _, err := s.games.Find(ctx, gameID); err != nil {
		return Inventory{}, err
	}
	slots, err := s.slots.List(ctx, gameID, playerID)
	if err != nil {
		return Inventory{}, err
	}
	if slots == nil {
		slots = []Slot{}
	}
	return Inventory{Capacity: Capacity, Slots: slots}, nil
}
