package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("secret12", bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "secret12" {
		t.Fatal("hash must not be the raw password")
	}
	if !PasswordMatches(hash, "secret12") {
		t.Fatal("expected match")
	}
	if PasswordMatches(hash, "other-password") {
		t.Fatal("expected mismatch")
	}
}
