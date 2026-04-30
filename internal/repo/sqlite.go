package repo

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type SQLiteDB struct {
	DB *sql.DB
}

func OpenSQLite(_ context.Context, dbPath string) (*SQLiteDB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := runSQLiteMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run sqlite migrations: %w", err)
	}

	return &SQLiteDB{DB: db}, nil
}

func (s *SQLiteDB) Close() {
	if s != nil && s.DB != nil {
		s.DB.Close()
	}
}

func (s *SQLiteDB) Ready(_ context.Context) error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Ping()
}

func runSQLiteMigrations(db *sql.DB) error {
	for _, ddl := range sqliteMigrations {
		if _, err := db.Exec(ddl); err != nil {
			if shouldIgnoreSQLiteMigrationError(err) {
				continue
			}
			return fmt.Errorf("migration: %w", err)
		}
	}
	return nil
}

func shouldIgnoreSQLiteMigrationError(err error) bool {
	return strings.Contains(err.Error(), "duplicate column name:")
}
