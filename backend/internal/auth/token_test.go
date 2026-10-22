package auth

import (
	"strings"
	"testing"
	"time"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	secret := []byte(strings.Repeat("k", 32))
	now := time.Now().UTC()
	raw, err := SignAccess(secret, "player-1", "fox", now, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	id, login, err := ParseAccess(secret, raw)
	if err != nil {
		t.Fatal(err)
	}
	if id != "player-1" || login != "fox" {
		t.Fatalf("claims %s %s", id, login)
	}
}

func TestAccessTokenRejectsBadTokens(t *testing.T) {
	secret := []byte(strings.Repeat("k", 32))
	other := []byte(strings.Repeat("z", 32))
	now := time.Now().UTC()
	raw, err := SignAccess(secret, "player-1", "fox", now, time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	cases := []string{
		raw[:len(raw)-2] + "xx",
		"",
	}
	for _, token := range cases {
		if _, _, err := ParseAccess(secret, token); err != ErrUnauthorized {
			t.Fatalf("token %q: %v", token, err)
		}
	}
	if _, _, err := ParseAccess(other, raw); err != ErrUnauthorized {
		t.Fatalf("other secret: %v", err)
	}

	expired, err := SignAccess(secret, "player-1", "fox", now.Add(-time.Hour), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ParseAccess(secret, expired); err != ErrUnauthorized {
		t.Fatalf("expired: %v", err)
	}
}

func TestRefreshTokenHash(t *testing.T) {
	raw, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || hash == raw {
		t.Fatalf("raw and hash must differ, raw len %d", len(raw))
	}
	if HashRefreshToken(raw) != hash {
		t.Fatal("hash is not stable")
	}
	other, otherHash, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if other == raw || otherHash == hash {
		t.Fatal("expected a unique token")
	}
}
