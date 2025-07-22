package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"

	handlers "wordgame/api"
	"wordgame/db"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Verify static files directory exists
	if _, err := os.Stat("./static"); os.IsNotExist(err) {
		log.Fatal("Static directory not found - required for JS/CSS assets")
	}

	// Initialize the database
	db.InitDB()

	// Serve static files (JS, CSS, etc.)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Route handlers
	http.HandleFunc("/", handlers.WelcomeHandler)

	// Authentication routes
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.RegisterPage(w, r)
		case http.MethodPost:
			handlers.RegisterHandler(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.LoginPage(w, r)
		case http.MethodPost:
			handlers.LoginHandler(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/logout", handlers.LogoutHandler)

	// Game routes
	http.HandleFunc("/create", handlers.CreateGameHandler)
	http.HandleFunc("/join", handlers.JoinGameHandler)
	http.HandleFunc("/create_ai", handlers.CreateAIHandler)
	http.HandleFunc("/wait", handlers.WaitRoomHandler)
	http.HandleFunc("/gameplay", handlers.GameplayHandler)
	http.HandleFunc("/guess", handlers.GuessHandler)
	http.HandleFunc("/state", handlers.StateHandler) // HTMX updates
	http.HandleFunc("/leaderboard", handlers.LeaderboardHandler)
	http.HandleFunc("/hint", handlers.HintHandler)

	// Get port from environment (required for Vercel)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default for local development
	}

	// Start server
	fmt.Printf("Server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
