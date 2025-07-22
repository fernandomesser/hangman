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

func GetHint(game *models.Game) (string, error) {
	if game.HasUsedHint {
		return game.HintText, nil
	}

	unguessed := []string{}
	for _, c := range game.Word {
		letter := string(c)
		if !game.GuessedLetters[letter] {
			unguessed = append(unguessed, letter)
		}
	}

	if len(unguessed) == 0 {
		return "", fmt.Errorf("no unguessed letters left")
	}

	rand.Seed(time.Now().UnixNano())
	hintLetter := unguessed[rand.Intn(len(unguessed))]

	game.GuessedLetters[hintLetter] = true
	game.HasUsedHint = true
	game.HintText = fmt.Sprintf("Try the letter '%s'.", hintLetter)

	// Update display word
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

func AIGuess(game *models.Game) string {
	prompt := fmt.Sprintf(
		"You're playing Hangman. Known word: '%s'. Letters guessed: [%v]. Suggest ONE new lowercase letter (a-z) that has not been guessed.",
		strings.ReplaceAll(game.DisplayWord, " ", ""),
		guessedLettersList(game.GuessedLetters),
	)

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

	// Fallback: frequency-based
	fmt.Println(" Using fallback AI")
	frequency := "etaoinshrdlcumwfgypbvkjxqz"
	for _, l := range frequency {
		letter := string(l)
		if !game.GuessedLetters[letter] {
			fmt.Println("Fallback guess:", letter)
			return letter
		}
	}

	// Final fallback
	rand.Seed(time.Now().UnixNano())
	letter := string('a' + rune(rand.Intn(26)))
	fmt.Println("Random guess:", letter)
	return letter
}

// Gemini API call
func getAIGuessFromGemini(prompt string) (string, error) {
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent"
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("missing GEMINI_API_KEY")
	}

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
		return "", fmt.Errorf("Gemini API status %d: %s", resp.StatusCode, string(body))
	}

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
		return "", fmt.Errorf("Gemini responded but no text")
	}

	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

// Register letter and update state
func RegisterGuess(game *models.Game, letter string) error {
	letter = strings.ToLower(letter)

	if game.GuessedLetters[letter] {
		return fmt.Errorf("Letter '%s' has already been guessed", letter)
	}

	game.GuessedLetters[letter] = true

	newDisplay := ""
	for _, c := range game.Word {
		if game.GuessedLetters[string(c)] {
			newDisplay += string(c) + " "
		} else {
			newDisplay += "_ "
		}
	}
	game.DisplayWord = newDisplay

	if !strings.Contains(game.Word, letter) {
		game.IncorrectGuesses++
	}

	if !strings.Contains(game.DisplayWord, "_") {
		game.Status = "finished"
		if game.PlayerTurn == 1 {
			game.Winner = game.Player1
		} else {
			game.Winner = game.Player2
		}
	} else if game.IncorrectGuesses >= game.MaxIncorrectGuesses {
		game.Status = "finished"
		game.Winner = "Draw"
	} else {
		if game.PlayerTurn == 1 {
			game.PlayerTurn = 2
		} else {
			game.PlayerTurn = 1
		}
	}

	return nil
}

// Convert guessed letters map to comma-separated string
func guessedLettersList(m map[string]bool) string {
	letters := []string{}
	for l := range m {
		letters = append(letters, l)
	}
	return strings.Join(letters, ", ")
}
