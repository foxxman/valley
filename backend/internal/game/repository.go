package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const gameColumns = `
	g.id::text, g.name, g.owner_id::text, p.login, g.world_seed, g.created_at, g.updated_at
`

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, ownerID, name string, worldSeed int64) (Game, error) {
	row := r.pool.QueryRow(ctx, `
		WITH created AS (
			INSERT INTO games (name, owner_id, world_seed)
			VALUES ($1, $2::uuid, $3)
			RETURNING id, name, owner_id, world_seed, created_at, updated_at
		)
		SELECT `+gameColumns+`
		FROM created g
		JOIN players p ON p.id = g.owner_id
	`, name, ownerID, worldSeed)
	created, err := scanGame(row)
	if err != nil {
		return Game{}, fmt.Errorf("create game: %w", mapGameError(err))
	}
	return created, nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (Game, error) {
	if !ValidID(id) {
		return Game{}, ErrNotFound
	}
	row := r.pool.QueryRow(ctx, `
		SELECT `+gameColumns+`
		FROM games g
		JOIN players p ON p.id = g.owner_id
		WHERE g.id = $1::uuid
	`, id)
	found, err := scanGame(row)
	if err != nil {
		return Game{}, mapGameError(err)
	}
	return found, nil
}

func (r *Repository) ListByOwner(ctx context.Context, ownerID string) ([]Game, error) {
	if !ValidID(ownerID) {
		return []Game{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+gameColumns+`
		FROM games g
		JOIN players p ON p.id = g.owner_id
		WHERE g.owner_id = $1::uuid
		ORDER BY g.created_at DESC, g.id DESC
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	defer rows.Close()

	found := []Game{}
	for rows.Next() {
		item, err := scanGame(rows)
		if err != nil {
			return nil, fmt.Errorf("list games: %w", err)
		}
		found = append(found, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list games: %w", err)
	}
	return found, nil
}

func (r *Repository) UpdateName(ctx context.Context, id, name string) (Game, error) {
	if !ValidID(id) {
		return Game{}, ErrNotFound
	}
	row := r.pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE games
			SET name = $2, updated_at = now()
			WHERE id = $1::uuid
			RETURNING id, name, owner_id, world_seed, created_at, updated_at
		)
		SELECT `+gameColumns+`
		FROM updated g
		JOIN players p ON p.id = g.owner_id
	`, id, name)
	updated, err := scanGame(row)
	if err != nil {
		return Game{}, fmt.Errorf("rename game: %w", mapGameError(err))
	}
	return updated, nil
}

func scanGame(row pgx.Row) (Game, error) {
	var found Game
	err := row.Scan(
		&found.ID,
		&found.Name,
		&found.OwnerID,
		&found.OwnerLogin,
		&found.WorldSeed,
		&found.CreatedAt,
		&found.UpdatedAt,
	)
	return found, err
}

func mapGameError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code == "23514" {
		if pgErr.ConstraintName == "games_world_seed_range" {
			return ErrInvalidSeed
		}
		return ErrInvalidName
	}
	if pgErr.Code == "22P02" {
		return ErrNotFound
	}
	return err
}
