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
	g.id::text, g.name, g.owner_id::text, p.login, g.created_at, g.updated_at
`

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, ownerID, name string) (Game, error) {
	row := r.pool.QueryRow(ctx, `
		WITH created AS (
			INSERT INTO games (name, owner_id)
			VALUES ($1, $2::uuid)
			RETURNING id, name, owner_id, created_at, updated_at
		)
		SELECT `+gameColumns+`
		FROM created g
		JOIN players p ON p.id = g.owner_id
	`, name, ownerID)
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
			RETURNING id, name, owner_id, created_at, updated_at
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
	err := row.Scan(&found.ID, &found.Name, &found.OwnerID, &found.OwnerLogin, &found.CreatedAt, &found.UpdatedAt)
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
	byCode := map[string]error{
		"23514": ErrInvalidName,
		"22P02": ErrNotFound,
	}
	if mapped, ok := byCode[pgErr.Code]; ok {
		return mapped
	}
	return err
}
