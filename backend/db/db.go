package db

import (
	"database/sql"
	"email_test_app/backend/auth"
	"fmt"
	"log"

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

func GetAccounts(db *sql.DB) (map[int64]auth.Account, error) {
	rows, err := db.Query("SELECT * FROM accounts")
	if err != nil {
		return nil, fmt.Errorf("error retrieving accounts: %w", err)
	}
	defer rows.Close()

	var accounts = make(map[int64]auth.Account)

	for rows.Next() {
		var account auth.Account
		var createdAt string
		if err := rows.Scan(&account.Id, &account.Email, &account.ImapUrl, &account.OAuthAccessToken, &account.OAuthRefreshToken, &account.OAuthExpiry, &account.AppSpecificPassword, &createdAt); err != nil {
			return nil, fmt.Errorf("error scanning account row: %w", err)
		}
		log.Println("Pulled account from DB:", account)
		accounts[account.Id] = account
	}

	return accounts, nil
}

func createSchema(db *sql.DB) error {
	schema := `
    CREATE TABLE IF NOT EXISTS accounts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        email TEXT UNIQUE NOT NULL,
		imap_url TEXT NOT NULL,
		oauth_access_token TEXT,
		oauth_refresh_token TEXT,
		oauth_expiry INTEGER,
		app_specific_password TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS mailboxes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL,
		name TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		mailbox_name TEXT NOT NULL,
		account_id INTEGER NOT NULL,
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
