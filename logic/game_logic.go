package logic

import (
	"math/rand"
	"strings"
	"time"
	"wordgame/models"
)

const AIPlayerName = "Computer"

func RegisterGuess(game *models.Game, letter string) {
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
}

func AIGuess(game *models.Game) string {
	frequency := "etaoinshrdlcumwfgypbvkjxqz"

	for _, l := range frequency {
		letter := string(l)
		if !game.GuessedLetters[letter] {
			return letter
		}
	}

	rand.Seed(time.Now().UnixNano())
	return string('a' + rune(rand.Intn(26)))
}
