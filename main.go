package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	_ "modernc.org/sqlite"

	handlers "wordgame/api"
	"wordgame/db"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// Initialize the database and schema
	db.InitDB()

	// Serve HTML, JS, CSS, etc.
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Home & public leaderboard
	http.HandleFunc("/", handlers.WelcomeHandler)

	// Route-safe REGISTER handler (GET + POST)
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

	// Route-safe LOGIN handler (GET + POST)
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

	// Logout
	http.HandleFunc("/logout", handlers.LogoutHandler)

	// Game handlers (authentication required inside handlers)
	http.HandleFunc("/create", handlers.CreateGameHandler)
	http.HandleFunc("/join", handlers.JoinGameHandler)
	http.HandleFunc("/create_ai", handlers.CreateAIHandler)
	http.HandleFunc("/wait", handlers.WaitRoomHandler)
	http.HandleFunc("/gameplay", handlers.GameplayHandler)
	http.HandleFunc("/guess", handlers.GuessHandler)
	http.HandleFunc("/state", handlers.StateHandler) // For htmx updates
	http.HandleFunc("/leaderboard", handlers.LeaderboardHandler)
	http.HandleFunc("/hint", handlers.HintHandler)

	// Start server
	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
