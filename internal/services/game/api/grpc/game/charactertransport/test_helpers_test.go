package charactertransport

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
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

// --- test helpers ---

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

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	return data
}

// testStoresBuilder reduces repeated store construction in tests.
type testStoresBuilder struct {
	Campaign    *gametest.FakeCampaignStore
	Participant *gametest.FakeParticipantStore
	Event       *gametest.FakeEventStore
	Character   *gametest.FakeCharacterStore
	Daggerheart *gametest.FakeDaggerheartStore

	domain       handler.Domain
	writeRuntime bool
}

func newTestStores() *testStoresBuilder {
	return &testStoresBuilder{
		Campaign:    gametest.NewFakeCampaignStore(),
		Participant: gametest.NewFakeParticipantStore(),
		Event:       gametest.NewFakeEventStore(),
	}
}

func (b *testStoresBuilder) withCharacter() *testStoresBuilder {
	b.Character = gametest.NewFakeCharacterStore()
	b.Daggerheart = gametest.NewFakeDaggerheartStore()
	return b
}

func (b *testStoresBuilder) withDomain(d handler.Domain) *testStoresBuilder {
	b.domain = d
	b.writeRuntime = true
	return b
}

func (b *testStoresBuilder) build() Deps {
	d := Deps{
		Auth: authz.PolicyDeps{
			Participant: b.Participant,
		},
		Campaign:    b.Campaign,
		Participant: b.Participant,
	}
	if b.Character != nil {
		d.Character = b.Character
		d.Auth.Character = b.Character
	}
	if b.Daggerheart != nil {
		d.Daggerheart = b.Daggerheart
		adapters, _ := manifest.AdapterRegistry(b.Daggerheart)
		d.Applier = projection.Applier{
			Campaign:  b.Campaign,
			Character: b.Character,
			Adapters:  adapters,
		}
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

func testDaggerheartProfile(overrides func(*daggerheart.CharacterProfile)) daggerheart.CharacterProfile {
	profile := daggerheart.CharacterProfile{
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

func testDaggerheartProfileReplacedEvent(
	t *testing.T,
	now time.Time,
	campaignID, characterID string,
	actorType event.ActorType,
	actorID string,
	profile daggerheart.CharacterProfile,
) event.Event {
	t.Helper()
	return event.Event{
		CampaignID:    ids.CampaignID(campaignID),
		Type:          daggerheart.EventTypeCharacterProfileReplaced,
		Timestamp:     now,
		ActorType:     actorType,
		ActorID:       actorID,
		EntityType:    "character",
		EntityID:      characterID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheart.CharacterProfileReplacedPayload{
			CharacterID: ids.CharacterID(characterID),
			Profile:     profile,
		}),
	}
}

func testCreateCharacterResults(
	t *testing.T,
	now time.Time,
	campaignID, characterID string,
	actorType event.ActorType,
	actorID string,
	createPayload any,
) map[command.Type]engine.Result {
	t.Helper()
	return map[command.Type]engine.Result{
		handler.CommandTypeCharacterCreate: {
			Decision: command.Accept(event.Event{
				CampaignID:  ids.CampaignID(campaignID),
				Type:        event.Type("character.created"),
				Timestamp:   now,
				ActorType:   actorType,
				ActorID:     actorID,
				EntityType:  "character",
				EntityID:    characterID,
				PayloadJSON: mustJSON(t, createPayload),
			}),
		},
	}
}

// Ensure fakeDomainEngine satisfies the interface at compile time.
var _ handler.Domain = (*fakeDomainEngine)(nil)
