package handlers

import (
	"net/http"
	"net/url"
	"wordgame/utils"
)

// WelcomeHandler displays the welcome page, with user greeting and any error messages.
func WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	user := "" // Default: empty user

	// Try to read the "user" cookie, which identifies logged-in users.
	if c, err := r.Cookie("user"); err == nil {
		user = c.Value // If present, get username
	}

	// Prepare data for the template, always including user (may be empty).
	data := map[string]interface{}{
		"User": user,
	}

	// See if an "error" cookie is set (usually after a redirect), and pass it to the template.
	if errCookie, err := r.Cookie("error"); err == nil {
		if msg, decodeErr := url.QueryUnescape(errCookie.Value); decodeErr == nil {
			data["Error"] = msg // Pass decoded error message to template if possible
		}
		// Expire/clear the error cookie so errors aren't persistent or shown multiple times.
		http.SetCookie(w, &http.Cookie{
			Name:   "error",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
	}

	// Render the main page ("index.html"), giving user info and error (if any).
	utils.RenderPage(w, r, "index.html", data)
}
