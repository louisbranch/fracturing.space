//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	webserver "github.com/louisbranch/fracturing.space/internal/services/web"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type webRuntime struct {
	baseURL string
}

type webRuntimeConfig struct {
	playHTTPAddr      string
	authAddr          string
	socialAddr        string
	gameAddr          string
	inviteAddr        string
	notificationsAddr string
	userhubAddr       string
	statusAddr        string
}

func startWebRuntime(t *testing.T, cfg webRuntimeConfig) webRuntime {
	t.Helper()

	if strings.TrimSpace(cfg.playHTTPAddr) == "" {
		t.Fatal("play http address is required")
	}
	if strings.TrimSpace(cfg.authAddr) == "" {
		t.Fatal("auth address is required")
	}
	if strings.TrimSpace(cfg.socialAddr) == "" {
		t.Fatal("social address is required")
	}
	if strings.TrimSpace(cfg.gameAddr) == "" {
		t.Fatal("game address is required")
	}
	if strings.TrimSpace(cfg.inviteAddr) == "" {
		t.Fatal("invite address is required")
	}

	bundle := webserver.NewDependencyBundle("")
	bindWebDependency(t, &bundle, cfg.authAddr, webserver.BindAuthDependency)
	bindWebDependency(t, &bundle, cfg.socialAddr, webserver.BindSocialDependency)
	bindWebDependency(t, &bundle, cfg.gameAddr, webserver.BindGameDependency)
	bindWebDependency(t, &bundle, cfg.inviteAddr, webserver.BindInviteDependency)
	bundle.Modules.Campaigns.DiscoveryClient = inertDiscoveryClient{}
	bundle.Modules.Campaigns.AgentClient = inertCampaignAgentClient{}
	bundle.Modules.Campaigns.CampaignArtifactClient = inertCampaignArtifactClient{}

	if addr := strings.TrimSpace(cfg.notificationsAddr); addr != "" {
		bindWebDependency(t, &bundle, addr, webserver.BindNotificationsDependency)
	}
	if addr := strings.TrimSpace(cfg.userhubAddr); addr != "" {
		bindWebDependency(t, &bundle, addr, webserver.BindUserHubDependency)
	}
	if addr := strings.TrimSpace(cfg.statusAddr); addr != "" {
		bindWebDependency(t, &bundle, addr, webserver.BindStatusDependency)
	}

	httpAddr := pickUnusedAddress(t)
	server, err := webserver.NewServer(context.Background(), webserver.Config{
		HTTPAddr:        httpAddr,
		PlayHTTPAddr:    cfg.playHTTPAddr,
		PlayLaunchGrant: testPlayLaunchGrantConfig(t),
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

	return webRuntime{baseURL: baseURL}
}

func bindWebDependency(
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

type inertDiscoveryClient struct{}

func (inertDiscoveryClient) GetDiscoveryEntry(context.Context, *discoveryv1.GetDiscoveryEntryRequest, ...grpc.CallOption) (*discoveryv1.GetDiscoveryEntryResponse, error) {
	return &discoveryv1.GetDiscoveryEntryResponse{}, nil
}

type inertCampaignAgentClient struct{}

func (inertCampaignAgentClient) ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error) {
	return &aiv1.ListAgentsResponse{}, nil
}

type inertCampaignArtifactClient struct{}

func (inertCampaignArtifactClient) EnsureCampaignArtifacts(context.Context, *aiv1.EnsureCampaignArtifactsRequest, ...grpc.CallOption) (*aiv1.EnsureCampaignArtifactsResponse, error) {
	return &aiv1.EnsureCampaignArtifactsResponse{}, nil
}
