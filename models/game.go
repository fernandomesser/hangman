package models

type Game struct {
	ID                  string
	Word                string
	DisplayWord         string
	GuessedLetters      map[string]bool
	IncorrectGuesses    int
	MaxIncorrectGuesses int
	PlayerTurn          int
	Player1             string
	Player2             string
	Status              string // "waiting", "in_progress", "finished"
	Winner              string
	HasUsedHint         bool
	HintText            string
	GuessHistory        []string
}
