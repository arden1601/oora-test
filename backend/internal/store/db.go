package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sort"

	_ "github.com/lib/pq" // PostgreSQL driver for database/sql

	"staffportal/migrations"
)

// ErrNotFound means the row asked for does not exist (or no longer counts).
var ErrNotFound = errors.New("not found")

// Store is the only code that talks to the database.
type Store struct {
	db *sql.DB
}

// New wraps an open database.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// Open connects to PostgreSQL and checks that it answers.
func Open(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("reach database: %w", err)
	}
	return db, nil
}

// Migrate applies every SQL file that has not been applied yet, in name order.
// Each file runs in its own transaction, so a failed file leaves nothing half-done.
func Migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var done bool
		err := db.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`, name).Scan(&done)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if done {
			continue
		}
		if err := applyMigration(ctx, db, name); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, name string) error {
	body, err := migrations.Files.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer tx.Rollback() // does nothing after a successful Commit

	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return fmt.Errorf("run migration %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// IsID reports whether s looks like one of our IDs (a UUID). Handlers check this
// before a query, so a malformed ID is a clean "not found" instead of a database error.
func IsID(s string) bool {
	return idPattern.MatchString(s)
}
