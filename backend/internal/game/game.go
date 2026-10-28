package game

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxNameLen = 40
	// MaxWorldSeed is the largest value the world generator keeps.
	// The client turns the seed into a uint32, so the useful range is 0..2^32-1.
	MaxWorldSeed int64 = 4294967295
)

var (
	ErrInvalidName = errors.New("invalid game name")
	ErrInvalidSeed = errors.New("invalid world seed")
	ErrNotFound    = errors.New("game not found")
	ErrForbidden   = errors.New("forbidden")
)

var idPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type Game struct {
	ID         string
	Name       string
	OwnerID    string
	OwnerLogin string
	WorldSeed  int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func ValidWorldSeed(seed int64) bool {
	return seed >= 0 && seed <= MaxWorldSeed
}

func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > MaxNameLen {
		return "", ErrInvalidName
	}
	return name, nil
}
