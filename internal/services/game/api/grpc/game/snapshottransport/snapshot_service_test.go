package snapshottransport

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
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

// --- test helpers (local copies; not exported from root package) ---

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	// Simulate the ErrorConversionUnaryInterceptor: handlers may return
	// domain errors that the interceptor would convert to gRPC status.
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

func emptyDeps() Deps {
	return Deps{}
}

// testApplier builds a projection.Applier wired for daggerheart system events.
// This mirrors what Stores.Applier() does in the root game package.
func testApplier(dhStore projectionstore.Store) projection.Applier {
	adapters, err := systemmanifest.AdapterRegistry(dhStore)
	if err != nil {
		return projection.Applier{BuildErr: err}
	}
	return projection.Applier{Adapters: adapters}
}

func TestGetSnapshot_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.GetSnapshot(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSnapshot_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
		Character:   gametest.NewFakeCharacterStore(),
	})
	_, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSnapshot_CampaignNotFound(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
		Character:   gametest.NewFakeCharacterStore(),
	})
	_, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetSnapshot_RequiresCampaignReadPolicy(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
		Character:   gametest.NewFakeCharacterStore(),
	})

	_, err := svc.GetSnapshot(context.Background(), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetSnapshot_CampaignArchivedAllowed(t *testing.T) {
	// GetSnapshot uses CampaignOpRead which is allowed for all campaign statuses,
	// including archived campaigns. This allows viewing historical campaign state.
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	characterStore := gametest.NewFakeCharacterStore()

	campaignStore.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 5}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("GetSnapshot response has nil snapshot")
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 5 {
		t.Errorf("Snapshot GmFear = %d, want %d", dh.GetGmFear(), 5)
	}
}

func TestGetSnapshot_Success_NoCharacters(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	characterStore := gametest.NewFakeCharacterStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 5}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if resp.Snapshot == nil {
		t.Fatal("GetSnapshot response has nil snapshot")
	}
	if resp.Snapshot.CampaignId != "c1" {
		t.Errorf("Snapshot CampaignId = %q, want %q", resp.Snapshot.CampaignId, "c1")
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 5 {
		t.Errorf("Snapshot GmFear = %d, want %d", dh.GetGmFear(), 5)
	}
	if len(resp.Snapshot.CharacterStates) != 0 {
		t.Errorf("Snapshot CharacterStates = %d, want 0", len(resp.Snapshot.CharacterStates))
	}
}

func TestGetSnapshot_Success_WithCharacters(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	characterStore := gametest.NewFakeCharacterStore()
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 3}
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
		"ch2": {CampaignID: "c1", CharacterID: "ch2", Hp: 12, Hope: 2, Stress: 0},
	}
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {ID: "ch1", CampaignID: "c1", Name: "Hero", Kind: character.KindPC, CreatedAt: now, UpdatedAt: now},
		"ch2": {ID: "ch2", CampaignID: "c1", Name: "Sidekick", Kind: character.KindPC, CreatedAt: now, UpdatedAt: now},
	}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 3 {
		t.Errorf("Snapshot GmFear = %d, want %d", dh.GetGmFear(), 3)
	}
	if len(resp.Snapshot.CharacterStates) != 2 {
		t.Errorf("Snapshot CharacterStates = %d, want 2", len(resp.Snapshot.CharacterStates))
	}
}

func TestGetSnapshot_Success_DefaultGmFear(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	characterStore := gametest.NewFakeCharacterStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	// No DaggerheartSnapshot entry - should default to 0

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Character:   characterStore,
	})

	resp, err := svc.GetSnapshot(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.GetSnapshotRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("GetSnapshot returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 0 {
		t.Errorf("Snapshot GmFear = %d, want 0 (default)", dh.GetGmFear())
	}
}

