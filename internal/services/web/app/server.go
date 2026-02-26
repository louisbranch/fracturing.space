package app

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
)

// BuildRootHandler composes a root mux using the configured module groups.
func BuildRootHandler(cfg Config, authRequired func(*http.Request) bool) (http.Handler, error) {
	composer := Composer{}
	deps := cfg.Dependencies
	// TODO(web-composition): consider fail-fast startup checks for required resolver seams; implicit defaults can hide missing wiring.
	if deps.ResolveLanguage == nil {
		deps.ResolveLanguage = i18n.ResolveLanguage
	}
	if deps.ResolveViewer == nil {
		deps.ResolveViewer = func(*http.Request) module.Viewer { return module.Viewer{} }
	}
	return composer.Compose(ComposeInput{
		Dependencies:     deps,
		AuthRequired:     authRequired,
		PublicModules:    cfg.PublicModules,
		ProtectedModules: cfg.ProtectedModules,
	})
}
