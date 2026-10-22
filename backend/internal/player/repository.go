package player

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("player not found")
	ErrLoginTaken = errors.New("login taken")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, login, passwordHash string) (Player, error) {
	var created Player
	err := r.pool.QueryRow(ctx, `
		INSERT INTO players (login, password_hash)
		VALUES ($1, $2)
		RETURNING id::text, login, password_hash, created_at
	`, login, passwordHash).Scan(&created.ID, &created.Login, &created.PasswordHash, &created.CreatedAt)
	if err != nil {
		return Player{}, mapWriteError(err)
	}
	return created, nil
}

func (r *Repository) FindByLogin(ctx context.Context, login string) (Player, error) {
	return r.find(ctx, `
		SELECT id::text, login, password_hash, created_at
		FROM players
		WHERE lower(login) = lower($1)
	`, login)
}

func (r *Repository) FindByID(ctx context.Context, id string) (Player, error) {
	return r.find(ctx, `
		SELECT id::text, login, password_hash, created_at
		FROM players
		WHERE id = $1::uuid
	`, id)
}

func (r *Repository) find(ctx context.Context, query, arg string) (Player, error) {
	var found Player
	err := r.pool.QueryRow(ctx, query, arg).Scan(&found.ID, &found.Login, &found.PasswordHash, &found.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Player{}, ErrNotFound
	}
	if err != nil {
		return Player{}, fmt.Errorf("find player: %w", err)
	}
	return found, nil
}

func mapWriteError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("create player: %w", err)
	}
	byCode := map[string]error{
		"23505": ErrLoginTaken,
		"23514": ErrInvalidLogin,
	}
	if mapped, ok := byCode[pgErr.Code]; ok {
		return mapped
	}
	return fmt.Errorf("create player: %w", err)
}

// ErrInvalidLogin is the check-constraint failure for a login that reached SQL.
var ErrInvalidLogin = errors.New("invalid login")