func TestPatchCharacterState_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.PatchCharacterState(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_MissingCharacterId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_CampaignNotFound(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:  "nonexistent",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterState_RequiresCharacterMutationPolicy(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})

	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 10, Hope: 3, Stress: 1},
		},
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestPatchCharacterState_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(context.Background(), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestPatchCharacterState_StateNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.PatchCharacterState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "nonexistent",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestPatchCharacterState_InvalidHope(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
	})
	_, err := svc.PatchCharacterState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 15, Hope: 7, Stress: 1}}, // Hope max is 6
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_InvalidStress(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
	})
	_, err := svc.PatchCharacterState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 15, Hope: 3, Stress: 10}}, // Stress max is 6
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_InvalidHp(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
	})
	_, err := svc.PatchCharacterState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 25, Hope: 3, Stress: 1}}, // Hp max is 18
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestPatchCharacterState_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
	})

	_, err := svc.PatchCharacterState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 10, Hope: 5, Stress: 1},
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestPatchCharacterState_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	hp := 10
	hope := 5
	stress := 3

	payloadJSON, err := json.Marshal(daggerheart.CharacterStatePatchedPayload{
		CharacterID: "ch1",
		HP:          &hp,
		Hope:        &hope,
		Stress:      &stress,
	})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      "ch1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	resp, err := svc.PatchCharacterState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 10, Hope: 5, Stress: 3}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterState returned error: %v", err)
	}
	if resp.State == nil {
		t.Fatal("PatchCharacterState response has nil state")
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHope() != 5 {
		t.Errorf("State Hope = %d, want %d", dh.GetHope(), 5)
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetStress() != 3 {
		t.Errorf("State Stress = %d, want %d", dh.GetStress(), 3)
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHp() != 10 {
		t.Errorf("State Hp = %d, want %d", dh.GetHp(), 10)
	}

	// Verify persisted
	dhStored, _ := dhStore.GetDaggerheartCharacterState(context.Background(), "c1", "ch1")
	if dhStored.Hope != 5 {
		t.Errorf("Stored Hope = %d, want %d", dhStored.Hope, 5)
	}
	if dhStored.Hp != 10 {
		t.Errorf("Stored Hp = %d, want %d", dhStored.Hp, 10)
	}

	if len(eventStore.Events["c1"]) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.Events["c1"]))
	}
	if eventStore.Events["c1"][0].Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, "sys.daggerheart.character_state_patched")
	}
}

func TestPatchCharacterState_SetToZero(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 5, Stress: 3},
	}
	dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	hp := 0
	hope := 0
	stress := 0

	payloadJSON, err := json.Marshal(daggerheart.CharacterStatePatchedPayload{
		CharacterID: "ch1",
		HP:          &hp,
		Hope:        &hope,
		Stress:      &stress,
	})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      "ch1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	resp, err := svc.PatchCharacterState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:       "c1",
		CharacterId:      "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 0, Hope: 0, Stress: 0}},
	})
	if err != nil {
		t.Fatalf("PatchCharacterState returned error: %v", err)
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHope() != 0 {
		t.Errorf("State Hope = %d, want 0", dh.GetHope())
	}
	if dh := resp.State.GetDaggerheart(); dh == nil || dh.GetHp() != 0 {
		t.Errorf("State Hp = %d, want 0", dh.GetHp())
	}

	// Verify persisted
	dhStored, _ := dhStore.GetDaggerheartCharacterState(context.Background(), "c1", "ch1")
	if dhStored.Hope != 0 {
		t.Errorf("Stored Hope = %d, want 0", dhStored.Hope)
	}
	if dhStored.Hp != 0 {
		t.Errorf("Stored Hp = %d, want 0", dhStored.Hp)
	}
}

