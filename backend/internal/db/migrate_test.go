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
	if len(names) != 4 || names[0] != "001_players.sql" || names[1] != "002_games.sql" || names[2] != "003_game_seed.sql" || names[3] != "004_inventory.sql" {
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

	seed, err := migrationsFS.ReadFile("migrations/003_game_seed.sql")
	if err != nil {
		t.Fatalf("read seed: %v", err)
	}
	seedSQL := string(seed)
	for _, fragment := range []string{
		"ADD COLUMN world_seed BIGINT NOT NULL DEFAULT 42",
		"CHECK (world_seed BETWEEN 0 AND 4294967295)",
		"ALTER COLUMN world_seed DROP DEFAULT",
	} {
		if !strings.Contains(seedSQL, fragment) {
			t.Fatalf("seed migration missing %q", fragment)
		}
	}

	inventory, err := migrationsFS.ReadFile("migrations/004_inventory.sql")
	if err != nil {
		t.Fatalf("read inventory: %v", err)
	}
	inventorySQL := string(inventory)
	for _, fragment := range []string{
		"CREATE TABLE items",
		"slug TEXT NOT NULL UNIQUE",
		"CONSTRAINT items_max_stack_range CHECK (max_stack BETWEEN 1 AND 999)",
		"CREATE TABLE inventory_slots",
		"game_id UUID NOT NULL REFERENCES games (id) ON DELETE CASCADE",
		"player_id UUID NOT NULL REFERENCES players (id) ON DELETE CASCADE",
		"item_id UUID NOT NULL REFERENCES items (id)",
		"PRIMARY KEY (game_id, player_id, slot_index)",
		"CONSTRAINT inventory_slots_index_range CHECK (slot_index BETWEEN 0 AND 19)",
		"CONSTRAINT inventory_slots_quantity_positive CHECK (quantity >= 1)",
	} {
		if !strings.Contains(inventorySQL, fragment) {
			t.Fatalf("inventory migration missing %q", fragment)
		}
	}
}
