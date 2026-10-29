package inventory

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

// Cost is one ingredient taken from the bag while crafting.
type Cost struct {
	Slug     string
	Quantity int
}

// Exchange removes ingredients and places the result in one transaction.
// Slots freed by the cost are available for the result. Nothing is written
// when the ingredients are short or the result does not fit.
func (r *Repository) Exchange(ctx context.Context, tx pgx.Tx, gameID, playerID string, costs []Cost, resultSlug string, resultQty int) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, gameID, playerID); err != nil {
		return fmt.Errorf("lock bag: %w", err)
	}

	var result Item
	err := tx.QueryRow(ctx, `
		SELECT id::text, slug, name, icon_key, description, max_stack
		FROM items
		WHERE slug = $1
	`, resultSlug).Scan(
		&result.ID,
		&result.Slug,
		&result.Name,
		&result.IconKey,
		&result.Description,
		&result.MaxStack,
	)
	if err != nil {
		return fmt.Errorf("find item %s: %w", resultSlug, err)
	}

	slots, err := listSlots(ctx, tx, gameID, playerID, true)
	if err != nil {
		return err
	}
	next, err := planExchange(slots, costs, result, resultQty)
	if err != nil {
		return err
	}
	return applyExchange(ctx, tx, gameID, playerID, slots, next)
}

func (r *Repository) FindBySlugs(ctx context.Context, slugs []string) ([]Item, error) {
	if len(slugs) == 0 {
		return []Item{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, slug, name, icon_key, description, max_stack
		FROM items
		WHERE slug = ANY($1::text[])
	`, slugs)
	if err != nil {
		return nil, fmt.Errorf("find items: %w", err)
	}
	defer rows.Close()

	found := []Item{}
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Slug, &item.Name, &item.IconKey, &item.Description, &item.MaxStack); err != nil {
			return nil, fmt.Errorf("find items: %w", err)
		}
		found = append(found, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find items: %w", err)
	}
	return found, nil
}

// planExchange drops ingredient counts from the lowest slot indexes, then
// places the result the same way as Grant: fill an existing stack, then use
// an empty slot. A slot whose quantity falls to zero is empty. The input
// slice is not modified.
func planExchange(slots []Slot, costs []Cost, result Item, resultQty int) ([]Slot, error) {
	if resultQty < 1 || result.MaxStack < 1 || result.ID == "" {
		return nil, fmt.Errorf("exchange result %s", result.Slug)
	}
	next := make([]Slot, len(slots))
	copy(next, slots)
	sort.Slice(next, func(i, j int) bool { return next[i].Index < next[j].Index })

	for _, cost := range costs {
		if cost.Quantity < 1 || cost.Slug == "" {
			return nil, fmt.Errorf("exchange cost %s", cost.Slug)
		}
		remaining := cost.Quantity
		for i := range next {
			if remaining == 0 {
				break
			}
			if next[i].Item.Slug != cost.Slug || next[i].Quantity == 0 {
				continue
			}
			take := next[i].Quantity
			if take > remaining {
				take = remaining
			}
			next[i].Quantity -= take
			remaining -= take
		}
		if remaining > 0 {
			return nil, ErrShort
		}
	}

	kept := make([]Slot, 0, len(next)+1)
	for _, slot := range next {
		if slot.Quantity > 0 {
			kept = append(kept, slot)
		}
	}

	remaining := resultQty
	for i := range kept {
		if remaining == 0 {
			break
		}
		if kept[i].Item.ID != result.ID || kept[i].Quantity >= result.MaxStack {
			continue
		}
		add := result.MaxStack - kept[i].Quantity
		if add > remaining {
			add = remaining
		}
		kept[i].Quantity += add
		remaining -= add
	}

	used := make(map[int]struct{}, len(kept))
	for _, slot := range kept {
		used[slot.Index] = struct{}{}
	}
	for index := 0; index < Capacity && remaining > 0; index++ {
		if _, taken := used[index]; taken {
			continue
		}
		add := result.MaxStack
		if add > remaining {
			add = remaining
		}
		kept = append(kept, Slot{Index: index, Quantity: add, Item: result})
		used[index] = struct{}{}
		remaining -= add
	}
	if remaining > 0 {
		return nil, ErrBagFull
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].Index < kept[j].Index })
	return kept, nil
}

func applyExchange(ctx context.Context, tx pgx.Tx, gameID, playerID string, before, after []Slot) error {
	prev := make(map[int]Slot, len(before))
	for _, slot := range before {
		prev[slot.Index] = slot
	}
	next := make(map[int]Slot, len(after))
	for _, slot := range after {
		next[slot.Index] = slot
	}

	for index, slot := range prev {
		updated, ok := next[index]
		if !ok {
			tag, err := tx.Exec(ctx, `
				DELETE FROM inventory_slots
				WHERE game_id = $1::uuid AND player_id = $2::uuid AND slot_index = $3
			`, gameID, playerID, index)
			if err != nil {
				return fmt.Errorf("clear slot: %w", err)
			}
			if tag.RowsAffected() != 1 {
				return fmt.Errorf("clear slot %d", index)
			}
			continue
		}
		if updated.Item.ID == slot.Item.ID && updated.Quantity == slot.Quantity {
			continue
		}
		tag, err := tx.Exec(ctx, `
			UPDATE inventory_slots
			SET item_id = $4::uuid, quantity = $5
			WHERE game_id = $1::uuid AND player_id = $2::uuid AND slot_index = $3
		`, gameID, playerID, index, updated.Item.ID, updated.Quantity)
		if err != nil {
			return fmt.Errorf("update slot: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("update slot %d", index)
		}
	}

	for index, slot := range next {
		if _, ok := prev[index]; ok {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO inventory_slots (game_id, player_id, slot_index, item_id, quantity)
			VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5)
		`, gameID, playerID, index, slot.Item.ID, slot.Quantity); err != nil {
			return fmt.Errorf("insert slot: %w", err)
		}
	}
	return nil
}
