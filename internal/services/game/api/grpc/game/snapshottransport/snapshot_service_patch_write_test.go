package snapshottransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

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

	payloadJSON, err := json.Marshal(daggerheartpayload.CharacterStatePatchedPayload{
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
		Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
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

	payloadJSON, err := json.Marshal(daggerheartpayload.CharacterStatePatchedPayload{
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
		Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
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

	dhStored, _ := dhStore.GetDaggerheartCharacterState(context.Background(), "c1", "ch1")
	if dhStored.Hope != 0 {
		t.Errorf("Stored Hope = %d, want 0", dhStored.Hope)
	}
	if dhStored.Hp != 0 {
		t.Errorf("Stored Hp = %d, want 0", dhStored.Hp)
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

	payloadJSON, err := json.Marshal(daggerheartpayload.CharacterStatePatchedPayload{
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
		Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
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
