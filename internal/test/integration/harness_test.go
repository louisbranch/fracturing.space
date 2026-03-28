//go:build integration

package integration

import (
	"context"
	"fmt"
	"hash/fnv"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	grpcauthctx "github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	userhubapp "github.com/louisbranch/fracturing.space/internal/services/userhub/app"
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
	mesh     *testkit.Mesh
	grpcAddr string
	authAddr string
}

func newSuiteFixture(t *testing.T) *suiteFixture {
	t.Helper()
	runtime := testkit.StartGameRuntime(t, testkit.GameRuntimeConfig{
		ContentSeedProfile: testkit.ContentSeedProfileIntegration,
		JoinGrantIssuer:    joinGrantIssuer,
		JoinGrantAudience:  joinGrantAudience,
	})
	return &suiteFixture{
		mesh:     runtime.Mesh,
		grpcAddr: runtime.GameAddr,
		authAddr: runtime.AuthAddr,
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

func (f *suiteFixture) startSocialServer(t *testing.T) string {
	t.Helper()
	return f.mesh.StartSocialServer()
}

func (f *suiteFixture) startNotificationsServer(t *testing.T) string {
	t.Helper()
	return f.mesh.StartNotificationsServer()
}

func (f *suiteFixture) startInviteServer(t *testing.T) string {
	t.Helper()
	return f.mesh.StartInviteServer()
}

func (f *suiteFixture) startDiscoveryServer(t *testing.T) string {
	t.Helper()
	return f.mesh.StartDiscoveryServer()
}

func (f *suiteFixture) startWorkerRuntime(t *testing.T) string {
	t.Helper()
	return f.mesh.StartWorkerRuntime()
}

func (f *suiteFixture) startUserHubServer(t *testing.T) string {
	t.Helper()
	return f.mesh.StartUserHubServer(userhubapp.RuntimeConfig{
		AuthAddr:          f.authAddr,
		GameAddr:          f.grpcAddr,
		InviteAddr:        f.mesh.StartInviteServer(),
		SocialAddr:        f.mesh.StartSocialServer(),
		NotificationsAddr: f.mesh.StartNotificationsServer(),
		StatusAddr:        pickUnusedAddress(t),
		CacheFreshTTL:     time.Minute,
		CacheStaleTTL:     5 * time.Minute,
	})
}

// ctx returns a context with the suite's user identity attached as gRPC metadata.
func (s *integrationSuite) ctx(parent context.Context) context.Context {
	return withUserID(parent, s.userID)
}

var (
	joinGrantIssuer   = "test-issuer"
	joinGrantAudience = "game-service"
)

// integrationTimeout returns the default timeout for integration calls.
func integrationTimeout() time.Duration {
	return 10 * time.Second
}

// startGRPCServer boots the game server and returns its address and shutdown function.
func startGRPCServer(t *testing.T) (string, string, func()) {
	t.Helper()
	runtime := testkit.StartGameRuntime(t, testkit.GameRuntimeConfig{
		ContentSeedProfile: testkit.ContentSeedProfileIntegration,
		JoinGrantIssuer:    joinGrantIssuer,
		JoinGrantAudience:  joinGrantAudience,
	})
	return runtime.GameAddr, runtime.AuthAddr, func() {}
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
	return testkit.SignJoinGrantToken(t, joinGrantIssuer, joinGrantAudience, campaignID, inviteID, userID, now)
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
