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
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/app"
	inviteapp "github.com/louisbranch/fracturing.space/internal/services/invite/app"
	grpcauthctx "github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/louisbranch/fracturing.space/internal/test/testkit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// integrationSuite bundles gRPC game clients and user identity for subtests.
type integrationSuite struct {
	conn        *grpc.ClientConn
	campaign    statev1.CampaignServiceClient
	participant statev1.ParticipantServiceClient
	character   statev1.CharacterServiceClient
	session     statev1.SessionServiceClient
	fork        statev1.ForkServiceClient
	userID      string
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

func (f *suiteFixture) newGameSuite(t *testing.T, userID string) *integrationSuite {
	t.Helper()

	conn, err := grpc.NewClient(
		f.grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial game gRPC: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Logf("close game gRPC: %v", closeErr)
		}
	})

	return &integrationSuite{
		conn:        conn,
		campaign:    statev1.NewCampaignServiceClient(conn),
		participant: statev1.NewParticipantServiceClient(conn),
		character:   statev1.NewCharacterServiceClient(conn),
		session:     statev1.NewSessionServiceClient(conn),
		fork:        statev1.NewForkServiceClient(conn),
		userID:      userID,
	}
}

// ctx returns a context with the suite's user identity attached as gRPC metadata.
func (s *integrationSuite) ctx(parent context.Context) context.Context {
	return withUserID(parent, s.userID)
}

var (
	joinGrantIssuer     = "test-issuer"
	joinGrantAudience   = "game-service"
	joinGrantKeyOnce    sync.Once
	joinGrantPrivateKey ed25519.PrivateKey
	joinGrantPublicKey  ed25519.PublicKey

	sharedFixtureOnce sync.Once
	sharedFixtureData suiteFixture
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
	grpcServer, err := app.NewWithAddr("127.0.0.1:0")
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
		grpcServer, err := app.NewWithAddr("127.0.0.1:0")
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

const integrationSharedFixtureEnv = "INTEGRATION_SHARED_FIXTURE"

func integrationSharedFixtureEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(integrationSharedFixtureEnv)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
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
				HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{Heritage: &daggerheartv1.DaggerheartCreationStepHeritageSelectionInput{
					FirstFeatureAncestryId:  "heritage.human",
					SecondFeatureAncestryId: "heritage.human",
					CommunityId:             "heritage.highborne",
				}},
				TraitsInput:     &daggerheartv1.DaggerheartCreationStepTraitsInput{Agility: 2, Strength: 1, Finesse: 1, Instinct: 0, Presence: 0, Knowledge: -1},
				DetailsInput:    &daggerheartv1.DaggerheartCreationStepDetailsInput{Description: "A brave adventurer."},
				EquipmentInput:  &daggerheartv1.DaggerheartCreationStepEquipmentInput{WeaponIds: []string{"weapon.longsword"}, ArmorId: "armor.gambeson-armor", PotionItemId: "item.minor-health-potion"},
				BackgroundInput: &daggerheartv1.DaggerheartCreationStepBackgroundInput{Background: "integration background"},
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

// dialGRPCWithServiceID dials a gRPC address with a service identity
// interceptor so calls carry the x-fracturing-space-service-id header.
func dialGRPCWithServiceID(t *testing.T, addr, serviceID string) *grpc.ClientConn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithChainUnaryInterceptor(grpcauthctx.ServiceIDUnaryClientInterceptor(serviceID)),
		grpc.WithChainStreamInterceptor(grpcauthctx.ServiceIDStreamClientInterceptor(serviceID)),
	)
	if err != nil {
		t.Fatalf("dial grpc (service-id=%s) %s: %v", serviceID, addr, err)
	}
	return conn
}

// pickUnusedAddress binds an ephemeral TCP port and returns its address.
func pickUnusedAddress(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick unused address: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

// startInviteServer boots an invite service against the given game and auth
// servers and returns its gRPC address. The server is shut down when t ends.
func startInviteServer(t *testing.T, gameAddr, authAddr string) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "invite-test.db")
	t.Setenv("FRACTURING_SPACE_INVITE_DB_PATH", dbPath)

	ctx, cancel := context.WithCancel(context.Background())

	server, err := inviteapp.NewWithAddr(ctx, "127.0.0.1:0", gameAddr, authAddr)
	if err != nil {
		cancel()
		t.Fatalf("new invite server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(ctx)
	}()

	addr := server.Addr()
	waitForGRPCHealth(t, addr)

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Logf("invite server error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Logf("timed out waiting for invite server to stop")
		}
	})

	return addr
}
