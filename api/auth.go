package handlers

import (
	"net/http"
	"wordgame/db"
	"wordgame/utils"

	"golang.org/x/crypto/bcrypt"
)

// === LOGIN ===
func LoginPage(w http.ResponseWriter, r *http.Request) {
	utils.RenderPage(w, r, "login.html", map[string]interface{}{})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Check user in DB
	var hash string
	err := db.DB.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		utils.RenderPage(w, r, "login.html", map[string]interface{}{
			"Error": "Invalid credentials",
		})
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		Value:    username,
		Path:     "/",
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// === REGISTER ===
func RegisterPage(w http.ResponseWriter, r *http.Request) {
	utils.RenderPage(w, r, "register.html", map[string]interface{}{})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		utils.RenderPage(w, r, "login.html", map[string]interface{}{
			"Error": "Invalid credentials",
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		utils.RenderPage(w, r, "login.html", map[string]interface{}{
			"Error": "Invalid credentials",
		})
		return
	}

	_, err = db.DB.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hash)
	if err != nil {
		utils.RenderPage(w, r, "login.html", map[string]interface{}{
			"Error": "Invalid credentials",
		})
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// === LOGOUT ===
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "user",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
