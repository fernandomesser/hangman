package words

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func GetRandomWord(length int) string {
	url := fmt.Sprintf("https://random-word-api.herokuapp.com/word?length=%d", length)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Word API failed, fallback to 'apple'")
		return "apple"
	}
	defer resp.Body.Close()

	var words []string
	err = json.NewDecoder(resp.Body).Decode(&words)
	if err != nil || len(words) == 0 {
		log.Println("Word API returned no words or decode failed, fallback to 'apple'")
		return "apple"
	}

	return words[0]
}
