package handlers

import (
	"net/http"
)

func WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	user := ""
	if c, err := r.Cookie("user"); err == nil {
		user = c.Value
	}

	renderPage(w, r, "index.html", map[string]interface{}{
		"User": user,
	})
}
