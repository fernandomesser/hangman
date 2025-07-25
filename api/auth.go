package handlers

import (
	"database/sql"
	"net/http"
	"wordgame/db"
	"wordgame/utils"

	"golang.org/x/crypto/bcrypt"
)

// LoginPage handles GET requests to the login page and renders the login HTML template.
func LoginPage(w http.ResponseWriter, r *http.Request) {
	utils.RenderPage(w, r, "login.html", map[string]interface{}{})
}

// LoginHandler handles POST requests for user login.
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() // Parses form data from the request body

	// Extract submitted username and password
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Query database for the stored password hash for the given username
	var hash string
	err := db.DB.QueryRow("SELECT password_hash FROM users WHERE LOWER(username) = LOWER(?)", username).Scan(&hash)

	// If user not found or password doesn't match the hash, show error and re-render login page
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		utils.RenderPage(w, r, "login.html", map[string]interface{}{
			"Error": "Invalid credentials", // Displayed on the login page
		})
		return
	}

	// If authentication succeeds, set a secure cookie with the username
	http.SetCookie(w, &http.Cookie{
		Name:     "user",   // Cookie name
		Value:    username, // Store username in cookie
		Path:     "/",      // Cookie is valid for all paths
		HttpOnly: true,     // JavaScript cannot access the cookie (helps prevent XSS)
	})

	// Redirect user to the homepage after successful login
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// RegisterPage handles GET requests to the registration page and renders the registration template.
func RegisterPage(w http.ResponseWriter, r *http.Request) {
	utils.RenderPage(w, r, "register.html", map[string]interface{}{})
}

// RegisterHandler handles POST requests for new user registration.
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Check for missing fields
	if username == "" || password == "" {
		utils.RenderPage(w, r, "register.html", map[string]interface{}{
			"Error": "Username and password required.",
		})
		return
	}

	// Check for existing username (case-insensitive)
	var existing string
	err := db.DB.QueryRow(
		"SELECT username FROM users WHERE LOWER(username) = LOWER(?)",
		username,
	).Scan(&existing)

	if err == nil {
		// Username exists
		utils.RenderPage(w, r, "register.html", map[string]interface{}{
			"Error": "Username already exists.",
		})
		return
	}
	if err != sql.ErrNoRows {
		// Unexpected database error
		utils.RenderPage(w, r, "register.html", map[string]interface{}{
			"Error": "Internal server error. Please try again.",
		})
		return
	}

	// Hash password securely
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		utils.RenderPage(w, r, "register.html", map[string]interface{}{
			"Error": "Password error. Please try again.",
		})
		return
	}

	// Insert new user
	_, err = db.DB.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, hash,
	)
	if err != nil {
		utils.RenderPage(w, r, "register.html", map[string]interface{}{
			"Error": "Could not register. Please try again.",
		})
		return
	}

	// Log in user automatically: set login cookie and go to homepage
	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		Value:    username,
		Path:     "/",
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// === LOGOUT HANDLER ===

// LogoutHandler removes the user cookie by setting it with an expired MaxAge,
// effectively logging the user out and redirecting to the homepage.
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "user", // Target the 'user' cookie
		Value:  "",     // Clear its value
		Path:   "/",    // Ensure it matches the original cookie path
		MaxAge: -1,     // Set MaxAge < 0 to delete the cookie immediately
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
