package server

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/instructionset"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
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

	if _, err := New(0); err == nil {
		t.Fatal("expected error for missing encryption key")
	}
}

func TestNewRejectsInvalidEncryptionKey(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", "not-base64")
	setAISessionGrantEnv(t)

	if _, err := New(0); err == nil {
		t.Fatal("expected error for invalid encryption key")
	}
}

func TestNewSuccess(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	srv, err := New(0)
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

func TestNewWithAddrContextRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	if _, err := NewWithAddrContext(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("NewWithAddrContext error = %v, want context is required", err)
	}
}

func TestRunWithAddrRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	if err := RunWithAddr(nil, "127.0.0.1:0"); err == nil || err.Error() != "context is required" {
		t.Fatalf("RunWithAddr error = %v, want context is required", err)
	}
}

func TestServeRequiresContext(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_DB_PATH", filepath.Join(t.TempDir(), "ai.db"))
	t.Setenv("FRACTURING_SPACE_AI_ENCRYPTION_KEY", base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	setAISessionGrantEnv(t)

	srv, err := New(0)
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

	srv, err := New(0)
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

func TestOpenAIOAuthConfigFromEnvReturnsNilWhenUnset(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI", "")

	cfg, err := openAIOAuthConfigFromEnv()
	if err != nil {
		t.Fatalf("openai oauth config from env: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config, got %+v", cfg)
	}
}

func TestOpenAIOAuthConfigFromEnvRejectsPartialConfig(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL", "https://provider.example.com/oauth/authorize")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET", "")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI", "")

	if _, err := openAIOAuthConfigFromEnv(); err == nil {
		t.Fatal("expected error for partial config")
	}
}

func TestOpenAIOAuthConfigFromEnvBuildsConfigWhenComplete(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_AUTH_URL", " https://provider.example.com/oauth/authorize ")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_TOKEN_URL", " https://provider.example.com/oauth/token ")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_ID", " client-1 ")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_CLIENT_SECRET", " secret-1 ")
	t.Setenv("FRACTURING_SPACE_AI_OPENAI_OAUTH_REDIRECT_URI", " https://app.example.com/oauth/callback ")

	cfg, err := openAIOAuthConfigFromEnv()
	if err != nil {
		t.Fatalf("openai oauth config from env: %v", err)
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
		"context://current":                        `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-1"}}`,
		"campaign://camp-1":                        `{"campaign":{"id":"camp-1","name":"Ashes","theme_prompt":"Ruined empire"}}`,
		"campaign://camp-1/participants":           `{"participants":[{"id":"gm-1","role":"GM"},{"id":"p-1","role":"PLAYER"}]}`,
		"campaign://camp-1/characters":             `{"characters":[{"id":"char-1","name":"Theron"}]}`,
		"campaign://camp-1/sessions":               `{"sessions":[{"id":"sess-1","status":"ACTIVE"}]}`,
		"campaign://camp-1/sessions/sess-1/scenes": `{"scenes":[{"scene_id":"scene-1"}]}`,
		"campaign://camp-1/interaction":            `{"campaign_id":"camp-1","active_session":{"session_id":"sess-1"},"active_scene":{"scene_id":"` + activeSceneID + `"}}`,
		"campaign://camp-1/artifacts/memory.md":    "",
		"daggerheart://rules/version":              `{"system":"Daggerheart","module":"duality"}`,
		"daggerheart://campaign/camp-1/snapshot":   `{"gm_fear":3,"characters":[]}`,
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
