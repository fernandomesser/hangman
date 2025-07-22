package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	DB         *sql.DB
	initDBOnce sync.Once
)

// getDBPath returns the appropriate database path based on environment
func getDBPath() string {
	// For Vercel deployments
	if os.Getenv("VERCEL") == "1" {
		return "/tmp/hangman.db" // Vercel's writable tmp directory
	}
	// For local development
	return "./hangman.db"
}

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

		// Enable WAL mode for better concurrency
		if _, err := DB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
			log.Printf("Warning: couldn't enable WAL mode: %v", err)
		}

		// Set busy timeout
		if _, err := DB.Exec("PRAGMA busy_timeout=5000;"); err != nil {
			log.Printf("Warning: couldn't set busy timeout: %v", err)
		}

		createTables := []string{
			`CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				username TEXT UNIQUE NOT NULL,
				password_hash TEXT NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);`,
			`CREATE TABLE IF NOT EXISTS leaderboard (
				user_id INTEGER PRIMARY KEY,
				wins INTEGER DEFAULT 0,
				best_score INTEGER DEFAULT 0,
				games_played INTEGER DEFAULT 0,
				FOREIGN KEY(user_id) REFERENCES users(id)
			);`,
			`CREATE TABLE IF NOT EXISTS games (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				word TEXT NOT NULL,
				guessed_letters TEXT DEFAULT '',
				remaining_attempts INTEGER DEFAULT 6,
				player_id INTEGER,
				status TEXT DEFAULT 'in_progress',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				finished_at TIMESTAMP,
				FOREIGN KEY(player_id) REFERENCES users(id)
			);`,
			`CREATE INDEX IF NOT EXISTS idx_games_player ON games(player_id);`,
			`CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);`,
		}

		for _, table := range createTables {
			if _, err := DB.Exec(table); err != nil {
				log.Fatalf("Failed to create table: %v\nQuery: %s", err, table)
			}
		}

		fmt.Println("Database initialized at:", dbPath)
	})
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
