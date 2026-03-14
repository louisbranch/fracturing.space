//go:build integration

package integration

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	server "github.com/louisbranch/fracturing.space/internal/services/game/app"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	mcpservice "github.com/louisbranch/fracturing.space/internal/services/mcp/service"
	"github.com/louisbranch/fracturing.space/internal/test/testkit"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// integrationSuite shares resources across integration subtests.
type integrationSuite struct {
	client *mcp.ClientSession
	userID string
}

// suiteFixture provides shared startup/shutdown wiring for integration tests.
type suiteFixture struct {
	grpcAddr string
	authAddr string
}

func newSuiteFixture(t *testing.T) *suiteFixture {
	t.Helper()
	grpcAddr, authAddr, stop := startGRPCServer(t)
	t.Cleanup(stop)
	return &suiteFixture{
		grpcAddr: grpcAddr,
		authAddr: authAddr,
	}
}

func (f *suiteFixture) newUserID(t *testing.T, username string) string {
	t.Helper()
	return createAuthUser(t, f.authAddr, username)
}

func (f *suiteFixture) newMCPClientSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	clientSession, closeClient := startMCPClient(t, f.grpcAddr)
	t.Cleanup(closeClient)
	return clientSession
}

var (
	joinGrantIssuer     = "test-issuer"
	joinGrantAudience   = "game-service"
	joinGrantKeyOnce    sync.Once
	joinGrantPrivateKey ed25519.PrivateKey
	joinGrantPublicKey  ed25519.PublicKey

	sharedFixtureOnce sync.Once
	sharedFixtureData suiteFixture

	mcpBinaryOnce sync.Once
	mcpBinaryPath string
	mcpBinaryErr  error
)

const (
	testAISessionGrantIssuer   = "fracturing-space-game"
	testAISessionGrantAudience = "fracturing-space-ai"
	testAISessionGrantTTL      = "10m"
	testAISessionGrantHMACKey  = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"
)

// integrationTimeout returns the default timeout for integration calls.
func integrationTimeout() time.Duration {
	return 10 * time.Second
}

// startGRPCServer boots the game server and returns its address and shutdown function.
func startGRPCServer(t *testing.T) (string, string, func()) {
	t.Helper()
	if integrationSharedFixtureEnabled() {
		shared := sharedSuiteFixture(t)
		return shared.grpcAddr, shared.authAddr, func() {}
	}

	setTempDBPath(t)
	setTempAuthDBPath(t)
	seedDaggerheartContent(t)
	setJoinGrantEnv(t)
	setAISessionGrantEnv(t)
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

func sharedSuiteFixture(t *testing.T) suiteFixture {
	t.Helper()
	sharedFixtureOnce.Do(func() {
		base, err := os.MkdirTemp("", "integration-shared-fixture-*")
		if err != nil {
			t.Fatalf("create shared fixture temp dir: %v", err)
		}

		testkit.SetGameDBPaths(t, base, os.Setenv)
		testkit.SetAuthDBPath(t, base, os.Setenv)

		seedDaggerheartContent(t)
		setJoinGrantProcessEnv(t)
		setAISessionGrantProcessEnv(t)

		authAddr, _ := startAuthServer(t)
		if err := os.Setenv("FRACTURING_SPACE_AUTH_ADDR", authAddr); err != nil {
			t.Fatalf("set shared auth addr env: %v", err)
		}

		ctx := context.Background()
		grpcServer, err := server.NewWithAddr("127.0.0.1:0")
		if err != nil {
			t.Fatalf("new shared game server: %v", err)
		}
		go func() {
			if serveErr := grpcServer.Serve(ctx); serveErr != nil {
				fmt.Fprintf(os.Stderr, "shared integration game server error: %v\n", serveErr)
			}
		}()

		grpcAddr := grpcServer.Addr()
		waitForGRPCHealth(t, grpcAddr)

		sharedFixtureData = suiteFixture{
			grpcAddr: grpcAddr,
			authAddr: authAddr,
		}
	})

	if strings.TrimSpace(sharedFixtureData.grpcAddr) == "" || strings.TrimSpace(sharedFixtureData.authAddr) == "" {
		t.Fatal("shared integration fixture failed to initialize")
	}
	return sharedFixtureData
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

func setJoinGrantProcessEnv(t *testing.T) {
	t.Helper()

	joinGrantKeyOnce.Do(func() {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate join grant key: %v", err)
		}
		joinGrantPublicKey = publicKey
		joinGrantPrivateKey = privateKey
	})

	if err := os.Setenv("FRACTURING_SPACE_JOIN_GRANT_ISSUER", joinGrantIssuer); err != nil {
		t.Fatalf("set join grant issuer: %v", err)
	}
	if err := os.Setenv("FRACTURING_SPACE_JOIN_GRANT_AUDIENCE", joinGrantAudience); err != nil {
		t.Fatalf("set join grant audience: %v", err)
	}
	if err := os.Setenv("FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY", base64.RawStdEncoding.EncodeToString(joinGrantPublicKey)); err != nil {
		t.Fatalf("set join grant public key: %v", err)
	}
	if err := os.Setenv("FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY", base64.RawStdEncoding.EncodeToString(joinGrantPrivateKey)); err != nil {
		t.Fatalf("set join grant private key: %v", err)
	}
}

func setAISessionGrantEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", testAISessionGrantIssuer)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", testAISessionGrantAudience)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", testAISessionGrantHMACKey)
	t.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", testAISessionGrantTTL)
}

