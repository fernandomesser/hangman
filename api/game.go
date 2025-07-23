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

// In-memory game store
var games = make(map[string]*models.Game)

// Helper: safely get logged-in user
func getUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie("user")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return "", false
	}
	return cookie.Value, true
}

// Helper: set game session cookies
func setGameCookies(w http.ResponseWriter, id, player, role string) {
	http.SetCookie(w, &http.Cookie{Name: "game_id", Value: id, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "player_name", Value: player, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "role", Value: role, Path: "/"}) // "1" or "2"
}

// Helper: ID generator
func generateGameID() string {
	rand.Seed(time.Now().UnixNano())
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Helper: parse int with fallback
func parseIntWithDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// Create Game (vs Human)
func CreateGameHandler(w http.ResponseWriter, r *http.Request) {
	player, ok := getUser(w, r)
	if !ok {
		return
	}

	r.ParseForm()
	wordLength := parseIntWithDefault(r.FormValue("word_length"), 5)
	maxGuesses := parseIntWithDefault(r.FormValue("max_guesses"), 7)
	word := words.GetRandomWord(wordLength)
	id := generateGameID()

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

	setGameCookies(w, id, player, "1")
	http.Redirect(w, r, "/wait", http.StatusSeeOther)
}

// Join Existing Game
func JoinGameHandler(w http.ResponseWriter, r *http.Request) {
	player, ok := getUser(w, r)
	if !ok {
		return
	}

	r.ParseForm()
	gameID := r.FormValue("game_id")
	game, found := games[gameID]

	if !found {
		// Game doesn't exist
		http.SetCookie(w, &http.Cookie{
			Name:  "error",
			Value: url.QueryEscape("Game not found."),
			Path:  "/",
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if game.Player2 != "" {
		// Game is full
		http.SetCookie(w, &http.Cookie{
			Name:  "error",
			Value: url.QueryEscape("Game already has two players."),
			Path:  "/",
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Success: join the game
	game.Player2 = player
	game.Status = "in_progress"
	setGameCookies(w, gameID, player, "2")
	http.Redirect(w, r, "/wait", http.StatusSeeOther)
}

// Create Game vs AI
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

// Wait Room Page
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
		html := fmt.Sprintf("<div class='center-box'><h2>Game ID: %s</h2><p>Waiting for player 2...</p><meta http-equiv='refresh' content='2'></div>", game.ID)
		w.Write([]byte(html))
	} else {
		http.Redirect(w, r, "/gameplay", http.StatusSeeOther)
	}
}

// Game View Handler (GET)
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

	data := map[string]interface{}{
		"Game":         game,
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
	}

	if errCookie, err := r.Cookie("error"); err == nil {
		if msg, decodeErr := url.QueryUnescape(errCookie.Value); decodeErr == nil {
			data["Error"] = msg
		}

		// Clear the error message after displaying it
		http.SetCookie(w, &http.Cookie{
			Name:   "error",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
	}

	utils.RenderPage(w, r, "gameplay.html", data)
}

func isPlayerTurn(r *http.Request, game *models.Game) bool {
	// Get the current player name from cookie
	cookie, err := r.Cookie("player_name")
	if err != nil {
		return false
	}
	player := cookie.Value

	// Determine which player it is and compare with whose turn it is
	if game.PlayerTurn == 1 && game.Player1 == player {
		return true
	}
	if game.PlayerTurn == 2 && game.Player2 == player {
		return true
	}
	return false
}

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

// Guess Handler (POST /guess)
func GuessHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	gameID, err := r.Cookie("game_id")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	game := games[gameID.Value]
	if game == nil || game.Status == "finished" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	letter := strings.ToLower(strings.TrimSpace(r.FormValue("letter")))
	if len(letter) != 1 || letter < "a" || letter > "z" {
		http.Redirect(w, r, "/gameplay", http.StatusSeeOther)
		return
	}

	if game.GuessedLetters[letter] {
		http.SetCookie(w, &http.Cookie{
			Name:  "error",
			Value: url.QueryEscape(fmt.Sprintf("Letter '%s' has already been guessed.", letter)),
			Path:  "/",
		})
		http.Redirect(w, r, "/gameplay", http.StatusSeeOther)
		return
	}

	// Register the new guess
	logic.RegisterGuess(game, letter)

	// If it's a game vs AI
	if game.Status != "finished" && game.Player2 == "Computer" && game.PlayerTurn == 2 {
		ai := logic.AIGuess(game)
		logic.RegisterGuess(game, ai)
	}

	// If the game ended, update leaderboard
	if game.Status == "finished" && game.Winner != "Draw" {
		updateLeaderboard(game.Winner, game.IncorrectGuesses)
	}

	http.Redirect(w, r, "/gameplay", http.StatusSeeOther)
}

// Build Game State for Template
func buildGameState(game *models.Game, role string) map[string]interface{} {
	correct := []string{}
	wrong := []string{}
	for l := range game.GuessedLetters {
		if strings.Contains(game.Word, l) {
			correct = append(correct, l)
		} else {
			wrong = append(wrong, l)
		}
	}
	sort.Strings(correct)
	sort.Strings(wrong)

	return map[string]interface{}{
		"DisplayWord":  game.DisplayWord,
		"Remaining":    game.MaxIncorrectGuesses - game.IncorrectGuesses,
		"Correct":      strings.Join(correct, ", "),
		"Wrong":        strings.Join(wrong, ", "),
		"GameOver":     game.Status == "finished",
		"Winner":       game.Winner,
		"Word":         game.Word, // ✅ pass the correct word
		"IsPlayerTurn": role == fmt.Sprintf("%d", game.PlayerTurn),
	}
}

// Replace with your dynamic game UI rendering logic
func generatePartialGameHTML(game *models.Game, role string) string {
	// Example: HTML snippet with word and remaining guesses
	remaining := game.MaxIncorrectGuesses - game.IncorrectGuesses
	return fmt.Sprintf(`
		<h3>Word: %s</h3>
		<p>Remaining guesses: %d</p>
	`, game.DisplayWord, remaining)
}

func updateLeaderboard(winner string, incorrectGuesses int) {
	var userID int
	err := db.DB.QueryRow("SELECT id FROM users WHERE username = ?", winner).Scan(&userID)
	if err != nil {
		fmt.Println("Leaderboard update error: could not find user", winner)
		return
	}

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

	// Prepare correct and wrong letters
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

	data := map[string]interface{}{
		"DisplayWord": game.DisplayWord,
		"Remaining":   game.MaxIncorrectGuesses - game.IncorrectGuesses,
		"Correct":     strings.Join(correct, ", "),
		"Wrong":       strings.Join(wrong, ", "),
		"HasUsedHint": game.HasUsedHint,
		"HintText":    game.HintText,
		"Status":      game.Status,
		"Winner":      game.Winner,
		"Word":        game.Word,
	}

	utils.RenderPartial(w, r, "gameplay.html", data) // Renders only the dynamic block
}
