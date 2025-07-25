package utils

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
)

// ----------- ID GENERATOR -----------

// GenerateID creates a random 4-letter lowercase string, e.g., "byzr", "qweh".
// Used for game/session IDs.
func GenerateID() string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, 4)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// ----------- WORD FETCHING (external/random) -----------

// GetWord fetches a random word of the specified length from an external API.
// Returns a fallback (e.g., "aaaaa") if the API fails or is unreachable.
func GetWord(length int) string {
	// Build API URL for requested word length
	url := fmt.Sprintf("https://random-word-api.herokuapp.com/word?length=%d", length)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Word API error:", err)
		// On error, use fallback (e.g., "aaaaa")
		return strings.Repeat("a", length)
	}
	defer resp.Body.Close()

	// Parse returned JSON (should be an array of one word)
	var words []string
	err = json.NewDecoder(resp.Body).Decode(&words)
	if err != nil || len(words) == 0 {
		// Bad API or unexpected response: fallback word
		return strings.Repeat("a", length)
	}

	return words[0]
}

// ----------- TEMPLATE RENDERING: FULL PAGE (with base layout) -----------

// RenderPage renders a full page using base.html + a specific page template.
// - If "User" isn't in data, it sets it from the user cookie (if present).
// - Renders with "base" as the root template.
// Example: RenderPage(w, r, "gameplay.html", data)
func RenderPage(w http.ResponseWriter, r *http.Request, file string, data map[string]interface{}) {
	// Parse both the base layout and the page-specific template
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/"+file))

	// Populate "User" for navigation, if not already in data.
	if _, ok := data["User"]; !ok {
		if c, err := r.Cookie("user"); err == nil {
			data["User"] = c.Value
		}
	}

	// Render the composed page ("base" template) to the HTTP response.
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		log.Println("TEMPLATE EXEC ERROR:", err)
	}
}

// ----------- TEMPLATE RENDERING: PARTIAL/FRAGMENT -----------

// RenderPartial renders a partial template (no site shell, just the inner content block).
// Used for AJAX/HTMX dynamic updates (e.g., only rerender a gameplay panel or polling UI).
// Expects the partial to define "content" block.
func RenderPartial(w http.ResponseWriter, r *http.Request, file string, data map[string]interface{}) {
	tmpl := template.Must(template.ParseFiles("templates/" + file))
	if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		log.Println("TEMPLATE EXEC ERROR (partial):", err)
	}
}