func setAISessionGrantProcessEnv(t *testing.T) {
	t.Helper()
	if err := os.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_ISSUER", testAISessionGrantIssuer); err != nil {
		t.Fatalf("set ai session grant issuer: %v", err)
	}
	if err := os.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_AUDIENCE", testAISessionGrantAudience); err != nil {
		t.Fatalf("set ai session grant audience: %v", err)
	}
	if err := os.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY", testAISessionGrantHMACKey); err != nil {
		t.Fatalf("set ai session grant hmac key: %v", err)
	}
	if err := os.Setenv("FRACTURING_SPACE_AI_SESSION_GRANT_TTL", testAISessionGrantTTL); err != nil {
		t.Fatalf("set ai session grant ttl: %v", err)
	}
}

func startAuthServer(t *testing.T) (string, func()) {
	t.Helper()
	return testkit.StartAuthServer(t)
}

const (
	integrationMCPTransportEnv    = "INTEGRATION_MCP_TRANSPORT"
	integrationMCPTransportStdIO  = "stdio"
	integrationMCPTransportMemory = "inmemory"
	integrationSharedFixtureEnv   = "INTEGRATION_SHARED_FIXTURE"
)

func integrationSharedFixtureEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(integrationSharedFixtureEnv)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

// startMCPClient connects a test MCP client. Default transport is in-memory for
// speed; set INTEGRATION_MCP_TRANSPORT=stdio to exercise process boundaries.
func startMCPClient(t *testing.T, grpcAddr string) (*mcp.ClientSession, func()) {
	t.Helper()

	transport := strings.ToLower(strings.TrimSpace(os.Getenv(integrationMCPTransportEnv)))
	if transport == "" || transport == integrationMCPTransportMemory {
		return startMCPClientInMemory(t, grpcAddr)
	}
	if transport == integrationMCPTransportStdIO {
		return startMCPClientStdio(t, grpcAddr)
	}
	t.Fatalf("unsupported %s %q", integrationMCPTransportEnv, transport)
	return nil, nil
}

func startMCPClientInMemory(t *testing.T, grpcAddr string) (*mcp.ClientSession, func()) {
	t.Helper()

	serverInstance, err := mcpservice.New(grpcAddr)
	if err != nil {
		t.Fatalf("new MCP server: %v", err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serveCtx, serveCancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- serverInstance.ServeWithTransport(serveCtx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "integration-client", Version: "dev"}, nil)
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()
	clientSession, err := client.Connect(connectCtx, clientTransport, nil)
	if err != nil {
		serveCancel()
		t.Fatalf("connect MCP in-memory client: %v", err)
	}

	closeClient := func() {
		if closeErr := clientSession.Close(); closeErr != nil {
			t.Fatalf("close MCP client: %v", closeErr)
		}
		serveCancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("MCP in-memory server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for MCP in-memory server to stop")
		}
	}

	return clientSession, closeClient
}

