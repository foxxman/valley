package game

import (
	"strings"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "trim", input: "  Долина  ", want: "Долина"},
		{name: "max runes", input: strings.Repeat("я", MaxNameLen), want: strings.Repeat("я", MaxNameLen)},
		{name: "empty", input: "   ", wantErr: true},
		{name: "too long", input: strings.Repeat("я", MaxNameLen+1), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeName(test.input)
			if test.wantErr {
				if err != ErrInvalidName {
					t.Fatalf("err %v", err)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("got %q err %v", got, err)
			}
		})
	}
}

func TestValidID(t *testing.T) {
	if !ValidID("11111111-1111-1111-1111-111111111111") {
		t.Fatal("uuid should be valid")
	}
	if ValidID("K7M2QA") || ValidID("11111111-1111-1111-1111-111111111111-extra") {
		t.Fatal("partial id should be rejected")
	}
}

func TestValidWorldSeed(t *testing.T) {
	if !ValidWorldSeed(0) || !ValidWorldSeed(MaxWorldSeed) {
		t.Fatal("edges of the uint32 range should be valid")
	}
	if ValidWorldSeed(-1) || ValidWorldSeed(MaxWorldSeed+1) {
		t.Fatal("seed outside uint32 range should be rejected")
	}
}

func TestRandomWorldSeed(t *testing.T) {
	for range 8 {
		seed, err := RandomWorldSeed()
		if err != nil || !ValidWorldSeed(seed) {
			t.Fatalf("seed %d err %v", seed, err)
		}
	}
}
