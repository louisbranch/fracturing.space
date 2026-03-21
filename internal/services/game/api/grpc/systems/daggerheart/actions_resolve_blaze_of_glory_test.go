package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestResolveBlazeOfGlory_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ResolveBlazeOfGlory(context.Background(), &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestResolveBlazeOfGlory_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveBlazeOfGlory_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ResolveBlazeOfGlory(context.Background(), &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResolveBlazeOfGlory_CharacterAlreadyDead(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		LifeState:   daggerheart.LifeStateDead,
	}
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestResolveBlazeOfGlory_NotInBlazeState(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestResolveBlazeOfGlory_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   daggerheart.LifeStateBlazeOfGlory,
	}
	charStore := svc.stores.Character.(*fakeCharacterStore)
	charStore.Characters["camp-1:char-1"] = storage.CharacterRecord{
		ID:         "char-1",
		CampaignID: "camp-1",
		Name:       "Hero",
		Kind:       character.KindPC,
	}
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		LifeState: func() *string {
			l := daggerheart.LifeStateDead
			return &l
		}(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode blaze of glory payload: %v", err)
	}
	eventStore := svc.stores.Event.(*fakeEventStore)
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-blaze-success",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("character.deleted"),
				Timestamp:   testTimestamp,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-blaze-success",
				EntityType:  "character",
				EntityID:    "char-1",
				PayloadJSON: []byte(`{"character_id":"char-1","reason":"blaze_of_glory"}`),
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-blaze-success")
	resp, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("ResolveBlazeOfGlory returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.Result.LifeState != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD {
		t.Fatalf("life_state = %v, want DEAD", resp.Result.LifeState)
	}
}

func TestResolveBlazeOfGlory_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   daggerheart.LifeStateBlazeOfGlory,
	}
	charStore := svc.stores.Character.(*fakeCharacterStore)
	charStore.Characters["camp-1:char-1"] = storage.CharacterRecord{
		ID:         "char-1",
		CampaignID: "camp-1",
		Name:       "Hero",
		Kind:       character.KindPC,
	}

	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		LifeState: func() *string {
			l := daggerheart.LifeStateDead
			return &l
		}(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode blaze of glory payload: %v", err)
	}

	eventStore := svc.stores.Event.(*fakeEventStore)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-blaze",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
		command.Type("character.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("character.deleted"),
				Timestamp:   testTimestamp,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-blaze",
				EntityType:  "character",
				EntityID:    "char-1",
				PayloadJSON: []byte(`{"character_id":"char-1","reason":"blaze_of_glory"}`),
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-blaze")
	_, err = svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("ResolveBlazeOfGlory returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("command[0] type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[1].Type != command.Type("character.delete") {
		t.Fatalf("command[1] type = %s, want %s", domain.commands[1].Type, "character.delete")
	}
	if got := len(eventStore.Events["camp-1"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("event[0] type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("sys.daggerheart.character_state_patched"))
	}
	if eventStore.Events["camp-1"][1].Type != event.Type("character.deleted") {
		t.Fatalf("event[1] type = %s, want %s", eventStore.Events["camp-1"][1].Type, event.Type("character.deleted"))
	}
}