func startMCPClientStdio(t *testing.T, grpcAddr string) (*mcp.ClientSession, func()) {
	t.Helper()

	cmd := exec.Command(mcpBinaryForTests(t), "-addr="+grpcAddr)
	cmd.Dir = repoRoot(t)
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

func createAuthUser(t *testing.T, authAddr, username string) string {
	t.Helper()
	return testkit.CreateAuthUser(t, authAddr, username)
}

func uniqueTestUsername(t *testing.T, parts ...string) string {
	t.Helper()

	base := ""
	for _, part := range parts {
		if token := sanitizeTestToken(part); token != "" {
			base = token
			break
		}
	}
	if base == "" {
		base = "integration"
	}
	if first := base[0]; first < 'a' || first > 'z' {
		base = "u" + base
	}

	inputs := append(append([]string{}, parts...), t.Name())
	hasher := fnv.New32a()
	for _, input := range inputs {
		_, _ = hasher.Write([]byte(input))
		_, _ = hasher.Write([]byte{0})
	}
	suffix := fmt.Sprintf("%08x", hasher.Sum32())

	const maxUsernameLen = 32
	maxBaseLen := maxUsernameLen - len(suffix) - 1
	if maxBaseLen < 3 {
		maxBaseLen = 3
	}
	if len(base) > maxBaseLen {
		base = strings.Trim(base[:maxBaseLen], "-")
	}
	if len(base) < 3 {
		base = "usr"
	}
	return base + "-" + suffix
}

func sanitizeTestToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}

	sanitized := strings.Trim(builder.String(), "-")
	return sanitized
}

// mcpBinaryForTests builds the MCP binary once so integration tests control the
// actual child process instead of the `go run` wrapper.
func mcpBinaryForTests(t *testing.T) string {
	t.Helper()

	mcpBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "mcp-integration-bin-*")
		if err != nil {
			mcpBinaryErr = fmt.Errorf("create temp dir: %w", err)
			return
		}

		binaryName := "mcp-integration"
		if runtime.GOOS == "windows" {
			binaryName += ".exe"
		}
		mcpBinaryPath = filepath.Join(dir, binaryName)

		cmd := exec.Command("go", "build", "-o", mcpBinaryPath, "./cmd/mcp")
		cmd.Dir = repoRoot(t)
		output, err := cmd.CombinedOutput()
		if err != nil {
			mcpBinaryErr = fmt.Errorf("build mcp binary: %w: %s", err, strings.TrimSpace(string(output)))
		}
	})

	if mcpBinaryErr != nil {
		t.Fatalf("prepare MCP binary: %v", mcpBinaryErr)
	}
	if strings.TrimSpace(mcpBinaryPath) == "" {
		t.Fatal("prepare MCP binary: path is empty")
	}
	return mcpBinaryPath
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

