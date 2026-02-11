package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	DB *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir sqlite dir: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	s := &SQLiteStore{DB: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) Close() error { return s.DB.Close() }

func (s *SQLiteStore) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS reset_tokens (
			token TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			expires_at INTEGER NOT NULL,
			used INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS pending_signups (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			email TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			approved INTEGER NOT NULL DEFAULT 0
		);`,
	}
	for _, q := range queries {
		if _, err := s.DB.Exec(q); err != nil {
			return err
		}
	}
	_, _ = s.DB.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now().Unix())
	_, _ = s.DB.Exec(`DELETE FROM reset_tokens WHERE expires_at < ? OR used = 1`, time.Now().Unix())
	return nil
}
