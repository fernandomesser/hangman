package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite" // Import SQLite driver for side-effects (registration)
)

// -- GLOBALS --

// DB is the global SQL database handle for your app.
// Only one connection is used/supported (see InitDB).
var (
	DB         *sql.DB   // Database handle, shared by all code
	initDBOnce sync.Once // Ensures database is initialized only once (thread-safe)
)

// getDBPath determines which database file to use based on environment.
// Looks for DB_PATH env var, then Docker/Railway-style /data, else local dir.
func getDBPath() string {
	// 1. User-specified in environment
	envPath := os.Getenv("DB_PATH")
	if envPath != "" {
		return envPath
	}
	// 2. Shared persistent volume, e.g. Railway/Docker /data directory
	if _, err := os.Stat("/data"); err == nil {
		return "/data/hangman.db"
	}
	// 3. Local fallback (project directory)
	return "./hangman.db"
}

// InitDB initializes (and, if needed, creates) the database file, schema, and optimizations.
// Safe to call multiple times (uses sync.Once).
func InitDB() {
	initDBOnce.Do(func() {
		dbPath := getDBPath() // Figure out DB location

		// Ensure parent directory exists (for custom/volume paths).
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			log.Fatalf("Failed to create db directory: %v", err)
		}

		var err error
		// Open (or create) the SQLite database at the chosen path.
		DB, err = sql.Open("sqlite", dbPath)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}

		// SQLite is not fully concurrent. These limits guarantee safety:
		DB.SetMaxOpenConns(1) // Only one open connection allowed (very important for SQLite)
		DB.SetMaxIdleConns(1)
		DB.SetConnMaxLifetime(5 * time.Minute)

		// Performance and durability optimization queries for SQLite:
		optimizationQueries := []string{
			"PRAGMA journal_mode=WAL;",   // Fast WAL mode supports concurrent reads with one writer
			"PRAGMA synchronous=NORMAL;", // Faster than full synchronous, safe for most servers
			"PRAGMA busy_timeout=5000;",  // Wait 5s if the DB is locked before returning error
			"PRAGMA foreign_keys=ON;",    // Enforce referential integrity across tables
		}

		// Try all the PRAGMAs; log warnings but do not exit fatally.
		for _, query := range optimizationQueries {
			if _, err := DB.Exec(query); err != nil {
				log.Printf("Warning: couldn't execute %q: %v", query, err)
			}
		}

		// -- SCHEMA DEFINITION --
		createTables := []string{
			// USERS: id, username, password (hashed), time data
			`CREATE TABLE IF NOT EXISTS users (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                username TEXT UNIQUE NOT NULL,
                password_hash TEXT NOT NULL,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                last_login TIMESTAMP
            );`,
			// LEADERBOARD: one row per player (by user ID), win stats, performance stats
			`CREATE TABLE IF NOT EXISTS leaderboard (
                player INTEGER PRIMARY KEY,
                wins INTEGER DEFAULT 0,
                best_score INTEGER DEFAULT 0,
                games_played INTEGER DEFAULT 0,
                last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY(player) REFERENCES users(id) ON DELETE CASCADE
            );`,
			// GAMES: all played games, with guessed letters stored as text (can be parsed as a list), win/loss, player association
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
			// Indexes to accelerate common queries (stats by player, lookup by username, filtering by game state)
			`CREATE INDEX IF NOT EXISTS idx_games_player ON games(player_id);`,
			`CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);`,
			`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);`,
		}

		// Create all tables and indexes. If any fail, crash immediately.
		for _, table := range createTables {
			if _, err := DB.Exec(table); err != nil {
				log.Fatalf("Failed to create table: %v\nQuery: %s", err, table)
			}
		}

		// Post-migration safety check: foreign key integrity
		if _, err := DB.Exec("PRAGMA foreign_key_check;"); err != nil {
			log.Printf("Warning: foreign key constraint issue: %v", err)
		}

		fmt.Printf("Database initialized at: %s\n", dbPath)
	})
}

// CloseDB cleanly closes the global DB connection when shutting down.
// Performs a final WAL checkpoint to persist all writes.
func CloseDB() error {
	if DB != nil {
		// Flush any outstanding data in the Write-Ahead Log to the main database file.
		_, _ = DB.Exec("PRAGMA wal_checkpoint(FULL);")
		return DB.Close()
	}
	return nil
}
