package forktransport

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	testRuntime = gametest.SetupRuntime()
	os.Exit(m.Run())
}

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

func testDaggerheartProfile(overrides func(*daggerheartstate.CharacterProfile)) daggerheartstate.CharacterProfile {
	profile := daggerheartstate.CharacterProfile{
		Level:           1,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  1,
		SevereThreshold: 2,
		Proficiency:     1,
		ArmorScore:      0,
		ArmorMax:        0,
	}
	if overrides != nil {
		overrides(&profile)
	}
	return profile
}

func newServiceForTest(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) *Service {
	return newServiceWithDependencies(deps, clock, idGenerator)
}

func testApplier(t *testing.T, deps Deps, dhStore *gametest.FakeDaggerheartStore) projection.Applier {
	t.Helper()
	adapters, err := manifest.AdapterRegistry(manifest.ProjectionStores{Daggerheart: dhStore})
	if err != nil {
		t.Fatalf("build adapter registry: %v", err)
	}
	return projection.Applier{
		Campaign:     deps.Campaign,
		Character:    deps.Character,
		CampaignFork: deps.CampaignFork,
		Participant:  deps.Participant,
		Adapters:     adapters,
	}
}

func appendEvent(t *testing.T, store *gametest.FakeEventStore, evt event.Event) event.Event {
	t.Helper()
	stored, err := store.AppendEvent(context.Background(), evt)
	if err != nil {
		t.Fatalf("append event failed: %v", err)
	}
	return stored
}

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	return data
}

func intPtr(value int) *int {
	return &value
}
