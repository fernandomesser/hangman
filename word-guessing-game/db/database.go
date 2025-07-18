package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", "./game.db")
	if err != nil {
		log.Fatal(err)
	}

	createUsers := `
	CREATE TABLE IF NOT EXISTS users (
    	id INTEGER PRIMARY KEY AUTOINCREMENT,
    	username TEXT UNIQUE NOT NULL,
    	password_hash TEXT NOT NULL
	);`
	_, err = DB.Exec(createUsers)
	if err != nil {
		log.Fatal(err)
	}

	createLeaderboard := `
	CREATE TABLE IF NOT EXISTS leaderboard (
		player TEXT PRIMARY KEY,
		wins INTEGER DEFAULT 0,
		best_score INTEGER
	);`
	_, err = DB.Exec(createLeaderboard)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database initialized.")
}
