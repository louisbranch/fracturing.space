package campaigntransport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testRuntime is a shared write-path runtime configured once for all tests.
var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	testRuntime = gametest.SetupRuntime()
	os.Exit(m.Run())
}

// assertStatusCode verifies the gRPC status code for an error.
func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		err = grpcerror.HandleDomainError(err)
		statusErr, ok = status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %T", err)
		}
	}
	if statusErr.Code() != want {
		t.Fatalf("status code = %v, want %v (message: %s)", statusErr.Code(), want, statusErr.Message())
	}
}

func assertStatusMessage(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with message %q", want)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}
	if statusErr.Message() != want {
		t.Fatalf("status message = %q, want %q", statusErr.Message(), want)
	}
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

// newTestCampaignService wraps newCampaignServiceWithDependencies with
// automatic Applier wiring so tests exercising domain write paths don't need
// to set it explicitly.
func newTestCampaignService(deps Deps, clock func() time.Time, idGenerator func() (string, error)) *CampaignService {
	deps.Applier = testApplier(deps)
	return newCampaignServiceWithDependencies(deps, clock, idGenerator)
}

// newTestCampaignAIService wraps newCampaignAIServiceWithDependencies for tests.
func newTestCampaignAIService(deps Deps, clock func() time.Time, idGenerator func() (string, error)) *CampaignAIService {
	return newCampaignAIServiceWithDependencies(deps, clock, idGenerator)
}

// writeDeps returns a Deps with domain executor and runtime configured.
func writeDeps(domain handler.Domain, base Deps) Deps {
	base.Write = domainwriteexec.WritePath{
		Executor: domain,
		Runtime:  testRuntime,
	}
	return base
}

// withAuthClient returns a copy of deps with the AuthClient set.
func withAuthClient(deps Deps, authClient authv1.AuthServiceClient) Deps {
	deps.AuthClient = authClient
	return deps
}
