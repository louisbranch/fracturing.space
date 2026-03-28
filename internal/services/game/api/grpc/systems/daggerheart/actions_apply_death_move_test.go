package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
)

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
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.States["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		LifeState:   mechanics.LifeStateDead,
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
	dhStore.States["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		Hope:        2,
		HopeMax:     mechanics.HopeMax,
		Stress:      1,
		LifeState:   daggerheartstate.LifeStateAlive,
	}
	profile := dhStore.Profiles["camp-1:char-1"]
	move := daggerheart.DeathMoveAvoidDeath
	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = daggerheartprofile.PCHpMax
	}
	stressMax := profile.StressMax
	if stressMax < 0 {
		stressMax = 0
	}
	hopeMax := mechanics.HopeMax
	level := profile.Level
	if level == 0 {
		level = daggerheartprofile.PCLevelDefault
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
	payload := daggerheartpayload.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		LifeState:   &result.LifeState,
		HP:          &result.HPAfter,
		Hope:        &result.HopeAfter,
		HopeMax:     &result.HopeMaxAfter,
		Stress:      &result.StressAfter,
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
				Timestamp:     testTimestamp,
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
	svc.stores.Write.Executor = serviceDomain
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
	now := testTimestamp

	state := projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          0,
		Hope:        2,
		HopeMax:     mechanics.HopeMax,
		Stress:      1,
		LifeState:   daggerheartstate.LifeStateAlive,
	}
	dhStore.States["camp-1:char-1"] = state
	profile := dhStore.Profiles["camp-1:char-1"]
	move := daggerheart.DeathMoveAvoidDeath

	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = daggerheartprofile.PCHpMax
	}
	stressMax := profile.StressMax
	if stressMax < 0 {
		stressMax = 0
	}
	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = mechanics.HopeMax
	}
	level := profile.Level
	if level == 0 {
		level = daggerheartprofile.PCLevelDefault
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

	payload := daggerheartpayload.CharacterStatePatchedPayload{
		CharacterID: "char-1",
		LifeState:   &result.LifeState,
		HP:          &result.HPAfter,
		Hope:        &result.HopeAfter,
		HopeMax:     &result.HopeMaxAfter,
		Stress:      &result.StressAfter,
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
	svc.stores.Write.Executor = domain

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
