// Package modulehandlertest provides test fixtures for module handler bases.
// Import this package only in test files.
package modulehandlertest

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
)

// NewBase builds a handler base with no-op resolvers suitable for tests
// that do not exercise user resolution, localization, or viewer state.
func NewBase() modulehandler.Base {
	return modulehandler.NewBase(
		func(*http.Request) string { return "" },
		func(*http.Request) string { return "" },
		func(*http.Request) module.Viewer { return module.Viewer{} },
	)
}
