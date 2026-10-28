package gather

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"

	"game/backend/internal/game"
	"game/backend/internal/inventory"
	"game/backend/internal/mapstate"
)

var (
	ErrAlreadyTaken = errors.New("already taken")
	ErrBagFull      = errors.New("bag full")
	ErrNotInReach   = errors.New("not in reach")
	ErrInvalidKind  = errors.New("invalid kind")
)

type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Cells interface {
	List(ctx context.Context, gameID string) ([]mapstate.Cell, error)
	ClaimFlora(ctx context.Context, tx pgx.Tx, gameID string, x, y int) (mapstate.FloraClaim, error)
	Hit(ctx context.Context, tx pgx.Tx, gameID string, x, y, maxDurability, damage int, clear mapstate.Clear) (mapstate.Hit, error)
}

type Bags interface {
	Grant(ctx context.Context, tx pgx.Tx, gameID, playerID, slug string, quantity int) error
	ListTx(ctx context.Context, tx pgx.Tx, gameID, playerID string) ([]inventory.Slot, error)
}

type Games interface {
	Find(ctx context.Context, id string) (game.Game, error)
}

type Presence interface {
	Position(gameID, playerID string) (x, y float64, ok bool)
}

type Result struct {
	Inventory inventory.Inventory
	Cell      mapstate.Cell
}

// StrikeResult is one blow. Inventory is set only when Broken is true.
type StrikeResult struct {
	Broken    bool
	Inventory inventory.Inventory
	Cell      mapstate.Cell
}

type Service struct {
	pool     txBeginner
	cells    Cells
	bags     Bags
	games    Games
	presence Presence
}

func NewService(pool txBeginner, cells Cells, bags Bags, games Games, presence Presence) *Service {
	return &Service{pool: pool, cells: cells, bags: bags, games: games, presence: presence}
}

func (s *Service) List(ctx context.Context, gameID string) ([]mapstate.Cell, error) {
	if _, err := s.games.Find(ctx, gameID); err != nil {
		return nil, err
	}
	cells, err := s.cells.List(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if cells == nil {
		cells = []mapstate.Cell{}
	}
	return cells, nil
}

// Gather claims the flora cell and grants one item in a single transaction.
// The cell is locked before the bag so two players cannot deadlock.
// A full bag rolls the claim back and leaves the flower on the map.
func (s *Service) Gather(ctx context.Context, gameID, playerID string, x, y int, kind string) (Result, error) {
	if !mapstate.IsFlower(kind) {
		return Result{}, ErrInvalidKind
	}
	if _, err := s.games.Find(ctx, gameID); err != nil {
		return Result{}, err
	}
	pixelX, pixelY, ok := s.presence.Position(gameID, playerID)
	if !ok || !inReach(pixelX, pixelY, x, y) {
		return Result{}, ErrNotInReach
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	claim, err := s.cells.ClaimFlora(ctx, tx, gameID, x, y)
	if err != nil {
		return Result{}, err
	}
	if claim.Taken {
		return Result{}, ErrAlreadyTaken
	}
	slug := kind
	if claim.Held != "" {
		slug = claim.Held
	} else if !claim.Inserted {
		return Result{}, fmt.Errorf("claim flora %s %d,%d", gameID, x, y)
	}

	if err := s.bags.Grant(ctx, tx, gameID, playerID, slug, 1); err != nil {
		if errors.Is(err, inventory.ErrBagFull) {
			return Result{}, ErrBagFull
		}
		return Result{}, err
	}

	slots, err := s.bags.ListTx(ctx, tx, gameID, playerID)
	if err != nil {
		return Result{}, err
	}
	if slots == nil {
		slots = []inventory.Slot{}
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	return Result{
		Inventory: inventory.Inventory{Capacity: inventory.Capacity, Slots: slots},
		Cell:      clearedFlora(x, y),
	}, nil
}

func clearedFlora(x, y int) mapstate.Cell {
	return mapstate.Cell{
		X:      x,
		Y:      y,
		Layers: map[string]*string{mapstate.LayerFlora: nil},
	}
}

// Strike spends one bare-hand blow on a destructible node.
// The cell lock inside Hit is taken before the bag lock inside Grant.
// A full bag rolls the blow back and leaves the recorded durability in place.
func (s *Service) Strike(ctx context.Context, gameID, playerID string, x, y int, kind string) (StrikeResult, error) {
	node, ok := breakable(kind)
	if !ok {
		return StrikeResult{}, ErrInvalidKind
	}
	if _, err := s.games.Find(ctx, gameID); err != nil {
		return StrikeResult{}, err
	}
	pixelX, pixelY, ok := s.presence.Position(gameID, playerID)
	if !ok || !inReach(pixelX, pixelY, x, y) {
		return StrikeResult{}, ErrNotInReach
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StrikeResult{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	hit, err := s.cells.Hit(ctx, tx, gameID, x, y, node.Max, handDamage, node.Clear)
	if err != nil {
		return StrikeResult{}, err
	}
	if hit.Gone {
		return StrikeResult{}, ErrAlreadyTaken
	}

	var slots []inventory.Slot
	if hit.Broken {
		for _, piece := range node.Loot {
			if err := s.bags.Grant(ctx, tx, gameID, playerID, piece.Slug, piece.Quantity); err != nil {
				if errors.Is(err, inventory.ErrBagFull) {
					return StrikeResult{}, ErrBagFull
				}
				return StrikeResult{}, err
			}
		}
		slots, err = s.bags.ListTx(ctx, tx, gameID, playerID)
		if err != nil {
			return StrikeResult{}, err
		}
		if slots == nil {
			slots = []inventory.Slot{}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return StrikeResult{}, err
	}

	cell := brokenNode(x, y, node.Clear)
	if !hit.Broken {
		cell = damagedNode(x, y, hit.Remaining)
	}
	result := StrikeResult{Broken: hit.Broken, Cell: cell}
	if hit.Broken {
		result.Inventory = inventory.Inventory{Capacity: inventory.Capacity, Slots: slots}
	}
	return result, nil
}

func damagedNode(x, y, remaining int) mapstate.Cell {
	text := strconv.Itoa(remaining)
	return mapstate.Cell{
		X:      x,
		Y:      y,
		Layers: map[string]*string{mapstate.LayerDurability: &text},
	}
}

func brokenNode(x, y int, clear mapstate.Clear) mapstate.Cell {
	return mapstate.Cell{
		X: x,
		Y: y,
		Layers: map[string]*string{
			clear.Layer:              clear.Value,
			mapstate.LayerDurability: nil,
		},
	}
}
