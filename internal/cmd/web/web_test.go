package web

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestParseConfigDefaults(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.HTTPAddr != "localhost:8080" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "localhost:8080")
	}
	if cfg.GameAddr != "game:8082" {
		t.Fatalf("GameAddr = %q, want %q", cfg.GameAddr, "game:8082")
	}
	if cfg.ChatHTTPAddr != "localhost:8086" {
		t.Fatalf("ChatHTTPAddr = %q, want %q", cfg.ChatHTTPAddr, "localhost:8086")
	}
	if cfg.AuthAddr != "auth:8083" {
		t.Fatalf("AuthAddr = %q, want %q", cfg.AuthAddr, "auth:8083")
	}
	if cfg.SocialAddr != "social:8090" {
		t.Fatalf("SocialAddr = %q, want %q", cfg.SocialAddr, "social:8090")
	}
	if cfg.AIAddr != "ai:8087" {
		t.Fatalf("AIAddr = %q, want %q", cfg.AIAddr, "ai:8087")
	}
	if cfg.NotificationsAddr != "notifications:8088" {
		t.Fatalf("NotificationsAddr = %q, want %q", cfg.NotificationsAddr, "notifications:8088")
	}
	if cfg.UserHubAddr != "userhub:8092" {
		t.Fatalf("UserHubAddr = %q, want %q", cfg.UserHubAddr, "userhub:8092")
	}
	if cfg.EnableExperimentalModules {
		t.Fatalf("EnableExperimentalModules = %t, want false", cfg.EnableExperimentalModules)
	}
	if cfg.TrustForwardedProto {
		t.Fatalf("TrustForwardedProto = %t, want false", cfg.TrustForwardedProto)
	}
	if cfg.AssetBaseURL != "" {
		t.Fatalf("AssetBaseURL = %q, want empty", cfg.AssetBaseURL)
	}
}

func TestParseConfigOverrideHTTPAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-http-addr", "127.0.0.1:9002"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9002" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "127.0.0.1:9002")
	}
}

func TestParseConfigOverrideGameAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-game-addr", "127.0.0.1:19082"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.GameAddr != "127.0.0.1:19082" {
		t.Fatalf("GameAddr = %q, want %q", cfg.GameAddr, "127.0.0.1:19082")
	}
}

func TestParseConfigOverrideChatHTTPAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-chat-http-addr", "127.0.0.1:18086"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.ChatHTTPAddr != "127.0.0.1:18086" {
		t.Fatalf("ChatHTTPAddr = %q, want %q", cfg.ChatHTTPAddr, "127.0.0.1:18086")
	}
}

func TestParseConfigOverrideExperimentalModules(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-enable-experimental-modules"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if !cfg.EnableExperimentalModules {
		t.Fatalf("EnableExperimentalModules = %t, want true", cfg.EnableExperimentalModules)
	}
}

func TestParseConfigOverrideTrustForwardedProto(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-trust-forwarded-proto"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if !cfg.TrustForwardedProto {
		t.Fatalf("TrustForwardedProto = %t, want true", cfg.TrustForwardedProto)
	}
}

func TestParseConfigOverrideUserHubAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-userhub-addr", "127.0.0.1:18092"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.UserHubAddr != "127.0.0.1:18092" {
		t.Fatalf("UserHubAddr = %q, want %q", cfg.UserHubAddr, "127.0.0.1:18092")
	}
}

func TestParseConfigOverrideNotificationsAddr(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-notifications-addr", "127.0.0.1:18088"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.NotificationsAddr != "127.0.0.1:18088" {
		t.Fatalf("NotificationsAddr = %q, want %q", cfg.NotificationsAddr, "127.0.0.1:18088")
	}
}

