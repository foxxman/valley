package auth

import (
	"errors"

	"game/backend/internal/player"
)

var (
	ErrInvalidLogin       = errors.New("invalid login")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrLoginTaken         = player.ErrLoginTaken
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNotFound           = errors.New("not found")
)
