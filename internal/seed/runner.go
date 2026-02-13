package seed

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

// Run executes seed scenarios against the MCP server.
func Run(ctx context.Context, cfg Config) error {
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

	client, err := StartMCPClient(ctx, cfg.RepoRoot, cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("start MCP client: %w", err)
	}
	defer client.Close()

	authAddr := strings.TrimSpace(cfg.AuthAddr)
	if authAddr == "" {
		return fmt.Errorf("auth server address is required")
	}
	userID, err := createSeedUser(ctx, authAddr)
	if err != nil {
		return err
	}

	for _, fixture := range fixtures {
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "Running scenario: %s\n", fixture.Name)
		}
		if err := runFixture(ctx, client, fixture, cfg.Verbose, userID); err != nil {
			return fmt.Errorf("scenario %q: %w", fixture.Name, err)
		}
	}

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Seeding complete\n")
	}
	return nil
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

func runFixture(ctx context.Context, client *StdioClient, fixture BlackboxFixture, verbose bool, userID string) error {
	captures := make(map[string]string)
	for _, step := range fixture.Steps {
		if err := executeStep(ctx, client, step, captures, verbose, userID); err != nil {
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
	}
	return nil
}

func executeStep(ctx context.Context, client *StdioClient, step BlackboxStep, captures map[string]string, verbose bool, userID string) error {
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

	responseAny, responseBytes, err := client.ReadResponseForID(requestID, 30*time.Second)
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
		ctx = context.Background()
	}
	conn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return "", fmt.Errorf("connect auth server: %w", err)
	}
	defer conn.Close()
	client := authv1.NewAuthServiceClient(conn)
	resp, err := client.CreateUser(ctx, &authv1.CreateUserRequest{DisplayName: "Seed Creator"})
	if err != nil {
		return "", fmt.Errorf("create seed user: %w", err)
	}
	userID := resp.GetUser().GetId()
	if userID == "" {
		return "", fmt.Errorf("create seed user: missing user id in response")
	}
	return userID, nil
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
