package scenario

// authProvider creates a scenario runner user for the target runtime.
type authProvider interface {
	CreateUser(displayName string) (string, error)
}

// runnerDeps bundles injectable dependencies for runner construction.
type runnerDeps struct {
	env  scenarioEnv
	auth authProvider
}
