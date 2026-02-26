package app

import module "github.com/louisbranch/fracturing.space/internal/services/web/module"

// Config captures the composition inputs for the web root handler.
type Config struct {
	Dependencies     module.Dependencies
	PublicModules    []module.Module
	ProtectedModules []module.Module
}
