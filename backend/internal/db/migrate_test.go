package db

import (
	"strings"
	"testing"
)

func TestMigrationFiles(t *testing.T) {
	names, err := migrationFiles()
	if err != nil {
		t.Fatalf("files: %v", err)
	}
	if len(names) != 2 || names[0] != "001_players.sql" || names[1] != "002_games.sql" {
		t.Fatalf("names %v", names)
	}

	body, err := migrationsFS.ReadFile("migrations/001_players.sql")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	sqlText := string(body)
	required := []string{
		"CREATE TABLE players",
		"CREATE UNIQUE INDEX players_login_lower_idx ON players (lower(login))",
		"REFERENCES players (id) ON DELETE CASCADE",
		"token_hash TEXT NOT NULL UNIQUE",
	}
	for _, fragment := range required {
		if !strings.Contains(sqlText, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}

	games, err := migrationsFS.ReadFile("migrations/002_games.sql")
	if err != nil {
		t.Fatalf("read games: %v", err)
	}
	gamesSQL := string(games)
	for _, fragment := range []string{
		"CREATE TABLE games",
		"owner_id UUID NOT NULL REFERENCES players (id) ON DELETE CASCADE",
		"CONSTRAINT games_name_length CHECK (char_length(name) BETWEEN 1 AND 40)",
		"CREATE INDEX games_owner_id_idx ON games (owner_id)",
	} {
		if !strings.Contains(gamesSQL, fragment) {
			t.Fatalf("games migration missing %q", fragment)
		}
	}
}
