package campaigntransport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	daggerhearttestkit "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/testkit"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// assertStatusCode verifies the gRPC status code for an error.
// It wraps grpcerror.HandleDomainError as a fallback before delegating to
// grpcassert.StatusCode, because transport tests in this package exercise
// handlers that may return unwrapped domain errors.
func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v, got nil", want)
	}
	if _, ok := status.FromError(err); !ok {
		err = grpcerror.HandleDomainError(err)
	}
	grpcassert.StatusCode(t, err, want)
}

// testRuntime is a shared write-path runtime configured once for all tests.
var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	testRuntime = runtimekit.SetupRuntime()
	os.Exit(m.Run())
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	return data
}

// testApplier returns a projection.Applier wired to the stores in deps so that
// domain write paths can project events into the same fakes used for assertions.
func testApplier(deps Deps) projection.Applier {
	return projection.Applier{
		Campaign:    deps.Campaign,
		Participant: deps.Participant,
		Session:     deps.Session,
	}
}

// testDepsBuilder reduces repeated store construction in tests.
type testDepsBuilder struct {
	Campaign    *gametest.FakeCampaignStore
	Participant *gametest.FakeParticipantStore
	Event       *gametest.FakeEventStore
	Session     *gametest.FakeSessionStore

	domain       handler.Domain
	writeRuntime bool
}

func newTestDeps() *testDepsBuilder {
	return &testDepsBuilder{
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: gametest.NewFakeParticipantStore(),
		Event:       gametest.NewFakeEventStore(),
	}
}

func (b *testDepsBuilder) withSession() *testDepsBuilder {
	b.Session = gametest.NewFakeSessionStore()
	return b
}

func testSystemRegistries(t *testing.T) (*bridge.MetadataRegistry, *module.Registry) {
	t.Helper()
	return mustTestSystemRegistries()
}

func mustTestSystemRegistries() (*bridge.MetadataRegistry, *module.Registry) {
	metadata := bridge.NewMetadataRegistry()
	if err := metadata.Register(daggerheartdomain.NewRegistrySystem()); err != nil {
		panic("register daggerheart metadata system: " + err.Error())
	}
	modules := module.NewRegistry()
	if err := modules.Register(daggerheartdomain.NewModule()); err != nil {
		panic("register daggerheart module: " + err.Error())
	}
	return metadata, modules
}

func (b *testDepsBuilder) withDomain(d handler.Domain) *testDepsBuilder {
	b.domain = d
	b.writeRuntime = true
	return b
}

func (b *testDepsBuilder) build() Deps {
	d := Deps{
		Campaign:    b.Campaign,
		Participant: b.Participant,
	}
	if b.Session != nil {
		d.Session = b.Session
	}
	if b.domain != nil {
		d.Write.Executor = b.domain
	}
	if b.writeRuntime {
		d.Write.Runtime = testRuntime
	}
	return d
}

type fakeDomainEngine struct {
	store         storage.EventStore
	result        engine.Result
	resultsByType map[command.Type]engine.Result
	calls         int
	lastCommand   command.Command
	commands      []command.Command
}

func (f *fakeDomainEngine) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	f.calls++
	f.lastCommand = cmd
	f.commands = append(f.commands, cmd)

	result := f.result
	if len(f.resultsByType) > 0 {
		if selected, ok := f.resultsByType[cmd.Type]; ok {
			result = selected
		}
	}
	if f.store == nil {
		return result, nil
	}
	if len(result.Decision.Events) == 0 {
		return result, nil
	}
	stored := make([]event.Event, 0, len(result.Decision.Events))
	for _, evt := range result.Decision.Events {
		storedEvent, err := f.store.AppendEvent(ctx, evt)
		if err != nil {
			return engine.Result{}, err
		}
		stored = append(stored, storedEvent)
	}
	result.Decision.Events = stored
	return result, nil
}

func ownerParticipantStore(campaignID string) *gametest.FakeParticipantStore {
	store := gametest.NewFakeParticipantStore()
	store.Participants[campaignID] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord(campaignID, "owner-1"),
	}
	return store
}

// orderedCampaignStore is an in-memory campaign store with deterministic
// ordering for pagination tests.
type orderedCampaignStore struct {
	Campaigns []storage.CampaignRecord
}

func (s *orderedCampaignStore) Put(_ context.Context, c storage.CampaignRecord) error {
	if s == nil {
		return fmt.Errorf("storage is not configured")
	}
	s.Campaigns = append(s.Campaigns, c)
	return nil
}

func (s *orderedCampaignStore) Get(_ context.Context, id string) (storage.CampaignRecord, error) {
	if s == nil {
		return storage.CampaignRecord{}, fmt.Errorf("storage is not configured")
	}
	for _, c := range s.Campaigns {
		if c.ID == id {
			return c, nil
		}
	}
	return storage.CampaignRecord{}, storage.ErrNotFound
}

func (s *orderedCampaignStore) List(_ context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	if s == nil {
		return storage.CampaignPage{}, fmt.Errorf("storage is not configured")
	}
	if pageSize <= 0 {
		return storage.CampaignPage{}, fmt.Errorf("page size must be greater than zero")
	}

	page := storage.CampaignPage{
		Campaigns: make([]storage.CampaignRecord, 0, pageSize),
	}

	start := 0
	if pageToken != "" {
		for idx, c := range s.Campaigns {
			if c.ID == pageToken {
				start = idx + 1
				break
			}
		}
	}
	if start < 0 || start > len(s.Campaigns) {
		start = 0
	}

	end := start + pageSize
	if end > len(s.Campaigns) {
		end = len(s.Campaigns)
	}

	page.Campaigns = append(page.Campaigns, s.Campaigns[start:end]...)
	if end < len(s.Campaigns) {
		page.NextPageToken = s.Campaigns[end-1].ID
	}
	return page, nil
}

