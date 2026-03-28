package server

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/instructionset"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

func setAISessionGrantEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", "fracturing-space-game")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", "fracturing-space-ai")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", "10m")
}

func TestNewRequiresEncryptionKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", "")
	setAISessionGrantEnv(t)

	if _, err := New(context.Background(), ":0"); err == nil {
		t.Fatal("expected error for missing encryption key")
	}
}

func TestNewRejectsInvalidEncryptionKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", "not-base64")
	setAISessionGrantEnv(t)

	if _, err := New(context.Background(), ":0"); err == nil {
		t.Fatal("expected error for invalid encryption key")
	}
}

func TestNewSuccess(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	srv, err := New(context.Background(), ":0")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	t.Cleanup(func() {
		srv.Close()
	})
	if srv.Addr() == "" {
		t.Fatal("expected non-empty address")
	}
}

func TestNewRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	if _, err := New(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("New error = %v, want context is required", err)
	}
}

func TestRunRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	if err := Run(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("Run error = %v, want context is required", err)
	}
}

func TestServeRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	srv, err := New(context.Background(), ":0")
	if err != nil {
		t.Fatalf("new server for nil-context test: %v", err)
	}
	t.Cleanup(func() {
		srv.Close()
	})
	if err := srv.Serve(nil); err == nil || err.Error() != "context is required" {
		t.Fatalf("Serve error = %v, want context is required", err)
	}
}

func TestServerCloseReleasesListener(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	srv, err := New(context.Background(), ":0")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	addr := srv.Addr()
	if addr == "" {
		t.Fatal("expected non-empty address")
	}

	srv.Close()

	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("listen after close: %v", err)
	}
	_ = l.Close()
}

func TestOpenAIOAuthConfigReturnsNilWhenUnset(t *testing.T) {
	cfg, err := openAIOAuthConfig(serverEnv{})
	if err != nil {
		t.Fatalf("openai oauth config: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config, got %+v", cfg)
	}
}

func TestOpenAIOAuthConfigRejectsPartialConfig(t *testing.T) {
	env := serverEnv{
		OpenAIOAuthAuthURL: "https://provider.example.com/oauth/authorize",
	}
	if _, err := openAIOAuthConfig(env); err == nil {
		t.Fatal("expected error for partial config")
	}
}

func TestOpenAIOAuthConfigBuildsConfigWhenComplete(t *testing.T) {
	env := serverEnv{
		OpenAIOAuthAuthURL:      " https://provider.example.com/oauth/authorize ",
		OpenAIOAuthTokenURL:     " https://provider.example.com/oauth/token ",
		OpenAIOAuthClientID:     " client-1 ",
		OpenAIOAuthClientSecret: " secret-1 ",
		OpenAIOAuthRedirectURI:  " https://app.example.com/oauth/callback ",
	}

	cfg, err := openAIOAuthConfig(env)
	if err != nil {
		t.Fatalf("openai oauth config: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.AuthorizationURL != "https://provider.example.com/oauth/authorize" {
		t.Fatalf("authorization_url = %q", cfg.AuthorizationURL)
	}
	if cfg.TokenURL != "https://provider.example.com/oauth/token" {
		t.Fatalf("token_url = %q", cfg.TokenURL)
	}
	if cfg.ClientID != "client-1" {
		t.Fatalf("client_id = %q", cfg.ClientID)
	}
	if cfg.ClientSecret != "secret-1" {
		t.Fatalf("client_secret = %q", cfg.ClientSecret)
	}
	if cfg.RedirectURI != "https://app.example.com/oauth/callback" {
		t.Fatalf("redirect_uri = %q", cfg.RedirectURI)
	}
}

func TestAISessionGrantConfigFromEnvReturnsNilWhenUnset(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", "")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", "")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", "")
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", "")

	cfg, err := aiSessionGrantConfigFromEnv()
	if err != nil {
		t.Fatalf("ai session grant config from env: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config, got %+v", cfg)
	}
}