func ensureSessionStartReadiness(
	t *testing.T,
	ctx context.Context,
	participantClient statev1.ParticipantServiceClient,
	characterClient statev1.CharacterServiceClient,
	campaignID string,
	ownerParticipantID string,
	controlledCharacterIDs ...string,
) string {
	t.Helper()

	if participantClient == nil {
		t.Fatal("participant client is required")
	}
	if characterClient == nil {
		t.Fatal("character client is required")
	}
	ownerParticipantID = strings.TrimSpace(ownerParticipantID)
	if ownerParticipantID == "" {
		t.Fatal("owner participant id is required")
	}

	participantsResp, err := participantClient.ListParticipants(ctx, &statev1.ListParticipantsRequest{
		CampaignId: campaignID,
		PageSize:   200,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	playerParticipantIDs := make([]string, 0)
	for _, p := range participantsResp.GetParticipants() {
		if p.GetRole() != statev1.ParticipantRole_PLAYER {
			continue
		}
		pid := strings.TrimSpace(p.GetId())
		if pid == "" {
			continue
		}
		playerParticipantIDs = append(playerParticipantIDs, pid)
	}
	if len(playerParticipantIDs) == 0 {
		participantResp, createErr := participantClient.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
			CampaignId: campaignID,
			Name:       "Readiness Player",
			Role:       statev1.ParticipantRole_PLAYER,
			Controller: statev1.Controller_CONTROLLER_HUMAN,
		})
		if createErr != nil {
			t.Fatalf("create readiness player participant: %v", createErr)
		}
		if participantResp == nil || participantResp.GetParticipant() == nil {
			t.Fatal("create readiness player participant returned empty participant")
		}
		playerParticipantID := strings.TrimSpace(participantResp.GetParticipant().GetId())
		if playerParticipantID == "" {
			t.Fatal("create readiness player participant returned empty id")
		}
		playerParticipantIDs = append(playerParticipantIDs, playerParticipantID)
	}

	seenCharacters := make(map[string]struct{}, len(controlledCharacterIDs))
	for _, characterID := range controlledCharacterIDs {
		characterID = strings.TrimSpace(characterID)
		if characterID == "" {
			continue
		}
		if _, exists := seenCharacters[characterID]; exists {
			continue
		}
		seenCharacters[characterID] = struct{}{}
		setCharacterController(t, ctx, characterClient, campaignID, characterID, ownerParticipantID)
	}

	characters := listAllCharactersForReadiness(t, ctx, characterClient, campaignID)
	fallbackController := ownerParticipantID
	if fallbackController == "" {
		fallbackController = playerParticipantIDs[0]
	}
	for _, ch := range characters {
		characterID := strings.TrimSpace(ch.GetId())
		if characterID == "" {
			continue
		}
		if strings.TrimSpace(ch.GetParticipantId().GetValue()) != "" {
			continue
		}
		setCharacterController(t, ctx, characterClient, campaignID, characterID, fallbackController)
	}

	characters = listAllCharactersForReadiness(t, ctx, characterClient, campaignID)
	for _, ch := range characters {
		characterID := strings.TrimSpace(ch.GetId())
		if characterID == "" {
			continue
		}
		ensureDaggerheartCreationReadiness(t, ctx, characterClient, campaignID, characterID)
	}

	playerCharacterCounts := make(map[string]int, len(playerParticipantIDs))
	for _, pid := range playerParticipantIDs {
		playerCharacterCounts[pid] = 0
	}
	characters = listAllCharactersForReadiness(t, ctx, characterClient, campaignID)
	for _, ch := range characters {
		pid := strings.TrimSpace(ch.GetParticipantId().GetValue())
		if _, ok := playerCharacterCounts[pid]; ok {
			playerCharacterCounts[pid]++
		}
	}

	for idx, pid := range playerParticipantIDs {
		if playerCharacterCounts[pid] > 0 {
			continue
		}
		createResp, createErr := characterClient.CreateCharacter(ctx, &statev1.CreateCharacterRequest{
			CampaignId: campaignID,
			Name:       fmt.Sprintf("Readiness Character %d", idx+1),
			Kind:       statev1.CharacterKind_PC,
		})
		if createErr != nil {
			t.Fatalf("create readiness character: %v", createErr)
		}
		if createResp == nil || createResp.GetCharacter() == nil {
			t.Fatal("create readiness character returned empty character")
		}
		characterID := strings.TrimSpace(createResp.GetCharacter().GetId())
		if characterID == "" {
			t.Fatal("create readiness character returned empty id")
		}
		setCharacterController(t, ctx, characterClient, campaignID, characterID, pid)
		ensureDaggerheartCreationReadiness(t, ctx, characterClient, campaignID, characterID)
	}

	return playerParticipantIDs[0]
}

