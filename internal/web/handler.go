package web

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/louisbranch/duality-engine/internal/web/templates"
)

// NewHandler builds the HTTP handler for the web server.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", templ.Handler(templates.Home()))
	return mux
}
