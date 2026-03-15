package principal

import (
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
)

// ViewerFunc resolves app-chrome viewer state for one request.
type ViewerFunc func(*http.Request) module.Viewer

// SignedInFunc reports whether the request is associated with a signed-in actor.
type SignedInFunc func(*http.Request) bool

// UserIDFunc resolves the authenticated user id for one request.
type UserIDFunc func(*http.Request) string

// LanguageFunc returns the effective request language.
type LanguageFunc func(*http.Request) string
