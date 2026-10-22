package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var migrationName = regexp.MustCompile(`^[0-9]{3}_[a-z0-9_]+\.sql$`)

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	names, err := migrationFiles()
	if err != nil {
		return err
	}

	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		var applied bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := applyMigration(ctx, pool, version, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func migrationFiles() ([]string, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !migrationName.MatchString(name) {
			return nil, fmt.Errorf("migration name %q must look like 001_name.sql", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no migrations embedded")
	}
	return names, nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, version, body string) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire for %s: %w", version, err)
	}
	defer conn.Release()

	batch := "BEGIN;\n" + body + "\nINSERT INTO schema_migrations (version) VALUES ('" + version + "');\nCOMMIT;\n"
	if _, err := conn.Conn().PgConn().Exec(ctx, batch).ReadAll(); err != nil {
		_, _ = conn.Conn().PgConn().Exec(ctx, "ROLLBACK").ReadAll()
		return fmt.Errorf("apply %s: %w", version, err)
	}
	return nil
}