func TestAISessionGrantConfigFromEnvBuildsConfigWhenComplete(t *testing.T) {
	setAISessionGrantEnv(t)

	cfg, err := aiSessionGrantConfigFromEnv()
	if err != nil {
		t.Fatalf("ai session grant config from env: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Issuer != "fracturing-space-game" {
		t.Fatalf("issuer = %q", cfg.Issuer)
	}
	if cfg.Audience != "fracturing-space-ai" {
		t.Fatalf("audience = %q", cfg.Audience)
	}
}

func TestBuildPromptBuilderReturnsExplicitBuilderWithoutLoader(t *testing.T) {
	builder := buildPromptBuilder(nil)
	if builder == nil {
		t.Fatal("expected explicit degraded prompt builder")
	}

	prompt, err := builder.Build(context.Background(), promptTestSession{resources: promptTestResources("scene-1")}, orchestration.PromptInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !strings.Contains(prompt, "Daggerheart duality rules") {
		t.Fatalf("prompt missing configured context-source content: %q", prompt)
	}
	if !strings.Contains(prompt, "Daggerheart active character capabilities") {
		t.Fatalf("prompt missing active character capabilities content: %q", prompt)
	}
}

func TestLoadPromptInstructionsAllowsPartialInstructionLoad(t *testing.T) {
	root := t.TempDir()
	writePromptInstructionFile(t, filepath.Join(root, "v1/core/interaction.md"), "# Interaction Contract\nCommit output.")

	instructions := loadPromptInstructions(instructionset.New(root))
	if instructions.Skills != "" {
		t.Fatalf("expected missing skills to degrade to empty, got %q", instructions.Skills)
	}
	if !strings.Contains(instructions.InteractionContract, "Interaction Contract") {
		t.Fatalf("interaction contract = %q", instructions.InteractionContract)
	}
}

func TestBuildRuntimeDepsWithoutGameAddrBuildsDegradedGateway(t *testing.T) {
	logger := newDiscardLogger()
	deps, err := buildRuntimeDeps(context.Background(), testRuntimeConfig(t), logger)
	if err != nil {
		t.Fatalf("buildRuntimeDeps() error = %v", err)
	}
	t.Cleanup(func() {
		deps.close(logger)
	})

	if deps.gameMc != nil {
		t.Fatal("expected nil managed conn when game addr is unset")
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "worker"))
	if deps.gameBridge == nil || !deps.gameBridge.IsAllowedInternalServiceCaller(ctx) {
		t.Fatal("expected degraded gateway to preserve internal-service allowlist")
	}
}

func TestBuildRuntimeDepsWhenGameConnUnavailableStillBuildsGateway(t *testing.T) {
	original := newManagedConn
	newManagedConn = func(context.Context, platformgrpc.ManagedConnConfig) (*platformgrpc.ManagedConn, error) {
		return nil, errors.New("dial failed")
	}
	t.Cleanup(func() {
		newManagedConn = original
	})

	cfg := testRuntimeConfig(t)
	cfg.GameAddr = "127.0.0.1:7777"

	logger := newDiscardLogger()
	deps, err := buildRuntimeDeps(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("buildRuntimeDeps() error = %v", err)
	}
	t.Cleanup(func() {
		deps.close(logger)
	})

	if deps.gameMc != nil {
		t.Fatal("expected nil managed conn when optional game dial fails")
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "worker"))
	if deps.gameBridge == nil || !deps.gameBridge.IsAllowedInternalServiceCaller(ctx) {
		t.Fatal("expected fallback gateway when optional game dial fails")
	}
}

func TestBuildRuntimeDepsRegistersAnthropicWithoutToolRuntime(t *testing.T) {
	logger := newDiscardLogger()
	deps, err := buildRuntimeDeps(context.Background(), testRuntimeConfig(t), logger)
	if err != nil {
		t.Fatalf("buildRuntimeDeps() error = %v", err)
	}
	t.Cleanup(func() {
		deps.close(logger)
	})

	if deps.providerRegistry == nil || !deps.providerRegistry.HasProvider(provider.Anthropic) {
		t.Fatal("expected anthropic provider bundle to be registered")
	}
	if _, ok := deps.providerRegistry.InvocationAdapter(provider.Anthropic); !ok {
		t.Fatal("expected anthropic invocation adapter")
	}
	if _, ok := deps.providerRegistry.ModelAdapter(provider.Anthropic); !ok {
		t.Fatal("expected anthropic model adapter")
	}
	if _, ok := deps.providerRegistry.OAuthAdapter(provider.Anthropic); ok {
		t.Fatal("did not expect anthropic oauth adapter")
	}
	if _, ok := deps.providerRegistry.ToolAdapter(provider.Anthropic); ok {
		t.Fatal("did not expect anthropic tool adapter")
	}
}

