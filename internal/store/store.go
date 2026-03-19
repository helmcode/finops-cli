package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultDBDir  = ".finops"
	defaultDBFile = "data.db"
)

// Store wraps the database connection and sqlc queries.
type Store struct {
	DB      *sql.DB
	Queries *Queries
}

// Open creates or opens the SQLite database at the default location (~/.finops/data.db)
// and runs any pending migrations.
func Open() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	dbPath := filepath.Join(homeDir, defaultDBDir, defaultDBFile)
	return OpenAt(dbPath)
}

// OpenAt creates or opens the SQLite database at the given path
// and runs any pending migrations.
func OpenAt(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating database directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode and foreign keys
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting journal mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	if err := RunMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Store{
		DB:      db,
		Queries: New(db),
	}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.DB.Close()
}

// Prune deletes cost records and resources older than the given retention period in months.
func (s *Store) Prune(retentionMonths int) (int64, error) {
	cutoff := time.Now().AddDate(0, -retentionMonths, 0).Format("2006-01-01")
	slog.Info("pruning records", "older_than", cutoff, "retention_months", retentionMonths)

	result, err := s.DB.Exec("DELETE FROM cost_records WHERE period_start < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("pruning cost records: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("getting rows affected: %w", err)
	}

	// Also prune commitments
	commitResult, err := s.DB.Exec("DELETE FROM commitments WHERE period_start < ?", cutoff)
	if err == nil {
		if commitDeleted, e := commitResult.RowsAffected(); e == nil && commitDeleted > 0 {
			slog.Info("pruned commitment records", "deleted", commitDeleted)
		}
	}

	slog.Info("pruned cost records", "deleted", deleted)
	return deleted, nil
}

// DBSize returns the database file size in bytes.
func (s *Store) DBSize() (int64, error) {
	var pageCount, pageSize int64
	if err := s.DB.QueryRow("PRAGMA page_count").Scan(&pageCount); err != nil {
		return 0, fmt.Errorf("getting page count: %w", err)
	}
	if err := s.DB.QueryRow("PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0, fmt.Errorf("getting page size: %w", err)
	}
	return pageCount * pageSize, nil
}
