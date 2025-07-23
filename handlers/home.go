package handlers

import (
	"net/http"
	"net/url"
	"wordgame/utils"
)

func WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	user := ""
	if c, err := r.Cookie("user"); err == nil {
		user = c.Value
	}

	data := map[string]interface{}{
		"User": user,
	}

	// Check for an error message
	if errCookie, err := r.Cookie("error"); err == nil {
		if msg, decodeErr := url.QueryUnescape(errCookie.Value); decodeErr == nil {
			data["Error"] = msg
		}
		// Clear the error cookie so it's only shown once
		http.SetCookie(w, &http.Cookie{
			Name:   "error",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
	}

	utils.RenderPage(w, r, "index.html", data)
}
