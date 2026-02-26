package app

import module "github.com/louisbranch/fracturing.space/internal/services/web2/module"

// Config captures the composition inputs for the web2 root handler.
type Config struct {
	Dependencies     module.Dependencies
	PublicModules    []module.Module
	ProtectedModules []module.Module
}
