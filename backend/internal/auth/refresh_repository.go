package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshToken struct {
	ID        string
	PlayerID  string
	TokenHash string
	ExpiresAt time.Time
}

type RefreshRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshRepository(pool *pgxpool.Pool) *RefreshRepository {
	return &RefreshRepository{pool: pool}
}

func (r *RefreshRepository) Insert(ctx context.Context, playerID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (player_id, token_hash, expires_at)
		VALUES ($1::uuid, $2, $3)
	`, playerID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *RefreshRepository) FindByHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	var found RefreshToken
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, player_id::text, token_hash, expires_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(&found.ID, &found.PlayerID, &found.TokenHash, &found.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return RefreshToken{}, ErrNotFound
	}
	if err != nil {
		return RefreshToken{}, fmt.Errorf("find refresh token: %w", err)
	}
	return found, nil
}

func (r *RefreshRepository) DeleteByHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}
