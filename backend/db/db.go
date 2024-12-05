package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(path string) (*sql.DB, error) {
	// check if the database file exists
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Ensure database schema is set up
	if err := createSchema(db); err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}

	return db, nil
}

func createSchema(db *sql.DB) error {
	schema := `
    CREATE TABLE IF NOT EXISTS accounts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        email TEXT UNIQUE NOT NULL,
        password TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS mailboxes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mailbox_name TEXT NOT NULL,
		uid INTEGER NOT NULL,
		envelope BLOB NOT NULL,
		body_plain TEXT,
		body_html TEXT,
		body_raw BLOB,
		received_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(mailbox_name, uid)
	);
    `
	_, err := db.Exec(schema)
	return err
}
