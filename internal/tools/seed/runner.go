package seed

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type runDeps struct {
	startMCPClient func(ctx context.Context, repoRoot, grpcAddr string) (mcpClient, error)
	createSeedUser func(ctx context.Context, authAddr string) (string, error)
}

func defaultRunDeps() runDeps {
	return runDeps{
		startMCPClient: func(ctx context.Context, repoRoot, grpcAddr string) (mcpClient, error) {
			return StartMCPClient(ctx, repoRoot, grpcAddr)
		},
		createSeedUser: createSeedUser,
	}
}

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

// Run executes seed scenarios against the MCP server.
func Run(ctx context.Context, cfg Config) error {
	return runWithDeps(ctx, cfg, defaultRunDeps())
}

func runWithDeps(ctx context.Context, cfg Config, deps runDeps) error {
	if deps.startMCPClient == nil {
		return fmt.Errorf("MCP client starter is required")
	}
	if deps.createSeedUser == nil {
		return fmt.Errorf("seed user creator is required")
	}

	fixturesPath := filepath.Join(cfg.RepoRoot, cfg.FixturesDir, "*.json")
	if cfg.Scenario != "" {
		fixturesPath = filepath.Join(cfg.RepoRoot, cfg.FixturesDir, cfg.Scenario+".json")
	}

	fixtures, err := LoadFixtures(fixturesPath)
	if err != nil {
		return fmt.Errorf("load fixtures: %w", err)
	}

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Loaded %d fixture(s)\n", len(fixtures))
	}

	client, err := deps.startMCPClient(ctx, cfg.RepoRoot, cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("start MCP client: %w", err)
	}
	defer client.Close()

	authAddr := strings.TrimSpace(cfg.AuthAddr)
	if authAddr == "" {
		return fmt.Errorf("auth server address is required")
	}
	userID, err := deps.createSeedUser(ctx, authAddr)
	if err != nil {
		return err
	}

	return runFixtures(ctx, client, fixtures, cfg.Verbose, userID)
}

// ListScenarios returns available scenario names.
func ListScenarios(cfg Config) ([]string, error) {
	fixturesPath := filepath.Join(cfg.RepoRoot, cfg.FixturesDir, "*.json")
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

// runFixtures iterates over fixtures and runs each one. Extracted from Run
// so tests can inject a fake mcpClient.
func runFixtures(ctx context.Context, client mcpClient, fixtures []BlackboxFixture, verbose bool, userID string) error {
	for _, fixture := range fixtures {
		if verbose {
			fmt.Fprintf(os.Stderr, "Running scenario: %s\n", fixture.Name)
		}
		if err := runFixture(ctx, client, fixture, verbose, userID); err != nil {
			return fmt.Errorf("scenario %q: %w", fixture.Name, err)
		}
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Seeding complete\n")
	}
	return nil
}

func runFixture(ctx context.Context, client mcpClient, fixture BlackboxFixture, verbose bool, userID string) error {
	captures := make(map[string]string)
	for _, step := range fixture.Steps {
		if err := executeStep(ctx, client, step, captures, verbose, userID); err != nil {
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
	}
	return nil
}

func executeStep(ctx context.Context, client mcpClient, step BlackboxStep, captures map[string]string, verbose bool, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "  → %s\n", step.Name)
	}

	request, err := RenderPlaceholders(step.Request, captures)
	if err != nil {
		return fmt.Errorf("render placeholders: %w", err)
	}
	requestMap, ok := request.(map[string]any)
	if !ok {
		return fmt.Errorf("request is not an object")
	}
	if userID != "" {
		injectCampaignCreatorUserID(requestMap, userID)
	}
	requestID, hasID := requestMap["id"]

	if err := client.WriteMessage(request); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	if !hasID {
		return nil
	}

	responseAny, responseBytes, err := client.ReadResponseForID(ctx, requestID, 30*time.Second)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if responseAny == nil {
		return fmt.Errorf("response is nil")
	}

	// Check for JSON-RPC error
	if errDetails := FormatJSONRPCError(responseAny); errDetails != "" {
		return fmt.Errorf("JSON-RPC error: %s", errDetails)
	}

	// Validate expected paths (for seed, we mainly care about captures succeeding)
	for path, expected := range step.ExpectPaths {
		actual, err := LookupJSONPath(responseAny, path)
		if err != nil {
			return fmt.Errorf("lookup %s: %w (response=%s)", path, err, string(responseBytes))
		}
		resolvedExpected, err := RenderPlaceholders(expected, captures)
		if err != nil {
			return fmt.Errorf("render expected: %w", err)
		}
		if !ValuesEqual(actual, resolvedExpected) {
			return fmt.Errorf("expected %s = %v, got %v", path, resolvedExpected, actual)
		}
	}

	// Validate expect_contains
	for path, expected := range step.ExpectContains {
		actual, err := LookupJSONPath(responseAny, path)
		if err != nil {
			return fmt.Errorf("lookup %s: %w (response=%s)", path, err, string(responseBytes))
		}
		resolvedExpected, err := RenderPlaceholders(expected, captures)
		if err != nil {
			return fmt.Errorf("render expected: %w", err)
		}
		if err := AssertArrayContains(actual, resolvedExpected); err != nil {
			return fmt.Errorf("expected %s to contain %v: %w", path, resolvedExpected, err)
		}
	}

	// Process captures
	for key, paths := range step.Captures {
		value, err := CaptureFromPaths(responseAny, paths)
		if err != nil {
			hints := CaptureHints(responseAny)
			if len(hints) > 0 {
				return fmt.Errorf("capture %s: %w (hints=%s, response=%s)", key, err, FormatCaptureHints(hints), string(responseBytes))
			}
			return fmt.Errorf("capture %s: %w (response=%s)", key, err, string(responseBytes))
		}
		if value == "" {
			return fmt.Errorf("capture %s: empty value", key)
		}
		captures[key] = value
		if verbose {
			fmt.Fprintf(os.Stderr, "    captured %s=%s\n", key, value)
		}
	}

	return nil
}

func createSeedUser(ctx context.Context, authAddr string) (string, error) {
	if ctx == nil {
		return "", errors.New("context is nil")
	}
	if strings.TrimSpace(authAddr) == "" {
		return "", fmt.Errorf("auth server address is required")
	}
	// The legacy fixture runner is no longer responsible for provisioning auth
	// accounts. It only needs a stable creator identity for MCP campaign calls.
	return "seed-runner-user", nil
}

func injectCampaignCreatorUserID(request map[string]any, userID string) {
	if request == nil {
		return
	}
	method, _ := request["method"].(string)
	if method != "tools/call" {
		return
	}
	params, ok := request["params"].(map[string]any)
	if !ok {
		return
	}
	toolName, _ := params["name"].(string)
	if toolName != "campaign_create" {
		return
	}
	arguments, ok := params["arguments"].(map[string]any)
	if !ok {
		return
	}
	if _, exists := arguments["user_id"]; !exists {
		arguments["user_id"] = userID
	}
}
