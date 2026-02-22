package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

// --- ApplyDeathMove tests ---

func TestApplyDeathMove_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyDeathMove(context.Background(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDeathMove_MissingSeedFunc(t *testing.T) {
	svc := &DaggerheartService{
		stores: Stores{
			Campaign:    newFakeCampaignStore(),
			Daggerheart: newFakeDaggerheartStore(),
			Event:       newFakeActionEventStore(),
		},
	}
	_, err := svc.ApplyDeathMove(context.Background(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDeathMove_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyDeathMove(context.Background(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_UnspecifiedMove(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		Hope:        2,
		HopeMax:     daggerheart.HopeMax,
		Stress:      1,
		LifeState:   daggerheart.LifeStateAlive,
	}
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDeathMove_HpClearOnNonRiskItAll(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	hpClear := int32(1)
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
		HpClear:     &hpClear,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDeathMove_HpNotZero(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyDeathMove_AlreadyDead(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	// Set up state with hp=0 and life_state=dead
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   daggerheart.LifeStateDead,
	}
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyDeathMove_AvoidDeath_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		Hope:        2,
		HopeMax:     daggerheart.HopeMax,
		Stress:      1,
		LifeState:   daggerheart.LifeStateAlive,
	}
	profile := dhStore.Profiles["camp-1:char-1"]
	move, err := daggerheartDeathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH)
	if err != nil {
		t.Fatalf("map death move: %v", err)
	}
	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = daggerheart.PCHpMax
	}
	stressMax := profile.StressMax
	if stressMax < 0 {
		stressMax = 0
	}
	hopeMax := daggerheart.HopeMax
	level := profile.Level
	if level == 0 {
		level = daggerheart.PCLevelDefault
	}
	result, err := daggerheart.ResolveDeathMove(daggerheart.DeathMoveInput{
		Move:      move,
		Level:     level,
		HP:        0,
		HPMax:     hpMax,
		Hope:      2,
		HopeMax:   hopeMax,
		Stress:    1,
		StressMax: stressMax,
		Seed:      42,
	})
	if err != nil {
		t.Fatalf("resolve death move: %v", err)
	}
	lifeStateBefore := daggerheart.LifeStateAlive
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     "char-1",
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &result.LifeState,
		HPBefore:        &result.HPBefore,
		HPAfter:         &result.HPAfter,
		HopeBefore:      &result.HopeBefore,
		HopeAfter:       &result.HopeAfter,
		HopeMaxBefore:   &result.HopeMaxBefore,
		HopeMaxAfter:    &result.HopeMaxAfter,
		StressBefore:    &result.StressBefore,
		StressAfter:     &result.StressAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode death move payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: svc.stores.Event, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-death-success",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-death-success")
	resp, err := svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	if err != nil {
		t.Fatalf("ApplyDeathMove returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
}

func TestApplyDeathMove_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	state := storage.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		Hope:        2,
		HopeMax:     daggerheart.HopeMax,
		Stress:      1,
		LifeState:   daggerheart.LifeStateAlive,
	}
	dhStore.States["camp-1:char-1"] = state
	profile := dhStore.Profiles["camp-1:char-1"]
	move, err := daggerheartDeathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH)
	if err != nil {
		t.Fatalf("map death move: %v", err)
	}

	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = daggerheart.PCHpMax
	}
	stressMax := profile.StressMax
	if stressMax < 0 {
		stressMax = 0
	}
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}
	level := profile.Level
	if level == 0 {
		level = daggerheart.PCLevelDefault
	}

	result, err := daggerheart.ResolveDeathMove(daggerheart.DeathMoveInput{
		Move:      move,
		Level:     level,
		HP:        state.Hp,
		HPMax:     hpMax,
		Hope:      state.Hope,
		HopeMax:   hopeMax,
		Stress:    state.Stress,
		StressMax: stressMax,
		Seed:      42,
	})
	if err != nil {
		t.Fatalf("resolve death move: %v", err)
	}

	lifeStateBefore := state.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheart.LifeStateAlive
	}
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     "char-1",
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &result.LifeState,
		HPBefore:        &result.HPBefore,
		HPAfter:         &result.HPAfter,
		HopeBefore:      &result.HopeBefore,
		HopeAfter:       &result.HopeAfter,
		HopeMaxBefore:   &result.HopeMaxBefore,
		HopeMaxAfter:    &result.HopeMaxAfter,
		StressBefore:    &result.StressBefore,
		StressAfter:     &result.StressAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode death move payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-death-move",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-death-move")
	_, err = svc.ApplyDeathMove(ctx, &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH,
	})
	if err != nil {
		t.Fatalf("ApplyDeathMove returned error: %v", err)
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
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		CharacterID    string `json:"character_id"`
		LifeStateAfter string `json:"life_state_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode death move command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.LifeStateAfter == "" {
		t.Fatal("expected life_state_after in command payload")
	}
}

// --- ResolveBlazeOfGlory tests ---

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

func TestResolveBlazeOfGlory_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
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
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ResolveBlazeOfGlory(ctx, &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestResolveBlazeOfGlory_CharacterAlreadyDead(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
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
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
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
		LifeStateBefore: func() *string {
			l := daggerheart.LifeStateBlazeOfGlory
			return &l
		}(),
		LifeStateAfter: func() *string {
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
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
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
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-blaze-success",
				EntityType:  "character",
				EntityID:    "char-1",
				PayloadJSON: []byte(`{"character_id":"char-1","reason":"blaze_of_glory"}`),
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
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
	dhStore.States["camp-1:char-1"] = storage.DaggerheartCharacterState{
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
		LifeStateBefore: func() *string {
			l := daggerheart.LifeStateBlazeOfGlory
			return &l
		}(),
		LifeStateAfter: func() *string {
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
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
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
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-blaze",
				EntityType:  "character",
				EntityID:    "char-1",
				PayloadJSON: []byte(`{"character_id":"char-1","reason":"blaze_of_glory"}`),
			}),
		},
	}}
	svc.stores.Domain = domain

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
