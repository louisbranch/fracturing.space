package sessiontransport

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testRuntime is a shared write-path runtime configured once for all tests.
var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	testRuntime = runtimekit.SetupRuntime()
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
		Campaign:           deps.Campaign,
		Session:            deps.Session,
		SessionGate:        deps.SessionGate,
		SessionSpotlight:   deps.SessionSpotlight,
		SessionInteraction: deps.SessionInteraction,
	}
}

// newTestSessionService wraps newSessionServiceWithDependencies with automatic
// Applier wiring so tests exercising domain write paths don't need to set it
// explicitly.
func newTestSessionService(deps Deps, clock func() time.Time, idGenerator func() (string, error)) *SessionService {
	deps.Applier = testApplier(deps)
	return newSessionServiceWithDependencies(deps, clock, idGenerator)
}

func sessionManagerParticipantStore(campaignID string) *gametest.FakeParticipantStore {
	store := gametest.NewFakeParticipantStore()
	record := gametest.ManagerParticipantRecord(campaignID, "manager-1")
	record.Role = participant.RoleGM
	record.Controller = participant.ControllerHuman
	store.Participants[campaignID] = map[string]storage.ParticipantRecord{"manager-1": record}
	return store
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

func testWritePath(executor domainwrite.Executor) domainwrite.WritePath {
	return domainwrite.WritePath{Executor: executor, Runtime: testRuntime}
}
