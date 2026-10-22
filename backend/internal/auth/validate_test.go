package auth

import "testing"

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		login string
		ok    bool
	}{
		{login: "fox", ok: true},
		{login: "Fox_12", ok: true},
		{login: "ab", ok: false},
		{login: "has space", ok: false},
		{login: "кириллица", ok: false},
		{login: "aaaaaaaaaaaaaaaaaaaaaaaaa", ok: false},
		{login: "", ok: false},
	}

	for _, test := range tests {
		err := ValidateLogin(test.login)
		if test.ok && err != nil {
			t.Fatalf("login %q: %v", test.login, err)
		}
		if !test.ok && err != ErrInvalidLogin {
			t.Fatalf("login %q error %v", test.login, err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("12345678"); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePassword("short"); err != ErrInvalidPassword {
		t.Fatalf("got %v", err)
	}
}
