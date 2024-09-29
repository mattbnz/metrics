package reporting

import (
	"log"
	"net/http"

	"mattb.nz/web/metrics/templates"
)

func Home(w http.ResponseWriter, r *http.Request) {
	page, err := templates.Get("home.html")
	if err != nil {
		log.Printf("Could not load home page template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	page.Execute(w, map[string]any{"Config": conf})
}
