package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"wordgame/db"
	"wordgame/logic"
	"wordgame/models"
	"wordgame/words"
)

// In-memory game store
var games = make(map[string]*models.Game)

// ✅ Helper: safely get logged-in user
func getUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie("user")
	if err != nil || cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return "", false
	}
	return cookie.Value, true
}

// ✅ Helper: set game session cookies
func setGameCookies(w http.ResponseWriter, id, player, role string) {
	http.SetCookie(w, &http.Cookie{Name: "game_id", Value: id, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "player_name", Value: player, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "role", Value: role, Path: "/"}) // "1" or "2"
}

// ✅ Helper: ID generator
func generateGameID() string {
	rand.Seed(time.Now().UnixNano())
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// ✅ Helper: parse int with fallback
func parseIntWithDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// ✅ Create Game (vs Human)
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

// ✅ Join Existing Game
func JoinGameHandler(w http.ResponseWriter, r *http.Request) {
	player, ok := getUser(w, r)
	if !ok {
		return
	}

	r.ParseForm()
	gameID := r.FormValue("game_id")
	game, found := games[gameID]
	if !found || game.Player2 != "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	game.Player2 = player
	game.Status = "in_progress"
	setGameCookies(w, gameID, player, "2")
	http.Redirect(w, r, "/wait", http.StatusSeeOther)
}

// ✅ Create Game vs AI
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

// ✅ Wait Room Page
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

// ✅ Game View Handler (GET)
func GameplayHandler(w http.ResponseWriter, r *http.Request) {
	gameIDCookie, err := r.Cookie("game_id")
	roleCookie, err2 := r.Cookie("role")
	if err != nil || err2 != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	game := games[gameIDCookie.Value]
	if game == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	data := buildGameState(game, roleCookie.Value)
	renderPage(w, r, "gameplay.html", data)
}

// ✅ Guess Handler (POST /guess)
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
	if len(letter) != 1 || letter < "a" || letter > "z" || game.GuessedLetters[letter] {
		http.Redirect(w, r, "/gameplay", http.StatusSeeOther)
		return
	}

	logic.RegisterGuess(game, letter)

	if game.Status != "finished" && game.Player2 == "Computer" && game.PlayerTurn == 2 {
		ai := logic.AIGuess(game)
		logic.RegisterGuess(game, ai)
	}

	http.Redirect(w, r, "/gameplay", http.StatusSeeOther)

	if game.Status == "finished" && game.Winner != "Draw" {
		updateLeaderboard(game.Winner, game.IncorrectGuesses)
	}

}

// ✅ Build Game State for Template
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

func StateHandler(w http.ResponseWriter, r *http.Request) {
	gameIDCookie, err := r.Cookie("game_id")
	roleCookie, err2 := r.Cookie("role")
	if err != nil || err2 != nil {
		http.Error(w, "Missing session cookies", http.StatusBadRequest)
		return
	}

	game := games[gameIDCookie.Value]
	if game == nil {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	html := generatePartialGameHTML(game, roleCookie.Value)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

// 💡 Replace with your dynamic game UI rendering logic
func generatePartialGameHTML(game *models.Game, role string) string {
	// Example: HTML snippet with word and remaining guesses
	remaining := game.MaxIncorrectGuesses - game.IncorrectGuesses
	return fmt.Sprintf(`
		<h3>Word: %s</h3>
		<p>Remaining guesses: %d</p>
	`, game.DisplayWord, remaining)
}

func updateLeaderboard(winner string, incorrectGuesses int) {
	_, err := db.DB.Exec(`
		INSERT INTO leaderboard (player, wins, best_score)
		VALUES (?, 1, ?)
		ON CONFLICT(player) DO UPDATE SET
			wins = wins + 1,
			best_score = CASE
				WHEN best_score IS NULL OR ? < best_score THEN ?
				ELSE best_score END
	`, winner, incorrectGuesses, incorrectGuesses, incorrectGuesses)

	if err != nil {
		fmt.Println("Leaderboard update error:", err)
	}
}
