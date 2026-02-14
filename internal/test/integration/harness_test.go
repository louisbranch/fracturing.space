//go:build integration

package integration

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	authserver "github.com/louisbranch/fracturing.space/internal/services/auth/app"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	server "github.com/louisbranch/fracturing.space/internal/services/game/app"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

// integrationSuite shares resources across integration subtests.
type integrationSuite struct {
	client *mcp.ClientSession
	userID string
}

var (
	joinGrantIssuer     = "test-issuer"
	joinGrantAudience   = "game-service"
	joinGrantKeyOnce    sync.Once
	joinGrantPrivateKey ed25519.PrivateKey
	joinGrantPublicKey  ed25519.PublicKey
)

// integrationTimeout returns the default timeout for integration calls.
func integrationTimeout() time.Duration {
	return 10 * time.Second
}

// startGRPCServer boots the game server and returns its address and shutdown function.
func startGRPCServer(t *testing.T) (string, string, func()) {
	t.Helper()

	setTempDBPath(t)
	setTempAuthDBPath(t)
	setJoinGrantEnv(t)
	authAddr, stopAuth := startAuthServer(t)
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", authAddr)

	ctx, cancel := context.WithCancel(context.Background())
	grpcServer, err := server.NewWithAddr("127.0.0.1:0")
	if err != nil {
		cancel()
		stopAuth()
		t.Fatalf("new game server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	addr := grpcServer.Addr()
	waitForGRPCHealth(t, addr)
	stop := func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("game server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for game server to stop")
		}
		stopAuth()
	}

	return addr, authAddr, stop
}

func setJoinGrantEnv(t *testing.T) {
	t.Helper()

	joinGrantKeyOnce.Do(func() {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate join grant key: %v", err)
		}
		joinGrantPublicKey = publicKey
		joinGrantPrivateKey = privateKey
	})

	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", joinGrantIssuer)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", joinGrantAudience)
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(joinGrantPublicKey))
	t.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.RawStdEncoding.EncodeToString(joinGrantPrivateKey))
}

func startAuthServer(t *testing.T) (string, func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	authServer, err := authserver.New(0, "")
	if err != nil {
		cancel()
		t.Fatalf("new auth server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- authServer.Serve(ctx)
	}()

	authAddr := authServer.Addr()
	waitForGRPCHealth(t, authAddr)
	stop := func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("auth server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for auth server to stop")
		}
	}

	return authAddr, stop
}

// startMCPClient boots the MCP stdio process and returns a client session and shutdown function.
func startMCPClient(t *testing.T, grpcAddr string) (*mcp.ClientSession, func()) {
	t.Helper()

	cmd := exec.Command("go", "run", "./cmd/mcp")
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), fmt.Sprintf("FRACTURING_SPACE_GAME_ADDR=%s", grpcAddr))
	cmd.Stderr = os.Stderr

	transport := &mcp.CommandTransport{Command: cmd}
	client := mcp.NewClient(&mcp.Implementation{Name: "integration-client", Version: "dev"}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientSession, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("connect MCP client: %v", err)
	}

	closeClient := func() {
		closeErr := clientSession.Close()
		if closeErr != nil {
			t.Fatalf("close MCP client: %v", closeErr)
		}
	}

	return clientSession, closeClient
}

func createAuthUser(t *testing.T, authAddr, displayName string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial auth server: %v", err)
	}
	defer conn.Close()

	client := authv1.NewAuthServiceClient(conn)
	resp, err := client.CreateUser(ctx, &authv1.CreateUserRequest{DisplayName: displayName})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID := resp.GetUser().GetId()
	if userID == "" {
		t.Fatal("create user: missing user id")
	}
	return userID
}

func withUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(grpcmeta.UserIDHeader, userID))
}

func withSessionID(ctx context.Context, sessionID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(grpcmeta.SessionIDHeader, sessionID))
}

