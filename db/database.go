package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	DB         *sql.DB
	initDBOnce sync.Once
)

// getDBPath returns the appropriate database path based on environment
func getDBPath() string {
    if path := os.Getenv("DB_PATH"); path != "" {
        return path
    }
    if _, err := os.Stat("/data"); err == nil {
        return "/data/hangman.db"
    }
    return "./hangman.db" // local development fallback
}


// InitDB initializes the database connection and schema
func InitDB() {
	initDBOnce.Do(func() {
		dbPath := getDBPath()

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			log.Fatalf("Failed to create db directory: %v", err)
		}

		var err error
		DB, err = sql.Open("sqlite", dbPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}

		// Set connection pool settings
		DB.SetMaxOpenConns(1) // Important for SQLite in serverless
		DB.SetMaxIdleConns(1)
		DB.SetConnMaxLifetime(5 * time.Minute)

		// Enable WAL mode and other optimizations
		optimizationQueries := []string{
			"PRAGMA journal_mode=WAL;",
			"PRAGMA synchronous=NORMAL;",
			"PRAGMA busy_timeout=5000;",
			"PRAGMA foreign_keys=ON;",
		}

		for _, query := range optimizationQueries {
			if _, err := DB.Exec(query); err != nil {
				log.Printf("Warning: couldn't execute %q: %v", query, err)
			}
		}

		// Schema definition
		createTables := []string{
			`CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT UNIQUE NOT NULL,
				password_hash TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				last_login TIMESTAMP
			);`,
			`CREATE TABLE IF NOT EXISTS leaderboard (
				player INTEGER PRIMARY KEY,
				wins INTEGER DEFAULT 0,
				best_score INTEGER DEFAULT 0,
				games_played INTEGER DEFAULT 0,
				last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY(player) REFERENCES users(id) ON DELETE CASCADE
			);`,
			`CREATE TABLE IF NOT EXISTS games (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				word TEXT NOT NULL,
				guessed_letters TEXT DEFAULT '',
				remaining_attempts INTEGER DEFAULT 6,
				player_id INTEGER NOT NULL,
				status TEXT DEFAULT 'in_progress' CHECK(status IN ('in_progress', 'won', 'lost')),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				finished_at TIMESTAMP,
				FOREIGN KEY(player_id) REFERENCES users(id) ON DELETE CASCADE
			);`,
			`CREATE INDEX IF NOT EXISTS idx_games_player ON games(player_id);`,
			`CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);`,
			`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);`,
		}

		// Execute schema creation
		for _, table := range createTables {
			if _, err := DB.Exec(table); err != nil {
				log.Fatalf("Failed to create table: %v\nQuery: %s", err, table)
			}
		}

		// Verify foreign key support is working
		if _, err := DB.Exec("PRAGMA foreign_key_check;"); err != nil {
			log.Printf("Warning: foreign key constraint issue: %v", err)
		}

		fmt.Printf("Database initialized at: %s\n", dbPath)
	})
}

// CloseDB cleanly closes the database connection
func CloseDB() error {
	if DB != nil {
		// Final WAL checkpoint
		_, _ = DB.Exec("PRAGMA wal_checkpoint(FULL);")
		return DB.Close()
	}
	return nil
}
