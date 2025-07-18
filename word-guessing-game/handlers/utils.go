package handlers

import (
	"html/template"
	"log"
	"net/http"
)

func renderPage(w http.ResponseWriter, r *http.Request, file string, data map[string]interface{}) {
	tmpl := template.Must(template.ParseFiles("templates/base.html", "templates/"+file))

	if _, ok := data["User"]; !ok {
		if c, err := r.Cookie("user"); err == nil {
			data["User"] = c.Value
		}
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		log.Println("TEMPLATE EXEC ERROR:", err)
	}
}
