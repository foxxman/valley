package mapstate

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
