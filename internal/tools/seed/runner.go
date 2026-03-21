package seed

import (
	"fmt"
)

// Config holds seed runner configuration.
type Config struct {
	RepoRoot    string
	GRPCAddr    string
	AuthAddr    string
	Scenario    string
	Verbose     bool
	FixturesDir string
}

// DefaultConfig returns configuration with common defaults.
func DefaultConfig() Config {
	return Config{
		GRPCAddr:    "localhost:8080",
		AuthAddr:    "localhost:8083",
		FixturesDir: "internal/test/integration/fixtures/seed",
	}
}

// ListScenarios returns available scenario names.
func ListScenarios(cfg Config) ([]string, error) {
	fixturesPath := fmt.Sprintf("%s/%s/*.json", cfg.RepoRoot, cfg.FixturesDir)
	fixtures, err := LoadFixtures(fixturesPath)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(fixtures))
	for i, f := range fixtures {
		names[i] = f.Name
	}
	return names, nil
}
