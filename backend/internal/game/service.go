package game

import "context"

type Store interface {
	Create(ctx context.Context, ownerID, name string, worldSeed int64) (Game, error)
	FindByID(ctx context.Context, id string) (Game, error)
	ListByOwner(ctx context.Context, ownerID string) ([]Game, error)
	UpdateName(ctx context.Context, id, name string) (Game, error)
}

type Service struct {
	store   Store
	newSeed func() (int64, error)
}

func NewService(store Store) *Service {
	return &Service{store: store, newSeed: RandomWorldSeed}
}

func (s *Service) Create(ctx context.Context, ownerID, name string, worldSeed *int64) (Game, error) {
	normalized, err := NormalizeName(name)
	if err != nil {
		return Game{}, err
	}
	seed, err := s.resolveSeed(worldSeed)
	if err != nil {
		return Game{}, err
	}
	return s.store.Create(ctx, ownerID, normalized, seed)
}

func (s *Service) resolveSeed(worldSeed *int64) (int64, error) {
	if worldSeed == nil {
		return s.newSeed()
	}
	if !ValidWorldSeed(*worldSeed) {
		return 0, ErrInvalidSeed
	}
	return *worldSeed, nil
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
