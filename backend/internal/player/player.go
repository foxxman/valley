package player

import "time"

type Player struct {
	ID           string
	Login        string
	PasswordHash string
	CreatedAt    time.Time
}
