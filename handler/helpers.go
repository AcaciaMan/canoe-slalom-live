package handler

import (
	"fmt"
	"log"
	"net/http"
)

// renderError renders a styled error page with the given HTTP status code and message.
func (d *Deps) renderError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	err := d.Tmpls["error"].ExecuteTemplate(w, "layout.html", map[string]interface{}{
		"Code":    code,
		"Message": message,
		"Title":   fmt.Sprintf("%d — %s", code, message),
	})
	if err != nil {
		log.Printf("Error rendering error page: %v", err)
		http.Error(w, message, code)
	}
}
