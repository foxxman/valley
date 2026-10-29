package craft

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"game/backend/internal/game"
	"game/backend/internal/inventory"
)

var (
	ErrUnknownRecipe      = errors.New("unknown recipe")
	ErrMissingIngredients = errors.New("missing ingredients")
	ErrBagFull            = errors.New("bag full")
)

type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Bags interface {
	Exchange(ctx context.Context, tx pgx.Tx, gameID, playerID string, costs []inventory.Cost, resultSlug string, resultQty int) error
	ListTx(ctx context.Context, tx pgx.Tx, gameID, playerID string) ([]inventory.Slot, error)
	FindBySlugs(ctx context.Context, slugs []string) ([]inventory.Item, error)
}

type Games interface {
	Find(ctx context.Context, id string) (game.Game, error)
}

// Part is one displayed ingredient or result.
type Part struct {
	Item     inventory.Item
	Quantity int
}

// Listed is a recipe joined with catalog names and icons.
type Listed struct {
	Slug        string
	Result      Part
	Ingredients []Part
}

type Service struct {
	pool  txBeginner
	bags  Bags
	games Games
}

func NewService(pool txBeginner, bags Bags, games Games) *Service {
	return &Service{pool: pool, bags: bags, games: games}
}

func (s *Service) List(ctx context.Context, gameID string) ([]Listed, error) {
	if _, err := s.games.Find(ctx, gameID); err != nil {
		return nil, err
	}
	items, err := s.bags.FindBySlugs(ctx, catalogSlugs())
	if err != nil {
		return nil, err
	}
	bySlug := make(map[string]inventory.Item, len(items))
	for _, item := range items {
		bySlug[item.Slug] = item
	}

	listed := make([]Listed, 0, len(Recipes()))
	for _, recipe := range Recipes() {
		result, ok := bySlug[recipe.ResultSlug]
		if !ok {
			return nil, fmt.Errorf("recipe item %s", recipe.ResultSlug)
		}
		parts := make([]Part, 0, len(recipe.Ingredients))
		for _, ingredient := range recipe.Ingredients {
			item, ok := bySlug[ingredient.Slug]
			if !ok {
				return nil, fmt.Errorf("recipe item %s", ingredient.Slug)
			}
			parts = append(parts, Part{Item: item, Quantity: ingredient.Quantity})
		}
		listed = append(listed, Listed{
			Slug:        recipe.Slug,
			Result:      Part{Item: result, Quantity: recipe.ResultQuantity},
			Ingredients: parts,
		})
	}
	return listed, nil
}

// Craft spends the recipe and grants the result.
// The bag is unchanged unless the whole exchange fits.
func (s *Service) Craft(ctx context.Context, gameID, playerID, recipeSlug string) (inventory.Inventory, error) {
	if _, err := s.games.Find(ctx, gameID); err != nil {
		return inventory.Inventory{}, err
	}
	recipe, ok := Find(recipeSlug)
	if !ok {
		return inventory.Inventory{}, ErrUnknownRecipe
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return inventory.Inventory{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	costs := make([]inventory.Cost, len(recipe.Ingredients))
	for i, ingredient := range recipe.Ingredients {
		costs[i] = inventory.Cost{Slug: ingredient.Slug, Quantity: ingredient.Quantity}
	}
	if err := s.bags.Exchange(ctx, tx, gameID, playerID, costs, recipe.ResultSlug, recipe.ResultQuantity); err != nil {
		if errors.Is(err, inventory.ErrBagFull) {
			return inventory.Inventory{}, ErrBagFull
		}
		if errors.Is(err, inventory.ErrShort) {
			return inventory.Inventory{}, ErrMissingIngredients
		}
		return inventory.Inventory{}, err
	}

	slots, err := s.bags.ListTx(ctx, tx, gameID, playerID)
	if err != nil {
		return inventory.Inventory{}, err
	}
	if slots == nil {
		slots = []inventory.Slot{}
	}
	if err := tx.Commit(ctx); err != nil {
		return inventory.Inventory{}, err
	}
	return inventory.Inventory{Capacity: inventory.Capacity, Slots: slots}, nil
}

func catalogSlugs() []string {
	seen := map[string]struct{}{}
	slugs := []string{}
	add := func(slug string) {
		if _, ok := seen[slug]; ok {
			return
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	for _, recipe := range Recipes() {
		add(recipe.ResultSlug)
		for _, ingredient := range recipe.Ingredients {
			add(ingredient.Slug)
		}
	}
	return slugs
}
