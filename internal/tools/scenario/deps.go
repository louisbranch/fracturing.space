package scenario

// authProvider creates synthetic users for scenario runs.
type authProvider interface {
	CreateUser(displayName string) string
}

// runnerDeps bundles injectable dependencies for runner construction.
type runnerDeps struct {
	env  scenarioEnv
	auth authProvider
}
