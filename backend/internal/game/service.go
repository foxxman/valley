package game

import "context"

type Store interface {
	Create(ctx context.Context, ownerID, name string) (Game, error)
	FindByID(ctx context.Context, id string) (Game, error)
	ListByOwner(ctx context.Context, ownerID string) ([]Game, error)
	UpdateName(ctx context.Context, id, name string) (Game, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, ownerID, name string) (Game, error) {
	normalized, err := NormalizeName(name)
	if err != nil {
		return Game{}, err
	}
	return s.store.Create(ctx, ownerID, normalized)
}

func (s *Service) Find(ctx context.Context, id string) (Game, error) {
	if !ValidID(id) {
		return Game{}, ErrNotFound
	}
	return s.store.FindByID(ctx, id)
}

func (s *Service) ListOwned(ctx context.Context, ownerID string) ([]Game, error) {
	found, err := s.store.ListByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	if found == nil {
		return []Game{}, nil
	}
	return found, nil
}

func (s *Service) Rename(ctx context.Context, actorID, gameID, name string) (Game, error) {
	normalized, err := NormalizeName(name)
	if err != nil {
		return Game{}, err
	}
	found, err := s.Find(ctx, gameID)
	if err != nil {
		return Game{}, err
	}
	if found.OwnerID != actorID {
		return Game{}, ErrForbidden
	}
	return s.store.UpdateName(ctx, gameID, normalized)
}
