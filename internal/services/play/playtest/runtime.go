// Package playtest provides runtime-backed test helpers for the play service.
package playtest

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	playapp "github.com/louisbranch/fracturing.space/internal/services/play/app"
	playsqlite "github.com/louisbranch/fracturing.space/internal/services/play/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/test/testkit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	gogrpcmetadata "google.golang.org/grpc/metadata"
)

// Runtime exposes the live HTTP surface and launch-grant config for play tests.
type Runtime struct {
	BaseURL           string
	LaunchGrantConfig playlaunchgrant.Config
}

// StartRuntime boots a play runtime against real auth and game dependencies.
func StartRuntime(t *testing.T, authAddr, gameAddr string) Runtime {
	t.Helper()

	authConn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial auth gRPC: %v", err)
	}
	t.Cleanup(func() { _ = authConn.Close() })

	gameConn, err := grpc.NewClient(
		gameAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial game gRPC: %v", err)
	}
	t.Cleanup(func() { _ = gameConn.Close() })

	store, err := playsqlite.Open(filepath.Join(t.TempDir(), "play.db"))
	if err != nil {
		t.Fatalf("open play sqlite store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Fatalf("close play sqlite store: %v", closeErr)
		}
	})

	httpAddr := testkit.PickUnusedAddress(t)
	launchGrantCfg := LaunchGrantConfig(t)
	server, err := playapp.NewServer(playapp.Config{
		HTTPAddr:            httpAddr,
		WebHTTPAddr:         "127.0.0.1:8080",
		PlayUIDevServerURL:  "http://localhost:5173",
		RequestSchemePolicy: requestmeta.SchemePolicy{},
		LaunchGrant:         launchGrantCfg,
	}, playapp.Dependencies{
		Auth:               authv1.NewAuthServiceClient(authConn),
		AIDebug:            &idleAIDebugClient{},
		Interaction:        gamev1.NewInteractionServiceClient(gameConn),
		Campaign:           gamev1.NewCampaignServiceClient(gameConn),
		System:             gamev1.NewSystemServiceClient(gameConn),
		Participants:       gamev1.NewParticipantServiceClient(gameConn),
		Characters:         gamev1.NewCharacterServiceClient(gameConn),
		DaggerheartContent: daggerheartv1.NewDaggerheartContentServiceClient(gameConn),
		Events:             gamev1.NewEventServiceClient(gameConn),
		Transcripts:        store,
	})
	if err != nil {
		t.Fatalf("new play server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe(ctx)
	}()

	baseURL := "http://" + httpAddr
	WaitForHTTPOK(t, baseURL+"/up")

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("play server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for play server to stop")
		}
		server.Close()
	})

	return Runtime{
		BaseURL:           baseURL,
		LaunchGrantConfig: launchGrantCfg,
	}
}

// LaunchGrantConfig returns the standard launch grant config for play tests.
func LaunchGrantConfig(t *testing.T) playlaunchgrant.Config {
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

// WaitForHTTPOK waits until the target URL responds with HTTP 200.
func WaitForHTTPOK(t *testing.T, targetURL string) {
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

// WebsocketURLFromHTTP converts an HTTP base URL into a websocket endpoint.
func WebsocketURLFromHTTP(baseURL string, path string) string {
	return "ws" + strings.TrimPrefix(strings.TrimSpace(baseURL), "http") + path
}

// RequireCookieValue returns the named non-empty cookie value or fails the test.
func RequireCookieValue(t *testing.T, cookies []*http.Cookie, name string) string {
	t.Helper()

	for _, cookie := range cookies {
		if cookie != nil && cookie.Name == name && strings.TrimSpace(cookie.Value) != "" {
			return cookie.Value
		}
	}
	t.Fatalf("missing non-empty cookie %q", name)
	return ""
}

type idleAIDebugClient struct{}

func (c *idleAIDebugClient) ListCampaignDebugTurns(context.Context, *aiv1.ListCampaignDebugTurnsRequest, ...grpc.CallOption) (*aiv1.ListCampaignDebugTurnsResponse, error) {
	return &aiv1.ListCampaignDebugTurnsResponse{}, nil
}

func (c *idleAIDebugClient) GetCampaignDebugTurn(context.Context, *aiv1.GetCampaignDebugTurnRequest, ...grpc.CallOption) (*aiv1.GetCampaignDebugTurnResponse, error) {
	return &aiv1.GetCampaignDebugTurnResponse{}, nil
}

func (c *idleAIDebugClient) SubscribeCampaignDebugUpdates(ctx context.Context, _ *aiv1.SubscribeCampaignDebugUpdatesRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[aiv1.CampaignDebugTurnUpdate], error) {
	return &idleAIDebugStream{ctx: ctx}, nil
}

type idleAIDebugStream struct {
	ctx context.Context
}

func (s *idleAIDebugStream) Recv() (*aiv1.CampaignDebugTurnUpdate, error) {
	if s == nil || s.ctx == nil {
		return nil, io.EOF
	}
	<-s.ctx.Done()
	if errors.Is(s.ctx.Err(), context.Canceled) {
		return nil, io.EOF
	}
	return nil, s.ctx.Err()
}

func (s *idleAIDebugStream) Header() (gogrpcmetadata.MD, error) { return nil, nil }
func (s *idleAIDebugStream) Trailer() gogrpcmetadata.MD         { return nil }
func (s *idleAIDebugStream) CloseSend() error                   { return nil }
func (s *idleAIDebugStream) Context() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}
func (s *idleAIDebugStream) SendMsg(any) error { return nil }
func (s *idleAIDebugStream) RecvMsg(any) error { return nil }
