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
	if len(names) != 1 || names[0] != "001_players.sql" {
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
}
