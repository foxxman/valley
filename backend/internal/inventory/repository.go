package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, gameID, playerID string) ([]Slot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.slot_index, s.quantity,
			i.id::text, i.slug, i.name, i.icon_key, i.description, i.max_stack
		FROM inventory_slots s
		JOIN items i ON i.id = s.item_id
		WHERE s.game_id = $1::uuid AND s.player_id = $2::uuid
		ORDER BY s.slot_index
	`, gameID, playerID)
	if err != nil {
		return nil, fmt.Errorf("list inventory: %w", err)
	}
	defer rows.Close()

	found := []Slot{}
	for rows.Next() {
		var slot Slot
		if err := rows.Scan(
			&slot.Index,
			&slot.Quantity,
			&slot.Item.ID,
			&slot.Item.Slug,
			&slot.Item.Name,
			&slot.Item.IconKey,
			&slot.Item.Description,
			&slot.Item.MaxStack,
		); err != nil {
			return nil, fmt.Errorf("list inventory: %w", err)
		}
		found = append(found, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list inventory: %w", err)
	}
	return found, nil
}
