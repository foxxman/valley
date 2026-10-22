package game

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxNameLen = 40

var (
	ErrInvalidName = errors.New("invalid game name")
	ErrNotFound    = errors.New("game not found")
	ErrForbidden   = errors.New("forbidden")
)

var idPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type Game struct {
	ID         string
	Name       string
	OwnerID    string
	OwnerLogin string
	CreatedAt  time.Time
	UpdatedAt  time.Time
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
