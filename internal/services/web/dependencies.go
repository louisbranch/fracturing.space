package web

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

// DependencyBundle is a single source of startup dependencies used by web service
// composition.
type DependencyBundle struct {
	// Principal carries the clients required for request-scoped principal resolution.
	Principal principal.Dependencies
	// Modules carries feature module dependencies and shared runtime config.
	Modules modules.Dependencies
}
