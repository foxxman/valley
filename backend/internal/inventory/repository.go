package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, gameID, playerID string) ([]Slot, error) {
	return listSlots(ctx, r.pool, gameID, playerID, false)
}

func (r *Repository) ListTx(ctx context.Context, tx pgx.Tx, gameID, playerID string) ([]Slot, error) {
	return listSlots(ctx, tx, gameID, playerID, false)
}

type stackPlacement struct {
	index  int
	add    int
	insert bool
}

// Grant adds quantity of one item inside the caller's transaction.
// Either every unit fits, or the bag is unchanged and the error is ErrBagFull.
// The advisory lock is taken before the slot read: FOR UPDATE locks nothing
// when the bag is empty, so two grants by the same player could both insert slot 0.
func (r *Repository) Grant(ctx context.Context, tx pgx.Tx, gameID, playerID, slug string, quantity int) error {
	if quantity < 1 {
		return fmt.Errorf("grant quantity %d", quantity)
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, gameID, playerID); err != nil {
		return fmt.Errorf("lock bag: %w", err)
	}

	var itemID string
	var maxStack int
	err := tx.QueryRow(ctx, `
		SELECT id::text, max_stack
		FROM items
		WHERE slug = $1
	`, slug).Scan(&itemID, &maxStack)
	if err != nil {
		return fmt.Errorf("find item %s: %w", slug, err)
	}

	slots, err := listSlots(ctx, tx, gameID, playerID, true)
	if err != nil {
		return err
	}

	placements, err := planGrant(slots, itemID, maxStack, quantity)
	if err != nil {
		return err
	}
	for _, place := range placements {
		if place.insert {
			if _, err := tx.Exec(ctx, `
				INSERT INTO inventory_slots (game_id, player_id, slot_index, item_id, quantity)
				VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5)
			`, gameID, playerID, place.index, itemID, place.add); err != nil {
				return fmt.Errorf("insert slot: %w", err)
			}
			continue
		}
		tag, err := tx.Exec(ctx, `
			UPDATE inventory_slots
			SET quantity = quantity + $4
			WHERE game_id = $1::uuid AND player_id = $2::uuid AND slot_index = $3
				AND quantity + $4 <= $5
		`, gameID, playerID, place.index, place.add, maxStack)
		if err != nil {
			return fmt.Errorf("stack item: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return ErrBagFull
		}
	}
	return nil
}

// planGrant fills existing stacks of itemID, then empty slots.
// A shortfall returns ErrBagFull before any write.
func planGrant(slots []Slot, itemID string, maxStack, quantity int) ([]stackPlacement, error) {
	used := make(map[int]struct{}, len(slots))
	placements := make([]stackPlacement, 0, 2)
	remaining := quantity
	for _, slot := range slots {
		used[slot.Index] = struct{}{}
		if remaining == 0 || slot.Item.ID != itemID || slot.Quantity >= maxStack {
			continue
		}
		add := maxStack - slot.Quantity
		if add > remaining {
			add = remaining
		}
		placements = append(placements, stackPlacement{index: slot.Index, add: add})
		remaining -= add
	}
	for index := 0; index < Capacity && remaining > 0; index++ {
		if _, taken := used[index]; taken {
			continue
		}
		add := maxStack
		if add > remaining {
			add = remaining
		}
		placements = append(placements, stackPlacement{index: index, add: add, insert: true})
		used[index] = struct{}{}
		remaining -= add
	}
	if remaining > 0 {
		return nil, ErrBagFull
	}
	return placements, nil
}

type slotQuery interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func listSlots(ctx context.Context, q slotQuery, gameID, playerID string, lock bool) ([]Slot, error) {
	query := `
		SELECT s.slot_index, s.quantity,
			i.id::text, i.slug, i.name, i.icon_key, i.description, i.max_stack
		FROM inventory_slots s
		JOIN items i ON i.id = s.item_id
		WHERE s.game_id = $1::uuid AND s.player_id = $2::uuid
		ORDER BY s.slot_index
	`
	if lock {
		query += " FOR UPDATE OF s"
	}
	rows, err := q.Query(ctx, query, gameID, playerID)
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
