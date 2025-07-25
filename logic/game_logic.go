package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"wordgame/models"
)

const AIPlayerName = "Computer"

// -------- HINT LOGIC --------

// Returns a hint for the player by revealing a random, as-yet-unguessed letter.
// Only gives one hint per game (tracks HasUsedHint).
func GetHint(game *models.Game) (string, error) {
	// If already used, just return old hint.
	if game.HasUsedHint {
		return game.HintText, nil
	}

	// Collect all letters in the word that have NOT been guessed yet.
	unguessed := []string{}
	for _, c := range game.Word {
		letter := string(c)
		if !game.GuessedLetters[letter] {
			unguessed = append(unguessed, letter)
		}
	}

	// No unguessed letters left? Edge case: don't suggest if word is fully revealed.
	if len(unguessed) == 0 {
		return "", fmt.Errorf("no unguessed letters left")
	}

	// Randomly choose one unguessed letter to recommend.
	rand.Seed(time.Now().UnixNano())
	hintLetter := unguessed[rand.Intn(len(unguessed))]

	game.HasUsedHint = true
	game.HintText = fmt.Sprintf("Try the letter '%s'.", hintLetter)

	// Update display word (for consistency in UI after hint; may not actually reveal the letter instantly).
	newDisplay := ""
	for _, c := range game.Word {
		if game.GuessedLetters[string(c)] {
			newDisplay += string(c) + " "
		} else {
			newDisplay += "_ "
		}
	}
	game.DisplayWord = newDisplay

	return game.HintText, nil
}

// -------- AI GUESSING LOGIC --------

// Returns the next letter for the AI to guess, using Gemini AI or falling back to English letter frequency.
func AIGuess(game *models.Game) string {
	// Prepare prompt summarizing game state for the AI chatbot.
	prompt := fmt.Sprintf(
		"You're playing Hangman. Known word: '%s'. Letters guessed: [%v]. Suggest ONE new lowercase letter (a-z) that has not been guessed.",
		strings.ReplaceAll(game.DisplayWord, " ", ""),
		guessedLettersList(game.GuessedLetters),
	)

	// Call Gemini AI API with prompt
	aiGuess, err := getAIGuessFromGemini(prompt)
	if err != nil {
		fmt.Println(" Gemini error:", err)
	} else {
		aiGuess = strings.ToLower(strings.TrimSpace(aiGuess))
		if len(aiGuess) > 0 {
			guess := string([]rune(aiGuess)[0])
			if guess >= "a" && guess <= "z" && !game.GuessedLetters[guess] {
				fmt.Println(" Gemini guess used:", guess)
				return guess
			}
			fmt.Println("Gemini guessed invalid or duplicate letter:", guess)
		}
	}

	// Fallback: frequency-based guessing if Gemini fails or gives nonsense
	fmt.Println(" Using fallback AI")
	frequency := "etaoinshrdlcumwfgypbvkjxqz" // most common English letters, in order
	for _, l := range frequency {
		letter := string(l)
		if !game.GuessedLetters[letter] {
			fmt.Println("Fallback guess:", letter)
			return letter
		}
	}

	// Very unlikely: if even frequency letters exhausted, random guess as last resort
	rand.Seed(time.Now().UnixNano())
	letter := string('a' + rune(rand.Intn(26)))
	fmt.Println("Random guess:", letter)
	return letter
}

// -------- GEMINI API INTEGRATION --------

// Returns a one-letter Hangman guess from Gemini (calls Google Generative Language API) or an error.
func getAIGuessFromGemini(prompt string) (string, error) {
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent"
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("missing GEMINI_API_KEY")
	}

	// Gemini JSON structure
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}
	data, _ := json.Marshal(reqBody)

	// Build POST request
	req, err := http.NewRequest("POST", endpoint+"?key="+apiKey, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// If API returns error, propagate error plus Gemini debug text
		return "", fmt.Errorf("gemini api status %d: %s", resp.StatusCode, string(body))
	}

	// Minimal (but robust) structure for Gemini JSON response
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini responded but no text")
	}

	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

// -------- GAMEPLAY LOGIC --------

// Registers a player's guess (letter) into the game state.
// Updates guessed list, display word, guesses counter, turn info, and winner.
// Returns error if letter has already been guessed.
func RegisterGuess(game *models.Game, letter string) error {
	letter = strings.ToLower(letter) // Standardize letter

	if game.GuessedLetters[letter] {
		return fmt.Errorf("letter '%s' has already been guessed", letter)
	}

	// Mark letter as guessed
	game.GuessedLetters[letter] = true
	game.GuessHistory = append(game.GuessHistory, letter)
	// Rebuild the display word, with spaces separating revealed letters and underscores for missing ones
	newDisplay := ""
	for _, c := range game.Word {
		if game.GuessedLetters[string(c)] {
			newDisplay += string(c) + " "
		} else {
			newDisplay += "_ "
		}
	}
	game.DisplayWord = newDisplay

	// If guess was wrong, increment incorrect guess count
	if !strings.Contains(game.Word, letter) {
		game.IncorrectGuesses++
	}

	// Winning condition: no underscores left? (fully revealed)
	if !strings.Contains(game.DisplayWord, "_") {
		game.Status = "finished"
		if game.PlayerTurn == 1 {
			game.Winner = game.Player1
		} else {
			game.Winner = game.Player2
		}
	} else if game.IncorrectGuesses >= game.MaxIncorrectGuesses {
		// Losing condition: too many wrong guesses
		game.Status = "finished"
		game.Winner = "Draw"
	} else {
		// Otherwise, switch turns
		if game.PlayerTurn == 1 {
			game.PlayerTurn = 2
		} else {
			game.PlayerTurn = 1
		}
	}

	return nil
}

// Converts a map[string]bool of guessed letters to a comma-separated "a, b, c" string.
// Used for creating human-readable guess history for AI prompts, debugging, etc.
func guessedLettersList(m map[string]bool) string {
	letters := []string{}
	for l := range m {
		letters = append(letters, l)
	}
	return strings.Join(letters, ", ")
}
