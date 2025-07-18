package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

func GenerateID() string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GetWord(length int) string {
	url := fmt.Sprintf("https://random-word-api.herokuapp.com/word?length=%d", length)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Word API error:", err)
		return strings.Repeat("a", length)
	}
	defer resp.Body.Close()

	var words []string
	err = json.NewDecoder(resp.Body).Decode(&words)
	if err != nil || len(words) == 0 {
		return strings.Repeat("a", length)
	}

	return words[0]
}