func joinGrantToken(t *testing.T, campaignID, inviteID, userID string, now time.Time) string {
	t.Helper()
	if joinGrantPrivateKey == nil {
		t.Fatal("join grant key is not configured")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	headerJSON, err := json.Marshal(map[string]string{
		"alg": "EdDSA",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("encode join grant header: %v", err)
	}
	payloadJSON, err := json.Marshal(map[string]any{
		"iss":         joinGrantIssuer,
		"aud":         joinGrantAudience,
		"exp":         now.Add(5 * time.Minute).Unix(),
		"iat":         now.Unix(),
		"jti":         fmt.Sprintf("jti-%d", now.UnixNano()),
		"campaign_id": campaignID,
		"invite_id":   inviteID,
		"user_id":     userID,
	})
	if err != nil {
		t.Fatalf("encode join grant payload: %v", err)
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload
	signature := ed25519.Sign(joinGrantPrivateKey, []byte(signingInput))
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + encodedSig
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

// decodeStructuredContent decodes structured MCP content into the target type.
func decodeStructuredContent[T any](t *testing.T, value any) T {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var output T
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
	return output
}

// parseCampaignListPayload decodes a campaign list JSON payload.
func parseCampaignListPayload(t *testing.T, raw string) domain.CampaignListPayload {
	t.Helper()

	var payload domain.CampaignListPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal campaign list payload: %v", err)
	}
	return payload
}

// parseParticipantListPayload decodes a participant list JSON payload.
func parseParticipantListPayload(t *testing.T, raw string) domain.ParticipantListPayload {
	t.Helper()

	var payload domain.ParticipantListPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal participant list payload: %v", err)
	}
	return payload
}

// readParticipantList fetches the participant list resource for a campaign.
func readParticipantList(t *testing.T, client *mcp.ClientSession, campaignID string) domain.ParticipantListPayload {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	res, err := client.ReadResource(ctx, &mcp.ReadResourceParams{URI: fmt.Sprintf("campaign://%s/participants", campaignID)})
	if err != nil {
		t.Fatalf("read participants resource: %v", err)
	}
	if res == nil || len(res.Contents) == 0 || res.Contents[0].Text == "" {
		t.Fatal("participants resource response missing content")
	}

	return parseParticipantListPayload(t, res.Contents[0].Text)
}

// setContext sets the MCP context for campaign/participant identity.
func setContext(t *testing.T, client *mcp.ClientSession, campaignID, participantID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	result, err := client.CallTool(ctx, &mcp.CallToolParams{
		Name: "set_context",
		Arguments: map[string]any{
			"campaign_id":    campaignID,
			"participant_id": participantID,
		},
	})
	if err != nil {
		t.Fatalf("call set_context: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("set_context failed: %+v", result)
	}
}

// findCampaignByID searches for a campaign entry by ID.
func findCampaignByID(payload domain.CampaignListPayload, id string) (domain.CampaignListEntry, bool) {
	for _, campaign := range payload.Campaigns {
		if campaign.ID == id {
			return campaign, true
		}
	}
	return domain.CampaignListEntry{}, false
}

// parseRFC3339 parses an RFC3339 timestamp string.
func parseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()

	if value == "" {
		t.Fatal("expected non-empty timestamp")
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse timestamp %q: %v", value, err)
	}
	return parsed
}

func newEventClient(t *testing.T, grpcAddr string) (statev1.EventServiceClient, func()) {
	t.Helper()

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}

	closeConn := func() {
		if err := conn.Close(); err != nil {
			t.Fatalf("close gRPC: %v", err)
		}
	}

	return statev1.NewEventServiceClient(conn), closeConn
}

func requireLatestSeq(t *testing.T, ctx context.Context, client statev1.EventServiceClient, campaignID string) uint64 {
	t.Helper()

	response, err := client.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
	})
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if response == nil || len(response.Events) == 0 {
		return 0
	}
	return response.Events[0].Seq
}

func requireEventAfterSeq(t *testing.T, ctx context.Context, client statev1.EventServiceClient, campaignID, eventType string, before uint64) {
	t.Helper()

	response, err := client.ListEvents(ctx, &statev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   1,
		OrderBy:    "seq desc",
		Filter:     "type = \"" + eventType + "\"",
	})
	if err != nil {
		t.Fatalf("list events for %s: %v", eventType, err)
	}
	if response == nil || len(response.Events) == 0 {
		t.Fatalf("expected event type %s in campaign %s", eventType, campaignID)
	}
	if response.Events[0].Seq <= before {
		t.Fatalf("expected %s to append event: before=%d after=%d", eventType, before, response.Events[0].Seq)
	}
}

func requireEventTypesAfterSeq(t *testing.T, ctx context.Context, client statev1.EventServiceClient, campaignID string, before uint64, eventTypes ...string) uint64 {
	t.Helper()

	after := requireLatestSeq(t, ctx, client, campaignID)
	if after <= before {
		t.Fatalf("expected events to append: before=%d after=%d", before, after)
	}
	for _, eventType := range eventTypes {
		requireEventAfterSeq(t, ctx, client, campaignID, eventType, before)
	}
	return after
}

// setTempDBPath configures a temporary database for integration tests.
func setTempDBPath(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_GAME_EVENTS_DB_PATH", filepath.Join(base, "game-events.db"))
	t.Setenv("FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH", filepath.Join(base, "game-projections.db"))
	t.Setenv("FRACTURING_SPACE_GAME_EVENT_HMAC_KEY", "test-key")
}

func setTempAuthDBPath(t *testing.T) {
	t.Helper()
	base := t.TempDir()
	t.Setenv("FRACTURING_SPACE_AUTH_DB_PATH", filepath.Join(base, "auth.db"))
}

// repoRoot returns the repository root by walking up to go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve runtime caller")
	}

	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatalf("go.mod not found from %s", filename)
	return ""
}

// waitForGRPCHealth waits for the gRPC health check to report SERVING.
func waitForGRPCHealth(t *testing.T, addr string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial game server: %v", err)
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	backoff := 100 * time.Millisecond
	for {
		callCtx, callCancel := context.WithTimeout(ctx, time.Second)
		response, err := healthClient.Check(callCtx, &grpc_health_v1.HealthCheckRequest{Service: ""})
		callCancel()
		if err == nil && response.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			return
		}

		select {
		case <-ctx.Done():
			if err != nil {
				t.Fatalf("wait for gRPC health: %v", err)
			}
			t.Fatalf("wait for gRPC health: %v", ctx.Err())
		case <-time.After(backoff):
		}

		if backoff < time.Second {
			backoff *= 2
			if backoff > time.Second {
				backoff = time.Second
			}
		}
	}
}

// intPointer returns a pointer to the provided int value.
func intPointer(value int) *int {
	return &value
}