func TestBootstrapDependenciesDialsAllConfiguredServices(t *testing.T) {
	t.Parallel()

	calls := []string{}
	dialer := func(_ context.Context, address string, _ time.Duration) (*grpc.ClientConn, error) {
		calls = append(calls, address)
		return &grpc.ClientConn{}, nil
	}
	bundle, conns, warnings, err := bootstrapDependencies(context.Background(), Config{
		AuthAddr:          "auth:8083",
		SocialAddr:        "social:8090",
		GameAddr:          "game:8082",
		AIAddr:            "ai:8087",
		UserHubAddr:       "userhub:8092",
		NotificationsAddr: "notifications:8088",
		AssetBaseURL:      "https://cdn.example.com/assets",
	}, dialer)
	if err != nil {
		t.Fatalf("bootstrapDependencies() error = %v", err)
	}
	if len(calls) != 6 {
		t.Fatalf("dial calls = %d, want %d", len(calls), 6)
	}
	if len(conns) != 6 {
		t.Fatalf("dependency connections = %d, want %d", len(conns), 6)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if bundle.Principal.SessionClient == nil {
		t.Fatalf("expected principal session client")
	}
	if bundle.Modules.CampaignClient == nil {
		t.Fatalf("expected campaign client")
	}
	if bundle.Modules.CredentialClient == nil {
		t.Fatalf("expected credential client")
	}
	if bundle.Modules.UserHubClient == nil {
		t.Fatalf("expected userhub client")
	}
	if bundle.Modules.NotificationClient == nil {
		t.Fatalf("expected notification client")
	}
	if bundle.Principal.AssetBaseURL != "https://cdn.example.com/assets" {
		t.Fatalf("Principal.AssetBaseURL = %q, want %q", bundle.Principal.AssetBaseURL, "https://cdn.example.com/assets")
	}
	if bundle.Modules.AssetBaseURL != "https://cdn.example.com/assets" {
		t.Fatalf("Modules.AssetBaseURL = %q, want %q", bundle.Modules.AssetBaseURL, "https://cdn.example.com/assets")
	}
}

func TestBootstrapDependenciesOptionalFailuresProduceWarnings(t *testing.T) {
	t.Parallel()

	calls := []string{}
	dialer := func(_ context.Context, address string, _ time.Duration) (*grpc.ClientConn, error) {
		calls = append(calls, address)
		if address == "ai:8087" {
			return nil, errors.New("ai down")
		}
		return &grpc.ClientConn{}, nil
	}
	bundle, conns, warnings, err := bootstrapDependencies(context.Background(), Config{
		AuthAddr:          "auth:8083",
		SocialAddr:        "social:8090",
		GameAddr:          "game:8082",
		AIAddr:            "ai:8087",
		UserHubAddr:       "userhub:8092",
		NotificationsAddr: "notifications:8088",
	}, dialer)
	if err != nil {
		t.Fatalf("bootstrapDependencies() error = %v", err)
	}
	if len(conns) != 5 {
		t.Fatalf("dependency connections = %d, want %d", len(conns), 5)
	}
	if len(calls) != 6 {
		t.Fatalf("dial calls = %d, want %d", len(calls), 6)
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want one warning", warnings)
	}
	if !strings.Contains(warnings[0], "ai") {
		t.Fatalf("warning = %q, want ai warning", warnings[0])
	}
	if bundle.Modules.CredentialClient != nil {
		t.Fatalf("expected credential client to be unavailable")
	}
}

func TestBootstrapDependenciesCollectsMultipleWarnings(t *testing.T) {
	t.Parallel()

	calls := []string{}
	failures := map[string]error{
		"ai:8087":      errors.New("ai down"),
		"userhub:8092": errors.New("userhub down"),
	}
	dialer := func(_ context.Context, address string, _ time.Duration) (*grpc.ClientConn, error) {
		calls = append(calls, address)
		if err, ok := failures[address]; ok {
			return nil, err
		}
		return &grpc.ClientConn{}, nil
	}
	bundle, conns, warnings, err := bootstrapDependencies(context.Background(), Config{
		AuthAddr:          "auth:8083",
		SocialAddr:        "social:8090",
		GameAddr:          "game:8082",
		AIAddr:            "ai:8087",
		UserHubAddr:       "userhub:8092",
		NotificationsAddr: "notifications:8088",
	}, dialer)
	if err != nil {
		t.Fatalf("bootstrapDependencies() error = %v", err)
	}
	if len(calls) != 6 {
		t.Fatalf("dial calls = %d, want %d", len(calls), 6)
	}
	if len(conns) != 4 {
		t.Fatalf("dependency connections = %d, want %d", len(conns), 4)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v, want two warnings", warnings)
	}
	var (
		hasAI      bool
		hasUserHub bool
	)
	for _, warning := range warnings {
		if strings.Contains(warning, "ai") {
			hasAI = true
		}
		if strings.Contains(warning, "userhub") {
			hasUserHub = true
		}
	}
	if !hasAI || !hasUserHub {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if bundle.Modules.CredentialClient != nil {
		t.Fatalf("expected credential client to be unavailable")
	}
	if bundle.Modules.UserHubClient != nil {
		t.Fatalf("expected userhub client to be unavailable")
	}
}