func TestRegisterServicesSetsHealthForAllRegistrations(t *testing.T) {
	logger := newDiscardLogger()
	deps, err := buildRuntimeDeps(context.Background(), testRuntimeConfig(t), logger)
	if err != nil {
		t.Fatalf("buildRuntimeDeps() error = %v", err)
	}
	t.Cleanup(func() {
		deps.close(logger)
	})

	handlers, err := buildHandlers(deps)
	if err != nil {
		t.Fatalf("buildHandlers() error = %v", err)
	}

	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	registerServices(grpcServer, healthServer, handlers)

	rootResp, err := healthServer.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("root health check: %v", err)
	}
	if rootResp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("root health status = %v, want %v", rootResp.GetStatus(), grpc_health_v1.HealthCheckResponse_SERVING)
	}

	for _, registration := range serviceRegistrations(handlers) {
		resp, err := healthServer.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: registration.healthName})
		if err != nil {
			t.Fatalf("health check for %s: %v", registration.healthName, err)
		}
		if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Fatalf("health status for %s = %v, want %v", registration.healthName, resp.GetStatus(), grpc_health_v1.HealthCheckResponse_SERVING)
		}
	}
}

type promptTestSession struct {
	resources map[string]string
}

func (s promptTestSession) ListTools(context.Context) ([]orchestration.Tool, error) {
	return nil, nil
}

func (s promptTestSession) CallTool(context.Context, string, any) (orchestration.ToolResult, error) {
	return orchestration.ToolResult{}, nil
}

func (s promptTestSession) ReadResource(_ context.Context, uri string) (string, error) {
	value, ok := s.resources[uri]
	if !ok {
		return "", errors.New("missing resource")
	}
	return value, nil
}

func (s promptTestSession) Close() error { return nil }

func promptTestResources(activeSceneID string) map[string]string {
	return map[string]string{
		"context://current":                                          `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-1"}}`,
		"campaign://camp-1":                                          `{"campaign":{"id":"camp-1","name":"Ashes","theme_prompt":"Ruined empire"}}`,
		"campaign://camp-1/participants":                             `{"participants":[{"id":"gm-1","role":"GM"},{"id":"p-1","role":"PLAYER"}]}`,
		"campaign://camp-1/characters":                               `{"characters":[{"id":"char-1","name":"Theron"}]}`,
		"campaign://camp-1/sessions":                                 `{"sessions":[{"id":"sess-1","status":"ACTIVE"}]}`,
		"campaign://camp-1/sessions/sess-1/scenes":                   `{"scenes":[{"scene_id":"scene-1","character_ids":["char-1"]}]}`,
		"campaign://camp-1/interaction":                              `{"campaign_id":"camp-1","active_session":{"session_id":"sess-1"},"active_scene":{"scene_id":"` + activeSceneID + `"}}`,
		"campaign://camp-1/characters/char-1/sheet":                  `{"character":{"id":"char-1","name":"Theron"},"daggerheart":{"class":{"name":"Guardian"},"resources":{"hope":2},"domain_cards":[{"name":"Shield Wall","domain":"Valor"}]}}`,
		"campaign://camp-1/artifacts/memory.md":                      "",
		"daggerheart://rules/version":                                `{"system":"Daggerheart","module":"duality"}`,
		"daggerheart://campaign/camp-1/sessions/sess-1/combat_board": `{"gm_fear":3,"session_id":"sess-1","scene_id":"` + activeSceneID + `","countdowns":[{"id":"cd-1","name":"Breach","kind":"CONSEQUENCE","current":1,"max":4,"direction":"INCREASE"}]}`,
		"daggerheart://campaign/camp-1/snapshot":                     `{"gm_fear":3,"characters":[]}`,
	}
}

func writePromptInstructionFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func testRuntimeConfig(t *testing.T) runtimeConfig {
	t.Helper()
	return runtimeConfig{
		DBPath:                   filepath.Join(t.TempDir(), "ai.db"),
		EncryptionKey:            base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		InternalServiceAllowlist: map[string]struct{}{"worker": {}},
	}
}

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
