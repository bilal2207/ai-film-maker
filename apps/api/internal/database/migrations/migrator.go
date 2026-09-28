package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

//go:embed sql/*.sql
var migrationFS embed.FS

type Migration struct {
	Version string
	Name    string
	SQL     string
}

func GetMigrations() ([]Migration, error) {
	entries, err := migrationFS.ReadDir("sql")
	if err != nil {
		return nil, fmt.Errorf("failed to read migration files: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := migrationFS.ReadFile("sql/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", entry.Name(), err)
		}

		parts := strings.SplitN(entry.Name(), "_", 2)
		version := parts[0]

		migrations = append(migrations, Migration{
			Version: version,
			Name:    entry.Name(),
			SQL:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func Run(ctx context.Context, db *sql.DB) error {
	// 1. Ensure schema_migrations table exists
	initTableSQL := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.ExecContext(ctx, initTableSQL); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	// 2. Fetch applied migrations
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error reading applied migrations: %w", err)
	}

	// 3. Apply pending migrations
	migrations, err := GetMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}

		log.Printf("Applying database migration %s (%s)...", m.Version, m.Name)

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", m.Version, err)
		}

		if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", m.Version, err)
		}

		insertSQL := "INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3)"
		if _, err := tx.ExecContext(ctx, insertSQL, m.Version, m.Name, time.Now().UTC()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.Version, err)
		}

		log.Printf("Successfully applied migration %s.", m.Version)
	}

	return nil
}
