package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"game/backend/internal/player"

	"golang.org/x/crypto/bcrypt"
)

type memPlayers struct {
	byID    map[string]player.Player
	byLogin map[string]string
	nextID  int
}

func newMemPlayers() *memPlayers {
	return &memPlayers{
		byID:    map[string]player.Player{},
		byLogin: map[string]string{},
	}
}

func (m *memPlayers) Create(_ context.Context, login, passwordHash string) (player.Player, error) {
	key := strings.ToLower(login)
	if _, ok := m.byLogin[key]; ok {
		return player.Player{}, player.ErrLoginTaken
	}
	m.nextID++
	created := player.Player{
		ID:           strings.Repeat("a", m.nextID),
		Login:        login,
		PasswordHash: passwordHash,
		CreatedAt:    time.Unix(1, 0).UTC(),
	}
	m.byID[created.ID] = created
	m.byLogin[key] = created.ID
	return created, nil
}

func (m *memPlayers) FindByLogin(_ context.Context, login string) (player.Player, error) {
	id, ok := m.byLogin[strings.ToLower(login)]
	if !ok {
		return player.Player{}, player.ErrNotFound
	}
	return m.byID[id], nil
}

func (m *memPlayers) FindByID(_ context.Context, id string) (player.Player, error) {
	found, ok := m.byID[id]
	if !ok {
		return player.Player{}, player.ErrNotFound
	}
	return found, nil
}

type memTokens struct {
	rows map[string]RefreshToken
}

func newMemTokens() *memTokens {
	return &memTokens{rows: map[string]RefreshToken{}}
}

func (m *memTokens) Insert(_ context.Context, playerID, tokenHash string, expiresAt time.Time) error {
	m.rows[tokenHash] = RefreshToken{PlayerID: playerID, TokenHash: tokenHash, ExpiresAt: expiresAt}
	return nil
}

func (m *memTokens) FindByHash(_ context.Context, tokenHash string) (RefreshToken, error) {
	row, ok := m.rows[tokenHash]
	if !ok {
		return RefreshToken{}, ErrNotFound
	}
	return row, nil
}

func (m *memTokens) DeleteByHash(_ context.Context, tokenHash string) error {
	delete(m.rows, tokenHash)
	return nil
}

func testService(t *testing.T, players *memPlayers, tokens *memTokens, now time.Time) *Service {
	t.Helper()
	svc, err := NewService(players, tokens, ServiceConfig{
		Secret:     []byte(strings.Repeat("k", 32)),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: time.Hour,
		BcryptCost: bcrypt.MinCost,
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestRegisterAndLogin(t *testing.T) {
	now := time.Now().UTC()
	players := newMemPlayers()
	tokens := newMemTokens()
	svc := testService(t, players, tokens, now)

	session, err := svc.Register(context.Background(), "Fox", "secret12")
	if err != nil {
		t.Fatal(err)
	}
	if session.Player.Login != "Fox" || session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatalf("session %+v", session)
	}
	if session.Player.PasswordHash == "secret12" || !PasswordMatches(session.Player.PasswordHash, "secret12") {
		t.Fatal("password was not hashed")
	}
	if _, ok := tokens.rows[session.RefreshToken]; ok {
		t.Fatal("refresh store must keep the hash, not the raw token")
	}
	if _, ok := tokens.rows[HashRefreshToken(session.RefreshToken)]; !ok {
		t.Fatal("refresh hash was not stored")
	}

	if _, err := svc.Register(context.Background(), "fox", "secret12"); !errors.Is(err, ErrLoginTaken) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := svc.Register(context.Background(), "no", "secret12"); !errors.Is(err, ErrInvalidLogin) {
		t.Fatalf("short login: %v", err)
	}
	if _, err := svc.Register(context.Background(), "ok_name", "short"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("short password: %v", err)
	}

	logged, err := svc.Login(context.Background(), "FOX", "secret12")
	if err != nil {
		t.Fatal(err)
	}
	if logged.Player.ID != session.Player.ID {
		t.Fatalf("login id %s", logged.Player.ID)
	}
	if _, err := svc.Login(context.Background(), "FOX", "wrongpass"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("bad password: %v", err)
	}
	if _, err := svc.Login(context.Background(), "missing", "secret12"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("missing user: %v", err)
	}
}

func TestRefreshLogoutAndCurrent(t *testing.T) {
	now := time.Now().UTC()
	players := newMemPlayers()
	tokens := newMemTokens()
	svc := testService(t, players, tokens, now)
	ctx := context.Background()

	session, err := svc.Register(ctx, "fox", "secret12")
	if err != nil {
		t.Fatal(err)
	}

	current, err := svc.Current(ctx, session.AccessToken)
	if err != nil || current.Login != "fox" {
		t.Fatalf("current %+v %v", current, err)
	}
	if _, err := svc.Current(ctx, "not-a-token"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("bad access: %v", err)
	}

	refreshed, err := svc.Refresh(ctx, session.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tokens.rows[HashRefreshToken(session.RefreshToken)]; ok {
		t.Fatal("old refresh token must be removed")
	}
	if refreshed.RefreshToken == session.RefreshToken {
		t.Fatal("expected a rotated refresh token")
	}

	if _, err := svc.Refresh(ctx, session.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("reused refresh: %v", err)
	}

	if err := svc.Logout(ctx, refreshed.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if len(tokens.rows) != 0 {
		t.Fatalf("tokens left: %d", len(tokens.rows))
	}
	if err := svc.Logout(ctx, ""); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshRejectsExpiredAndDeletedPlayer(t *testing.T) {
	now := time.Now().UTC()
	players := newMemPlayers()
	tokens := newMemTokens()
	svc := testService(t, players, tokens, now)
	ctx := context.Background()

	session, err := svc.Register(ctx, "fox", "secret12")
	if err != nil {
		t.Fatal(err)
	}
	hash := HashRefreshToken(session.RefreshToken)
	row := tokens.rows[hash]
	row.ExpiresAt = now.Add(-time.Minute)
	tokens.rows[hash] = row
	if _, err := svc.Refresh(ctx, session.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expired: %v", err)
	}

	again, err := svc.Register(ctx, "wolf", "secret12")
	if err != nil {
		t.Fatal(err)
	}
	delete(players.byID, again.Player.ID)
	if _, err := svc.Refresh(ctx, again.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("deleted player: %v", err)
	}
	if _, err := svc.Current(ctx, again.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("current deleted: %v", err)
	}
}
