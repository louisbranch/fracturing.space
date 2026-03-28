//go:build integration

package integration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	playsqlite "github.com/louisbranch/fracturing.space/internal/services/play/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"golang.org/x/net/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	gogrpcmetadata "google.golang.org/grpc/metadata"
)

type playRuntime struct {
	baseURL           string
	launchGrantConfig playlaunchgrant.Config
}

type playFrame struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type playSendFrame struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

func startPlayRuntime(t *testing.T, authAddr, gameAddr string) playRuntime {
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

	gameConn := dialGRPCWithServiceID(t, gameAddr, "play")
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

	httpAddr := pickUnusedAddress(t)
	launchGrantCfg := testPlayLaunchGrantConfig(t)
	server, err := playapp.NewServer(playapp.Config{
		HTTPAddr:            httpAddr,
		WebHTTPAddr:         "127.0.0.1:8080",
		PlayUIDevServerURL:  "http://localhost:5173",
		RequestSchemePolicy: requestmeta.SchemePolicy{},
		LaunchGrant:         launchGrantCfg,
	}, playapp.Dependencies{
		Auth:               authv1.NewAuthServiceClient(authConn),
		AIDebug:            &idlePlayAIDebugClient{},
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
	waitForHTTPOK(t, baseURL+"/up")

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

	return playRuntime{
		baseURL:           baseURL,
		launchGrantConfig: launchGrantCfg,
	}
}

func testPlayLaunchGrantConfig(t *testing.T) playlaunchgrant.Config {
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

func readPlayFrame(t *testing.T, conn *websocket.Conn) playFrame {
	t.Helper()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set websocket deadline: %v", err)
	}
	var frame playFrame
	if err := websocket.JSON.Receive(conn, &frame); err != nil {
		t.Fatalf("read websocket frame: %v", err)
	}
	return frame
}

func waitForPlayFrame(t *testing.T, conn *websocket.Conn, wantType string) playFrame {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		frame := readPlayFrame(t, conn)
		if strings.TrimSpace(frame.Type) == wantType {
			return frame
		}
	}
	t.Fatalf("timed out waiting for websocket frame type %q", wantType)
	return playFrame{}
}

func waitForPlayInteractionUpdate(
	t *testing.T,
	conn *websocket.Conn,
	match func(playprotocol.RoomSnapshot) bool,
) playprotocol.RoomSnapshot {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		frame := readPlayFrame(t, conn)
		if strings.TrimSpace(frame.Type) != "play.interaction.updated" {
			continue
		}
		snapshot := decodePlayPayload[playprotocol.RoomSnapshot](t, frame.Payload)
		if match(snapshot) {
			return snapshot
		}
	}
	t.Fatal("timed out waiting for matching play interaction update")
	return playprotocol.RoomSnapshot{}
}

type idlePlayAIDebugClient struct{}

func (c *idlePlayAIDebugClient) ListCampaignDebugTurns(context.Context, *aiv1.ListCampaignDebugTurnsRequest, ...grpc.CallOption) (*aiv1.ListCampaignDebugTurnsResponse, error) {
	return &aiv1.ListCampaignDebugTurnsResponse{}, nil
}

func (c *idlePlayAIDebugClient) GetCampaignDebugTurn(context.Context, *aiv1.GetCampaignDebugTurnRequest, ...grpc.CallOption) (*aiv1.GetCampaignDebugTurnResponse, error) {
	return &aiv1.GetCampaignDebugTurnResponse{}, nil
}

func (c *idlePlayAIDebugClient) SubscribeCampaignDebugUpdates(ctx context.Context, _ *aiv1.SubscribeCampaignDebugUpdatesRequest, _ ...grpc.CallOption) (grpc.ServerStreamingClient[aiv1.CampaignDebugTurnUpdate], error) {
	return &idlePlayAIDebugStream{ctx: ctx}, nil
}

type idlePlayAIDebugStream struct {
	ctx context.Context
}

func (s *idlePlayAIDebugStream) Recv() (*aiv1.CampaignDebugTurnUpdate, error) {
	if s == nil || s.ctx == nil {
		return nil, io.EOF
	}
	<-s.ctx.Done()
	if errors.Is(s.ctx.Err(), context.Canceled) {
		return nil, io.EOF
	}
	return nil, s.ctx.Err()
}

func (s *idlePlayAIDebugStream) Header() (gogrpcmetadata.MD, error) { return nil, nil }
func (s *idlePlayAIDebugStream) Trailer() gogrpcmetadata.MD         { return nil }
func (s *idlePlayAIDebugStream) CloseSend() error                   { return nil }
func (s *idlePlayAIDebugStream) Context() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}
func (s *idlePlayAIDebugStream) SendMsg(any) error { return nil }
func (s *idlePlayAIDebugStream) RecvMsg(any) error { return nil }

func websocketURLFromHTTP(baseURL string, path string) string {
	return "ws" + strings.TrimPrefix(strings.TrimSpace(baseURL), "http") + path
}

func requireCookieValue(t *testing.T, cookies []*http.Cookie, name string) string {
	t.Helper()

	for _, cookie := range cookies {
		if cookie != nil && cookie.Name == name && strings.TrimSpace(cookie.Value) != "" {
			return cookie.Value
		}
	}
	t.Fatalf("cookie %q not found", name)
	return ""
}

func decodePlayPayload[T any](t *testing.T, payload json.RawMessage) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatalf("decode play payload: %v", err)
	}
	return value
}

func sessionCookieHeader(sessionID string) http.Header {
	header := http.Header{}
	header.Set("Cookie", fmt.Sprintf("play_session=%s", strings.TrimSpace(sessionID)))
	return header
}
