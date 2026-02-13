package oauth

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html static/*.css
var assetsFS embed.FS

var templates = template.Must(template.ParseFS(assetsFS, "templates/*.html"))
