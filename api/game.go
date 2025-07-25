package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"wordgame/db"
	"wordgame/logic"
	"wordgame/models"
	"wordgame/utils"
	"wordgame/words"
)

// Broadcast channel for websocket messages; buffered for up to 16 messages
var wsBroadcast = make(chan WSMessage, 16)

// In-memory map to hold all games; key is game ID, value: pointer to game struct
var games = make(map[string]*models.Game)

// Helper: Retrieve the current logged-in user from cookie.
// If missing or invalid, redirect to login and return ("", false).
func getUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie("user")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return "", false
	}
	return cookie.Value, true
}

// Helper: Set cookies for game state (game id, player name, player role ID, e.g. "1" or "2")
func setGameCookies(w http.ResponseWriter, id, player, role string) {
	http.SetCookie(w, &http.Cookie{Name: "game_id", Value: id, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "player_name", Value: player, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "role", Value: role, Path: "/"})
}

// Helper: Create a unique 4-letter game ID from random lower-case letters
func generateGameID() string {
	rand.Seed(time.Now().UnixNano())
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Helper: Convert string to int, with fallback to default if not valid/positive
func parseIntWithDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// HTTP POST handler: create new HUMAN-vs-HUMAN game
func CreateGameHandler(w http.ResponseWriter, r *http.Request) {
	// Check login & get player name
	player, ok := getUser(w, r)
	if !ok {
		return
	}

	r.ParseForm() // Parse POST form fields

	// Get word length & guesses, falling back to defaults
	wordLength := parseIntWithDefault(r.FormValue("word_length"), 5)
	maxGuesses := parseIntWithDefault(r.FormValue("max_guesses"), 7)
	word := words.GetRandomWord(wordLength)
	id := generateGameID()

	// Store new game in memory
	games[id] = &models.Game{
		ID:                  id,
		Word:                word,
		DisplayWord:         strings.Repeat("_ ", len(word)),
		GuessedLetters:      make(map[string]bool),
		MaxIncorrectGuesses: maxGuesses,
		Player1:             player,
		PlayerTurn:          1,
		Status:              "waiting",
	}

	setGameCookies(w, id, player, "1")                // Set player 1 role cookies
	http.Redirect(w, r, "/wait", http.StatusSeeOther) // Go to waiting room
}

// HTTP POST handler: join an existing two-player game
func JoinGameHandler(w http.ResponseWriter, r *http.Request) {
	player, ok := getUser(w, r)
	if !ok {
		return
	}

	r.ParseForm()
	gameID := r.FormValue("game_id")
	game, found := games[gameID]
	if !found {
		// Set error message and redirect if can't find game
		http.SetCookie(w, &http.Cookie{
			Name:  "error",
			Value: url.QueryEscape("Game not found."),
			Path:  "/",
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if game.Player2 != "" {
		// Already has two players
		http.SetCookie(w, &http.Cookie{
			Name:  "error",
			Value: url.QueryEscape("Game already has two players."),
			Path:  "/",
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Add player2 and start game
	game.Player2 = player
	game.Status = "in_progress"
	setGameCookies(w, gameID, player, "2")
	http.Redirect(w, r, "/wait", http.StatusSeeOther)
}

// HTTP POST handler: create new game against AI (always instant start)
func CreateAIHandler(w http.ResponseWriter, r *http.Request) {
	player, ok := getUser(w, r)
	if !ok {
		return
	}

	r.ParseForm()
	wordLength := parseIntWithDefault(r.FormValue("word_length"), 5)
	maxGuesses := parseIntWithDefault(r.FormValue("max_guesses"), 7)
	word := words.GetRandomWord(wordLength)
	id := generateGameID()

	// Note Player2 is "Computer" and status is "in_progress" immediately
	games[id] = &models.Game{
		ID:                  id,
		Word:                word,
		DisplayWord:         strings.Repeat("_ ", len(word)),
		GuessedLetters:      make(map[string]bool),
		MaxIncorrectGuesses: maxGuesses,
		Player1:             player,
		Player2:             "Computer",
		PlayerTurn:          1,
		Status:              "in_progress",
	}

	setGameCookies(w, id, player, "1")
	http.Redirect(w, r, "/gameplay", http.StatusSeeOther)
}

// Wait room handler: shows "waiting for player 2", or advances if ready
func WaitRoomHandler(w http.ResponseWriter, r *http.Request) {
	gameIDCookie, err := r.Cookie("game_id")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	game, ok := games[gameIDCookie.Value]
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if game.Player2 == "" && game.Player2 != "Computer" {
		// Still waiting for second player
		data := map[string]interface{}{
			"GameID":  game.ID,
			"Player1": game.Player1,
		}
		utils.RenderPage(w, r, "waiting.html", data)
	} else {
		// Ready to play
		http.Redirect(w, r, "/gameplay", http.StatusSeeOther)
	}
}

// Render the main game view (game board, etc)
func GameplayHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("game_id")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	game, ok := games[cookie.Value]
	if !ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	lastGuess := ""
	if len(game.GuessHistory) > 0 {
		lastGuess = game.GuessHistory[len(game.GuessHistory)-1]
	}

	// Build data for template: game state, guess history, winner, etc.
	data := map[string]interface{}{
		"Game":         game,
		"Player1":      game.Player1,
		"Player2":      game.Player2,
		"Word":         game.Word,
		"DisplayWord":  game.DisplayWord,
		"Remaining":    game.MaxIncorrectGuesses - game.IncorrectGuesses,
		"Correct":      getCorrectLetters(game),
		"Wrong":        getWrongLetters(game),
		"IsPlayerTurn": isPlayerTurn(r, game),
		"GameOver":     game.Status == "finished",
		"Winner":       game.Winner,
		"HasUsedHint":  game.HasUsedHint,
		"HintText":     game.HintText,
		"LastGuess":    lastGuess,
	}

	// Get and display any error messages, then clear the cookie
	if errCookie, err := r.Cookie("error"); err == nil {
		if msg, decodeErr := url.QueryUnescape(errCookie.Value); decodeErr == nil {
			data["Error"] = msg
		}
		http.SetCookie(w, &http.Cookie{
			Name: "error", Value: "", Path: "/", MaxAge: -1,
		})
	}

	utils.RenderPage(w, r, "gameplay.html", data)
}

// Return true if the current player (from cookie) is the one whose turn it is
func isPlayerTurn(r *http.Request, game *models.Game) bool {
	cookie, err := r.Cookie("player_name")
	if err != nil {
		return false
	}
	player := cookie.Value
	// Compare which player turn we are at
	return (game.PlayerTurn == 1 && game.Player1 == player) ||
		(game.PlayerTurn == 2 && game.Player2 == player)
}

// Retrieve list of correct guessed letters, sorted alphabetically, as a string with commas
func getCorrectLetters(game *models.Game) string {
	letters := []string{}
	for letter := range game.GuessedLetters {
		if strings.Contains(game.Word, letter) {
			letters = append(letters, letter)
		}
	}
	sort.Strings(letters)
	return strings.Join(letters, ", ")
}

// Retrieve list of wrong guessed letters, sorted alphabetically, as a string
func getWrongLetters(game *models.Game) string {
	letters := []string{}
	for letter := range game.GuessedLetters {
		if !strings.Contains(game.Word, letter) {
			letters = append(letters, letter)
		}
	}
	sort.Strings(letters)
	return strings.Join(letters, ", ")
}

// (Deprecating) HTTP handler: disallow traditional guessing, as guessing should be real-time via WebSockets!
func GuessHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Use WebSocket to guess", http.StatusMethodNotAllowed)
}

// Helper: build the per-game, per player template state as a map
func buildGameState(game *models.Game, role string) map[string]interface{} {
	correct, wrong := []string{}, []string{}
	for l := range game.GuessedLetters {
		if strings.Contains(game.Word, l) {
			correct = append(correct, l)
		} else {
			wrong = append(wrong, l)
		}
	}
	sort.Strings(correct)
	sort.Strings(wrong)
	lastGuess := ""
	if len(game.GuessHistory) > 0 {
		lastGuess = game.GuessHistory[len(game.GuessHistory)-1]
	}
	return map[string]interface{}{
		"DisplayWord":  game.DisplayWord,
		"Remaining":    game.MaxIncorrectGuesses - game.IncorrectGuesses,
		"Correct":      strings.Join(correct, ", "),
		"Wrong":        strings.Join(wrong, ", "),
		"GameOver":     game.Status == "finished",
		"Winner":       game.Winner,
		"Word":         game.Word,
		"IsPlayerTurn": role == fmt.Sprintf("%d", game.PlayerTurn),
		"LastGuess":    lastGuess,
	}
}

// Update (or insert) leaderboard for player; increments win, tracks best score for user (fewest incorrect guesses)
func updateLeaderboard(winner string, incorrectGuesses int) {
	var userID int
	err := db.DB.QueryRow("SELECT id FROM users WHERE username = ?", winner).Scan(&userID)
	if err != nil {
		// User not found, log but continue
		fmt.Println("Leaderboard update error: could not find user", winner)
		return
	}
	// Uses upsert: on conflict, increase wins, update best_score if improved
	_, err = db.DB.Exec(`
        INSERT INTO leaderboard (player, wins, best_score)
        VALUES (?, 1, ?)
        ON CONFLICT(player) DO UPDATE SET
            wins = wins + 1,
            best_score = CASE WHEN best_score IS NULL OR ? < best_score THEN ? ELSE best_score END
    `, userID, incorrectGuesses, incorrectGuesses, incorrectGuesses)
	if err != nil {
		fmt.Println("Leaderboard update error:", err)
	}
}

// Give a hint to the current player, if none used yet, using logic.GetHint
func HintHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("game_id")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Missing game ID", http.StatusBadRequest)
		return
	}
	game, ok := games[cookie.Value]
	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	if game.HasUsedHint {
		// Already has hint, just return it again
		fmt.Fprint(w, game.HintText)
		return
	}
	hint, err := logic.GetHint(game)
	if err != nil {
		http.Error(w, "Hint unavailable", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, hint)
}

// State endpoint: Used for HTMX/live updates, returns slice of game state for the current session
func StateHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("game_id")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Missing game ID", http.StatusBadRequest)
		return
	}
	gameID := cookie.Value
	game, ok := games[gameID]
	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	if game.Player2 != "" && game.Status == "in_progress" {
		// If both players ready and in progress, tell HTMX client to redirect to main gameplay
		w.Header().Set("HX-Redirect", "/gameplay")
		return
	}

	// Gather correct and wrong guesses
	var correct, wrong []string
	for letter, guessed := range game.GuessedLetters {
		if guessed {
			if strings.Contains(game.Word, letter) {
				correct = append(correct, letter)
			} else {
				wrong = append(wrong, letter)
			}
		}
	}

	// Get player role from cookie if present
	role := ""
	roleCookie, err := r.Cookie("role")
	if err == nil {
		role = roleCookie.Value
	}
	isPlayerTurn := role == fmt.Sprintf("%d", game.PlayerTurn)

	// Build state dictionary for template/partial rendering
	data := map[string]interface{}{
		"GameID":       game.ID,
		"Player1":      game.Player1,
		"Player2":      game.Player2,
		"DisplayWord":  game.DisplayWord,
		"Remaining":    game.MaxIncorrectGuesses - game.IncorrectGuesses,
		"Correct":      strings.Join(correct, ", "),
		"Wrong":        strings.Join(wrong, ", "),
		"HasUsedHint":  game.HasUsedHint,
		"HintText":     game.HintText,
		"Status":       game.Status,
		"GameOver":     game.Status == "finished",
		"Winner":       game.Winner,
		"Word":         game.Word,
		"IsPlayerTurn": isPlayerTurn,
	}

	// If still waiting, render waiting or, if player2 just joined, send client redirect to /gameplay
	if game.Status == "waiting" {
		if game.Player2 != "" {
			fmt.Fprint(w, `<script>window.location.replace("/gameplay");</script>`)
		} else {
			utils.RenderPartial(w, r, "waiting.html", data)
		}
		return
	}

	// Game is in progress or finished; render/update gameplay partial
	utils.RenderPartial(w, r, "gameplay.html", data)
}
