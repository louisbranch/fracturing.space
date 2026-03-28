// Package webtest provides runtime-backed test helpers for the web service.
package webtest

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	webserver "github.com/louisbranch/fracturing.space/internal/services/web"
	"github.com/louisbranch/fracturing.space/internal/test/testkit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Runtime exposes the live HTTP surface for web tests.
type Runtime struct {
	BaseURL string
}

// RuntimeConfig controls web runtime startup for integration-style tests.
type RuntimeConfig struct {
	PlayHTTPAddr      string
	AuthAddr          string
	SocialAddr        string
	GameAddr          string
	InviteAddr        string
	NotificationsAddr string
	UserHubAddr       string
	StatusAddr        string
}

// StartRuntime boots a web runtime against real downstream dependencies.
func StartRuntime(t *testing.T, cfg RuntimeConfig) Runtime {
	t.Helper()

	if strings.TrimSpace(cfg.PlayHTTPAddr) == "" {
		t.Fatal("play http address is required")
	}
	if strings.TrimSpace(cfg.AuthAddr) == "" {
		t.Fatal("auth address is required")
	}
	if strings.TrimSpace(cfg.SocialAddr) == "" {
		t.Fatal("social address is required")
	}
	if strings.TrimSpace(cfg.GameAddr) == "" {
		t.Fatal("game address is required")
	}
	if strings.TrimSpace(cfg.InviteAddr) == "" {
		t.Fatal("invite address is required")
	}

	bundle := webserver.NewDependencyBundle("")
	bindDependency(t, &bundle, cfg.AuthAddr, webserver.BindAuthDependency)
	bindDependency(t, &bundle, cfg.SocialAddr, webserver.BindSocialDependency)
	bindDependency(t, &bundle, cfg.GameAddr, webserver.BindGameDependency)
	bindDependency(t, &bundle, cfg.InviteAddr, webserver.BindInviteDependency)
	bundle.Modules.Campaigns.DiscoveryClient = inertDiscoveryClient{}
	bundle.Modules.Campaigns.AgentClient = inertCampaignAgentClient{}
	bundle.Modules.Campaigns.CampaignArtifactClient = inertCampaignArtifactClient{}

	if addr := strings.TrimSpace(cfg.NotificationsAddr); addr != "" {
		bindDependency(t, &bundle, addr, webserver.BindNotificationsDependency)
	}
	if addr := strings.TrimSpace(cfg.UserHubAddr); addr != "" {
		bindDependency(t, &bundle, addr, webserver.BindUserHubDependency)
	}
	if addr := strings.TrimSpace(cfg.StatusAddr); addr != "" {
		bindDependency(t, &bundle, addr, webserver.BindStatusDependency)
	}

	httpAddr := testkit.PickUnusedAddress(t)
	server, err := webserver.NewServer(context.Background(), webserver.Config{
		HTTPAddr:        httpAddr,
		PlayHTTPAddr:    cfg.PlayHTTPAddr,
		PlayLaunchGrant: launchGrantConfig(t),
		Dependencies:    &bundle,
	})
	if err != nil {
		t.Fatalf("new web server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe(ctx)
	}()

	baseURL := "http://" + httpAddr
	waitForHTTPOK(t, baseURL+"/up")

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("web server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for web server to stop")
		}
		server.Close()
	})

	return Runtime{BaseURL: baseURL}
}

// bindDependency attaches one real downstream gRPC connection to the web bundle.
func bindDependency(
	t *testing.T,
	bundle *webserver.DependencyBundle,
	addr string,
	bind func(*webserver.DependencyBundle, *grpc.ClientConn),
) {
	t.Helper()

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial web dependency %q: %v", addr, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	bind(bundle, conn)
}

// launchGrantConfig returns a deterministic launch-grant signer for tests.
func launchGrantConfig(t *testing.T) playlaunchgrant.Config {
	t.Helper()
	key, err := base64.RawStdEncoding.DecodeString("MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY")
	if err != nil {
		t.Fatalf("decode play launch grant key: %v", err)
	}
	return playlaunchgrant.Config{
		Issuer:   "fracturing-space-web",
		Audience: "fracturing-space-play",
		HMACKey:  key,
		TTL:      2 * time.Minute,
		Now:      time.Now,
	}
}

// inertDiscoveryClient satisfies optional campaign discovery wiring in tests.
type inertDiscoveryClient struct{}

// GetDiscoveryEntry returns an empty response so tests can ignore discovery data.
func (inertDiscoveryClient) GetDiscoveryEntry(context.Context, *discoveryv1.GetDiscoveryEntryRequest, ...grpc.CallOption) (*discoveryv1.GetDiscoveryEntryResponse, error) {
	return &discoveryv1.GetDiscoveryEntryResponse{}, nil
}

// inertCampaignAgentClient disables AI agent listing for web runtime tests.
type inertCampaignAgentClient struct{}

// ListAgents returns an empty page so web tests can skip AI agent setup.
func (inertCampaignAgentClient) ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error) {
	return &aiv1.ListAgentsResponse{}, nil
}

// inertCampaignArtifactClient disables artifact provisioning for web tests.
type inertCampaignArtifactClient struct{}

// EnsureCampaignArtifacts returns an empty response without side effects.
func (inertCampaignArtifactClient) EnsureCampaignArtifacts(context.Context, *aiv1.EnsureCampaignArtifactsRequest, ...grpc.CallOption) (*aiv1.EnsureCampaignArtifactsResponse, error) {
	return &aiv1.EnsureCampaignArtifactsResponse{}, nil
}

// waitForHTTPOK blocks until the test HTTP endpoint returns 200 or times out.
func waitForHTTPOK(t *testing.T, targetURL string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(targetURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for http 200 at %s", targetURL)
}
