package db

import (
	"database/sql"
	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations.sql
var migrationsSQL string

//go:embed seed.sql
var seedSQL string

// Open opens the SQLite database at dbPath, enables foreign keys, and runs migrations.
func Open(dbPath string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		database.Close()
		return nil, err
	}

	if _, err := database.Exec(migrationsSQL); err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}

// Seed inserts demo data into the database. Idempotent due to INSERT OR IGNORE in seed.sql.
func Seed(database *sql.DB) error {
	_, err := database.Exec(seedSQL)
	return err
}
