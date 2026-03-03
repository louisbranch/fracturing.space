package composition

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/admin/app"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules"
)

// Registry builds admin module sets from composition input.
type Registry interface {
	Build(modules.BuildInput) modules.BuildOutput
}

// ComposeInput describes contracts needed to compose the admin handler.
type ComposeInput struct {
	Modules  modules.BuildInput
	Registry Registry
}

// ComposeAppHandler builds the admin app handler with selected module sets.
func ComposeAppHandler(input ComposeInput) (http.Handler, error) {
	registry := input.Registry
	if registry == nil {
		defaultRegistry := modules.NewRegistry()
		registry = defaultRegistry
	}
	built := registry.Build(input.Modules)
	return app.Compose(app.ComposeInput{Modules: built.Modules})
}
