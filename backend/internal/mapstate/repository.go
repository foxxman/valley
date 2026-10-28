package mapstate

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Hit is one durability change for a destructible cell.
// Gone means the node was already removed.
// Broken means this hit wrote the clear layer and emptied durability.
type Hit struct {
	Remaining int
	Broken    bool
	Gone      bool
}

// Clear is the layer written when a node breaks.
// A nil Value clears the layer. A stored value equal to Value means the node is already gone.
type Clear struct {
	Layer string
	Value *string
}

// FloraClaim is the result of trying to take the flora layer inside a transaction.
// Inserted means this transaction created the clear row and should grant the requested kind.
// Held is a flower slug that was already stored and is now cleared; grant Held, not the request.
// Taken means the cell is not a flower anymore.
type FloraClaim struct {
	Inserted bool
	Held     string
	Taken    bool
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, gameID string) ([]Cell, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT x, y, layer, value
		FROM map_cells
		WHERE game_id = $1::uuid
		ORDER BY x, y, layer
	`, gameID)
	if err != nil {
		return nil, fmt.Errorf("list map cells: %w", err)
	}
	defer rows.Close()

	found := []cellRow{}
	for rows.Next() {
		var row cellRow
		if err := rows.Scan(&row.X, &row.Y, &row.Layer, &row.Value); err != nil {
			return nil, fmt.Errorf("list map cells: %w", err)
		}
		found = append(found, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list map cells: %w", err)
	}
	return groupRows(found), nil
}

// ClaimFlora runs inside the caller's transaction.
// The INSERT is the mutex: ON CONFLICT DO NOTHING waits on the in-progress
// unique index row, then either inserts (the other transaction rolled back)
// or conflicts (the other transaction committed).
func (r *Repository) ClaimFlora(ctx context.Context, tx pgx.Tx, gameID string, x, y int) (FloraClaim, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO map_cells (game_id, x, y, layer, value)
		VALUES ($1::uuid, $2, $3, $4, NULL)
		ON CONFLICT (game_id, x, y, layer) DO NOTHING
		RETURNING game_id::text
	`, gameID, x, y, LayerFlora).Scan(&id)
	if err == nil {
		return FloraClaim{Inserted: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return FloraClaim{}, fmt.Errorf("claim flora: %w", err)
	}

	var value *string
	err = tx.QueryRow(ctx, `
		SELECT value
		FROM map_cells
		WHERE game_id = $1::uuid AND x = $2 AND y = $3 AND layer = $4
		FOR UPDATE
	`, gameID, x, y, LayerFlora).Scan(&value)
	if err != nil {
		return FloraClaim{}, fmt.Errorf("lock flora: %w", err)
	}
	if value != nil && IsFlower(*value) {
		slug := *value
		if _, err := tx.Exec(ctx, `
			UPDATE map_cells
			SET value = NULL
			WHERE game_id = $1::uuid AND x = $2 AND y = $3 AND layer = $4
		`, gameID, x, y, LayerFlora); err != nil {
			return FloraClaim{}, fmt.Errorf("clear flora: %w", err)
		}
		return FloraClaim{Held: slug}, nil
	}
	return FloraClaim{Taken: true}, nil
}

// Hit applies damage inside the caller's transaction.
// An advisory lock covers the first blow, when neither row exists yet and
// FOR UPDATE would lock nothing. Callers lock the bag only after this returns.
func (r *Repository) Hit(ctx context.Context, tx pgx.Tx, gameID string, x, y, maxDurability, damage int, clear Clear) (Hit, error) {
	if maxDurability < 1 || damage < 1 || clear.Layer == "" {
		return Hit{}, fmt.Errorf("hit durability %d damage %d layer %q", maxDurability, damage, clear.Layer)
	}
	lockKey := fmt.Sprintf("cell:%d,%d", x, y)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, gameID, lockKey); err != nil {
		return Hit{}, fmt.Errorf("lock cell: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT layer, value
		FROM map_cells
		WHERE game_id = $1::uuid AND x = $2 AND y = $3
			AND layer IN ($4, $5)
		FOR UPDATE
	`, gameID, x, y, clear.Layer, LayerDurability)
	if err != nil {
		return Hit{}, fmt.Errorf("lock node: %w", err)
	}
	defer rows.Close()

	var clearValue *string
	var clearSeen bool
	var durability *string
	var durabilitySeen bool
	for rows.Next() {
		var layer string
		var value *string
		if err := rows.Scan(&layer, &value); err != nil {
			return Hit{}, fmt.Errorf("lock node: %w", err)
		}
		if layer == clear.Layer {
			clearValue = value
			clearSeen = true
		}
		if layer == LayerDurability {
			durability = value
			durabilitySeen = true
		}
	}
	if err := rows.Err(); err != nil {
		return Hit{}, fmt.Errorf("lock node: %w", err)
	}
	if clearSeen && sameValue(clearValue, clear.Value) {
		return Hit{Gone: true}, nil
	}
	if durabilitySeen && durability == nil {
		return Hit{Gone: true}, nil
	}

	current := maxDurability
	if durabilitySeen && durability != nil {
		parsed, err := strconv.Atoi(*durability)
		if err != nil || parsed < 1 {
			return Hit{}, fmt.Errorf("durability %q at %d,%d", *durability, x, y)
		}
		current = parsed
	}
	next := current - damage
	if next > 0 {
		text := strconv.Itoa(next)
		if err := upsertCell(ctx, tx, gameID, x, y, LayerDurability, &text); err != nil {
			return Hit{}, err
		}
		return Hit{Remaining: next}, nil
	}
	if err := upsertCell(ctx, tx, gameID, x, y, clear.Layer, clear.Value); err != nil {
		return Hit{}, err
	}
	if err := upsertCell(ctx, tx, gameID, x, y, LayerDurability, nil); err != nil {
		return Hit{}, err
	}
	return Hit{Broken: true}, nil
}

func sameValue(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func upsertCell(ctx context.Context, tx pgx.Tx, gameID string, x, y int, layer string, value *string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO map_cells (game_id, x, y, layer, value)
		VALUES ($1::uuid, $2, $3, $4, $5)
		ON CONFLICT (game_id, x, y, layer) DO UPDATE SET value = EXCLUDED.value
	`, gameID, x, y, layer, value); err != nil {
		return fmt.Errorf("upsert %s: %w", layer, err)
	}
	return nil
}