func TestUpdateSnapshotState_NilRequest(t *testing.T) {
	svc := NewService(emptyDeps())
	_, err := svc.UpdateSnapshotState(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateSnapshotState_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateSnapshotState_CampaignNotFound(t *testing.T) {
	svc := NewService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "nonexistent",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestUpdateSnapshotState_RequiresManageSessionsPolicy(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: gametest.NewFakeParticipantStore()},
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})

	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 2},
		},
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestUpdateSnapshotState_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.UpdateSnapshotState(context.Background(), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestUpdateSnapshotState_NegativeGmFear(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: gametest.NewFakeDaggerheartStore(),
	})
	_, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: -1},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateSnapshotState_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
	})

	_, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 7},
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestUpdateSnapshotState_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	payloadJSON, err := json.Marshal(daggerheart.GMFearChangedPayload{Value: 7})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "campaign",
				EntityID:      "c1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	resp, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 7},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 7 {
		t.Errorf("Response GmFear = %d, want %d", dh.GetGmFear(), 7)
	}

	// Verify persisted
	stored, err := dhStore.GetDaggerheartSnapshot(context.Background(), "c1")
	if err != nil {
		t.Fatalf("DaggerheartSnapshot not persisted: %v", err)
	}
	if stored.GMFear != 7 {
		t.Errorf("Stored GMFear = %d, want %d", stored.GMFear, 7)
	}

	if len(eventStore.Events["c1"]) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.Events["c1"]))
	}
	if eventStore.Events["c1"][0].Type != event.Type("sys.daggerheart.gm_fear_changed") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, "sys.daggerheart.gm_fear_changed")
	}
}

func TestUpdateSnapshotState_UpdateExisting(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 3}

	payloadJSON, err := json.Marshal(daggerheart.GMFearChangedPayload{Value: 10})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "campaign",
				EntityID:      "c1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	resp, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 10},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 10 {
		t.Errorf("Response GmFear = %d, want %d", dh.GetGmFear(), 10)
	}

	// Verify updated
	stored, _ := dhStore.GetDaggerheartSnapshot(context.Background(), "c1")
	if stored.GMFear != 10 {
		t.Errorf("Stored GMFear = %d, want %d", stored.GMFear, 10)
	}
}

func TestUpdateSnapshotState_SetToZero(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.Snapshots["c1"] = projectionstore.DaggerheartSnapshot{CampaignID: "c1", GMFear: 5}

	payloadJSON, err := json.Marshal(daggerheart.GMFearChangedPayload{Value: 0})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "campaign",
				EntityID:      "c1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	resp, err := svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 0},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if dh := resp.Snapshot.GetDaggerheart(); dh == nil || dh.GetGmFear() != 0 {
		t.Errorf("Response GmFear = %d, want 0", dh.GetGmFear())
	}
}

func TestUpdateSnapshotState_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	payloadJSON, err := json.Marshal(daggerheart.GMFearChangedPayload{Value: 5})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_fear.set"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "campaign",
				EntityID:      "c1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	_, err = svc.UpdateSnapshotState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.UpdateSnapshotStateRequest{
		CampaignId: "c1",
		SystemSnapshotUpdate: &statev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{GmFear: 5},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSnapshotState returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.gm_fear.set") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.gm_fear.set")
	}
}

func TestPatchCharacterState_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	dhStore.States["c1"] = map[string]projectionstore.DaggerheartCharacterState{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", Hp: 15, Hope: 3, Stress: 1},
	}
	dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
		"ch1": {CampaignID: "c1", CharacterID: "ch1", HpMax: 18, StressMax: 6},
	}

	hp := 10
	hope := 5
	stress := 1

	payloadJSON, err := json.Marshal(daggerheart.CharacterStatePatchedPayload{
		CharacterID: "ch1",
		HP:          &hp,
		Hope:        &hope,
		Stress:      &stress,
	})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "c1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      "ch1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc := NewService(Deps{
		Campaign:    campaignStore,
		Daggerheart: dhStore,
		Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		Applier:     testApplier(dhStore),
	})

	_, err = svc.PatchCharacterState(gametest.ContextWithAdminOverride("snapshot-test"), &statev1.PatchCharacterStateRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStatePatch: &statev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{Hp: 10, Hope: 5, Stress: 1},
		},
	})
	if err != nil {
		t.Fatalf("PatchCharacterState returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
}
