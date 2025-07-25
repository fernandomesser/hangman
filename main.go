package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	handlers "wordgame/api"
	"wordgame/db"

	_ "modernc.org/sqlite"
)

func main() {
	// Seed global random for game IDs, word picking, AI, etc.
	rand.Seed(time.Now().UnixNano())

	// Ensure static files (JS/CSS/images) are present before starting.
	if _, err := os.Stat("./static"); os.IsNotExist(err) {
		log.Fatal("Static directory not found - required for JS/CSS assets")
	}

	// Initialize the DB and migrate schema, crash if it fails.
	db.InitDB()

	// Expose /static/ for frontend CSS/JS/assets.
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// ------------ ROUTE DEFINITIONS ----------------

	// Home page (welcome/login/register/go to game)
	http.HandleFunc("/", handlers.WelcomeHandler)

	// REGISTER: GET = show signup form, POST = process new user registration
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

	// LOGIN: GET = show login form, POST = process login
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

	// LOGOUT (always POST, but a GET works too): clear cookies, etc.
	http.HandleFunc("/logout", handlers.LogoutHandler)

	// GAME ACTION ROUTES:
	http.HandleFunc("/create", handlers.CreateGameHandler)       // Human vs. Human: create new game
	http.HandleFunc("/join", handlers.JoinGameHandler)           // Join existing multiplayer game
	http.HandleFunc("/create_ai", handlers.CreateAIHandler)      // Human vs. AI: create new AI game
	http.HandleFunc("/wait", handlers.WaitRoomHandler)           // Waiting room for 2nd player
	http.HandleFunc("/gameplay", handlers.GameplayHandler)       // Main game board/view
	http.HandleFunc("/guess", handlers.GuessHandler)             // (Deprecated: all guesses via WebSocket now!)
	http.HandleFunc("/state", handlers.StateHandler)             // For HTMX or polling-based live updates
	http.HandleFunc("/leaderboard", handlers.LeaderboardHandler) // Global stats/leaderboard
	http.HandleFunc("/hint", handlers.HintHandler)               // Hint API: get a hint (AJAX)
	http.HandleFunc("/ws", handlers.WebSocketHandler)            // WebSocket: multiplayer gameplay updates

	// Start background goroutine to relay messages from wsBroadcast (for live updates)
	handlers.StartWSBroadcaster()

	// -------- PORT DETECTION (Platform Adaptation) ---------
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port for local development
	}

	// Startup message
	fmt.Printf("Server running on port %s\n", port)
	// Start HTTP server; fatal on error.
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
