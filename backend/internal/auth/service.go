package auth

import (
	"context"
	"errors"
	"time"

	"game/backend/internal/player"

	"golang.org/x/crypto/bcrypt"
)

type Players interface {
	Create(ctx context.Context, login, passwordHash string) (player.Player, error)
	FindByLogin(ctx context.Context, login string) (player.Player, error)
	FindByID(ctx context.Context, id string) (player.Player, error)
}

type RefreshStore interface {
	Insert(ctx context.Context, playerID, tokenHash string, expiresAt time.Time) error
	FindByHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	DeleteByHash(ctx context.Context, tokenHash string) error
}

type Session struct {
	Player       player.Player
	AccessToken  string
	RefreshToken string
}

type ServiceConfig struct {
	Secret     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	BcryptCost int
	Now        func() time.Time
}

type Service struct {
	players    Players
	tokens     RefreshStore
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	bcryptCost int
	dummyHash  []byte
	now        func() time.Time
}

func NewService(players Players, tokens RefreshStore, cfg ServiceConfig) (*Service, error) {
	cost := cfg.BcryptCost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	dummy, err := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), cost)
	if err != nil {
		return nil, err
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		players:    players,
		tokens:     tokens,
		secret:     cfg.Secret,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		bcryptCost: cost,
		dummyHash:  dummy,
		now:        now,
	}, nil
}

func (s *Service) Register(ctx context.Context, login, password string) (Session, error) {
	if err := ValidateLogin(login); err != nil {
		return Session{}, err
	}
	if err := ValidatePassword(password); err != nil {
		return Session{}, err
	}
	hash, err := HashPassword(password, s.bcryptCost)
	if err != nil {
		return Session{}, err
	}
	created, err := s.players.Create(ctx, login, hash)
	if errors.Is(err, player.ErrInvalidLogin) {
		return Session{}, ErrInvalidLogin
	}
	if err != nil {
		return Session{}, err
	}
	return s.issue(ctx, created)
}

func (s *Service) Login(ctx context.Context, login, password string) (Session, error) {
	found, err := s.players.FindByLogin(ctx, login)
	if errors.Is(err, player.ErrNotFound) {
		_ = PasswordMatches(string(s.dummyHash), password)
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}
	if !PasswordMatches(found.PasswordHash, password) {
		return Session{}, ErrInvalidCredentials
	}
	return s.issue(ctx, found)
}

func (s *Service) Refresh(ctx context.Context, raw string) (Session, error) {
	if raw == "" {
		return Session{}, ErrUnauthorized
	}
	hash := HashRefreshToken(raw)
	row, err := s.tokens.FindByHash(ctx, hash)
	if errors.Is(err, ErrNotFound) {
		return Session{}, ErrUnauthorized
	}
	if err != nil {
		return Session{}, err
	}
	if !row.ExpiresAt.After(s.now()) {
		_ = s.tokens.DeleteByHash(ctx, hash)
		return Session{}, ErrUnauthorized
	}
	found, err := s.players.FindByID(ctx, row.PlayerID)
	if errors.Is(err, player.ErrNotFound) {
		_ = s.tokens.DeleteByHash(ctx, hash)
		return Session{}, ErrUnauthorized
	}
	if err != nil {
		return Session{}, err
	}
	if err := s.tokens.DeleteByHash(ctx, hash); err != nil {
		return Session{}, err
	}
	return s.issue(ctx, found)
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.tokens.DeleteByHash(ctx, HashRefreshToken(raw))
}

func (s *Service) Current(ctx context.Context, access string) (player.Player, error) {
	playerID, _, err := ParseAccess(s.secret, access)
	if err != nil {
		return player.Player{}, ErrUnauthorized
	}
	found, err := s.players.FindByID(ctx, playerID)
	if errors.Is(err, player.ErrNotFound) {
		return player.Player{}, ErrUnauthorized
	}
	return found, err
}

func (s *Service) issue(ctx context.Context, p player.Player) (Session, error) {
	now := s.now()
	access, err := SignAccess(s.secret, p.ID, p.Login, now, s.accessTTL)
	if err != nil {
		return Session{}, err
	}
	raw, hash, err := NewRefreshToken()
	if err != nil {
		return Session{}, err
	}
	if err := s.tokens.Insert(ctx, p.ID, hash, now.Add(s.refreshTTL)); err != nil {
		return Session{}, err
	}
	return Session{Player: p, AccessToken: access, RefreshToken: raw}, nil
}