func ensureDaggerheartCreationReadiness(
	t *testing.T,
	ctx context.Context,
	characterClient statev1.CharacterServiceClient,
	campaignID string,
	characterID string,
) {
	t.Helper()

	_, err := characterClient.ApplyCharacterCreationWorkflow(ctx, &statev1.ApplyCharacterCreationWorkflowRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemWorkflow: &statev1.ApplyCharacterCreationWorkflowRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationWorkflowInput{
				ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{ClassId: "class.guardian", SubclassId: "subclass.stalwart"},
				HeritageInput:      &daggerheartv1.DaggerheartCreationStepHeritageInput{AncestryId: "heritage.human", CommunityId: "heritage.highborne"},
				TraitsInput:        &daggerheartv1.DaggerheartCreationStepTraitsInput{Agility: 2, Strength: 1, Finesse: 1, Instinct: 0, Presence: 0, Knowledge: -1},
				DetailsInput:       &daggerheartv1.DaggerheartCreationStepDetailsInput{Description: "A brave adventurer."},
				EquipmentInput:     &daggerheartv1.DaggerheartCreationStepEquipmentInput{WeaponIds: []string{"weapon.longsword"}, ArmorId: "armor.gambeson-armor", PotionItemId: "item.minor-health-potion"},
				BackgroundInput:    &daggerheartv1.DaggerheartCreationStepBackgroundInput{Background: "integration background"},
				ExperiencesInput: &daggerheartv1.DaggerheartCreationStepExperiencesInput{Experiences: []*daggerheartv1.DaggerheartExperience{
					{Name: "integration experience", Modifier: 2},
					{Name: "integration patrol", Modifier: 2},
				}},
				DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{DomainCardIds: []string{"domain_card.valor-bare-bones", "domain_card.valor-shield-wall"}},
				ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{Connections: "integration connections"},
			},
		},
	})
	if err != nil {
		t.Fatalf("apply readiness workflow for %s: %v", characterID, err)
	}
}

func listAllCharactersForReadiness(
	t *testing.T,
	ctx context.Context,
	characterClient statev1.CharacterServiceClient,
	campaignID string,
) []*statev1.Character {
	t.Helper()

	pageToken := ""
	characters := make([]*statev1.Character, 0)
	for {
		resp, err := characterClient.ListCharacters(ctx, &statev1.ListCharactersRequest{
			CampaignId: campaignID,
			PageSize:   200,
			PageToken:  pageToken,
		})
		if err != nil {
			t.Fatalf("list characters: %v", err)
		}
		characters = append(characters, resp.GetCharacters()...)
		next := strings.TrimSpace(resp.GetNextPageToken())
		if next == "" {
			break
		}
		pageToken = next
	}
	return characters
}

func setCharacterController(
	t *testing.T,
	ctx context.Context,
	characterClient statev1.CharacterServiceClient,
	campaignID string,
	characterID string,
	participantID string,
) {
	t.Helper()

	_, err := characterClient.SetDefaultControl(ctx, &statev1.SetDefaultControlRequest{
		CampaignId:    campaignID,
		CharacterId:   strings.TrimSpace(characterID),
		ParticipantId: wrapperspb.String(strings.TrimSpace(participantID)),
	})
	if err != nil {
		t.Fatalf("set default control for %s: %v", characterID, err)
	}
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
	testkit.SetTempGameDBPaths(t)
}

// seedDaggerheartContent writes minimal catalog rows required by integration
// readiness setup so workflow apply can validate content IDs.
func seedDaggerheartContent(t *testing.T) {
	t.Helper()
	testkit.SeedDaggerheartContent(t, testkit.ContentSeedProfileIntegration)
}

func setTempAuthDBPath(t *testing.T) {
	t.Helper()
	testkit.SetTempAuthDBPath(t)
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
	testkit.WaitForGRPCHealth(t, addr)
}

// intPointer returns a pointer to the provided int value.
func intPointer(value int) *int {
	return &value
}