type readinessServiceFixtureStores struct {
	campaign    *gametest.FakeCampaignStore
	participant *gametest.FakeParticipantStore
	character   *gametest.FakeCharacterStore
	session     *gametest.FakeSessionStore
}

type readinessServiceFixtureConfig struct {
	status            campaign.Status
	gmMode            campaign.GmMode
	aiAgentID         string
	locale            string
	includeHumanGM    bool
	includeAIGM       bool
	includePlayerSeat bool
}

func newReadinessServiceFixture(config readinessServiceFixtureConfig) (*CampaignService, readinessServiceFixtureStores) {
	stores := readinessServiceFixtureStores{
		campaign:    gametest.NewFakeCampaignStore(),
		participant: gametest.NewFakeParticipantStore(),
		character:   gametest.NewFakeCharacterStore(),
		session:     gametest.NewFakeSessionStore(),
	}

	status := config.status
	if status == "" {
		status = campaign.StatusActive
	}
	gmMode := config.gmMode
	if gmMode == "" {
		gmMode = campaign.GmModeHuman
	}
	locale := config.locale
	if locale == "" {
		locale = "en-US"
	}
	stores.campaign.Campaigns["c1"] = storage.CampaignRecord{
		ID:        "c1",
		Name:      "Campaign One",
		Locale:    locale,
		Status:    status,
		GmMode:    gmMode,
		AIAgentID: strings.TrimSpace(config.aiAgentID),
	}

	includeHumanGM := config.includeHumanGM
	includePlayerSeat := config.includePlayerSeat
	if !includeHumanGM && !config.includeAIGM {
		includeHumanGM = true
	}
	if !includePlayerSeat {
		includePlayerSeat = true
	}

	participants := map[string]storage.ParticipantRecord{}
	if includeHumanGM {
		participants["gm-1"] = storage.ParticipantRecord{
			ID:             "gm-1",
			CampaignID:     "c1",
			UserID:         "user-gm-1",
			Name:           "GM One",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessOwner,
		}
	}
	if config.includeAIGM {
		participants["ai-gm-1"] = storage.ParticipantRecord{
			ID:             "ai-gm-1",
			CampaignID:     "c1",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessOwner,
		}
	}
	if includePlayerSeat {
		participants["player-1"] = storage.ParticipantRecord{
			ID:             "player-1",
			CampaignID:     "c1",
			UserID:         "user-player-1",
			Name:           "Player One",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessMember,
		}
	}
	stores.participant.Participants["c1"] = participants

	stores.character.Characters["c1"] = map[string]storage.CharacterRecord{
		"char-1": {
			ID:            "char-1",
			CampaignID:    "c1",
			ParticipantID: "player-1",
		},
	}
	metadata, modules := mustTestSystemRegistries()
	systemStores := systemmanifest.ProjectionStores{Daggerheart: daggerhearttestkit.NewFakeDaggerheartStore()}

	service := NewCampaignService(Deps{
		Campaign:       stores.campaign,
		Participant:    stores.participant,
		Character:      stores.character,
		Session:        stores.session,
		SystemStores:   systemStores,
		SystemMetadata: metadata,
		SystemModules:  modules,
	})
	return service, stores
}

func assertReadinessHasBlockerCode(t *testing.T, report *statev1.CampaignSessionReadiness, code string) {
	t.Helper()
	if report == nil {
		t.Fatal("readiness report is nil")
	}
	if report.GetReady() {
		t.Fatalf("readiness.ready = true, want false with blocker %s", code)
	}
	for _, blocker := range report.GetBlockers() {
		if strings.TrimSpace(blocker.GetCode()) == code {
			return
		}
	}
	t.Fatalf("expected blocker code %q, got %v", code, readinessBlockerCodes(report.GetBlockers()))
}

func findReadinessBlocker(t *testing.T, report *statev1.CampaignSessionReadiness, code string) *statev1.CampaignSessionReadinessBlocker {
	t.Helper()
	if report == nil {
		t.Fatal("readiness report is nil")
	}
	for _, blocker := range report.GetBlockers() {
		if strings.TrimSpace(blocker.GetCode()) == code {
			return blocker
		}
	}
	t.Fatalf("expected blocker code %q, got %v", code, readinessBlockerCodes(report.GetBlockers()))
	return nil
}

func readinessBlockerCodes(blockers []*statev1.CampaignSessionReadinessBlocker) []string {
	codes := make([]string, 0, len(blockers))
	for _, blocker := range blockers {
		codes = append(codes, strings.TrimSpace(blocker.GetCode()))
	}
	return codes
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("slice len = %d, want %d; got=%v want=%v", len(got), len(want), got, want)
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("slice[%d] = %q, want %q; got=%v want=%v", idx, got[idx], want[idx], got, want)
		}
	}
}

// newTestCampaignService wraps newCampaignServiceWithDependencies with
// automatic Applier wiring so tests exercising domain write paths don't need
// to set it explicitly.
func newTestCampaignService(deps Deps, clock func() time.Time, idGenerator func() (string, error)) *CampaignService {
	deps.Applier = testApplier(deps)
	return newCampaignServiceWithDependencies(deps, clock, idGenerator)
}

// writeDeps returns a Deps with domain executor and runtime configured.
func writeDeps(domain handler.Domain, base Deps) Deps {
	base.Write = domainwrite.WritePath{
		Executor: domain,
		Runtime:  testRuntime,
	}
	return base
}

// withAuthClient returns a copy of deps with the AuthClient set.
func withAuthClient(deps Deps, authClient handler.AuthUserClient) Deps {
	deps.AuthClient = authClient
	return deps
}
