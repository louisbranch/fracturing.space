package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

// --- ApplyDowntimeMove tests ---

func TestApplyDowntimeMove_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyDowntimeMove(context.Background(), &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDowntimeMove_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_CampaignNotFound(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "nonexistent", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDowntimeMove_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyDowntimeMove(context.Background(), &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_MissingMove(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_UnspecifiedMove(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDowntimeMove_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS,
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDowntimeMove_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	current := dhStore.States["camp-1:char-1"]
	profile := dhStore.Profiles["camp-1:char-1"]
	state := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          current.Hp,
		HPMax:       profile.HpMax,
		Hope:        current.Hope,
		HopeMax:     current.HopeMax,
		Stress:      current.Stress,
		StressMax:   profile.StressMax,
		Armor:       current.Armor,
		ArmorMax:    profile.ArmorMax,
		LifeState:   current.LifeState,
	})
	move, err := daggerheartDowntimeMoveFromProto(pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS)
	if err != nil {
		t.Fatalf("map downtime move: %v", err)
	}
	result := daggerheart.ApplyDowntimeMove(state, move, daggerheart.DowntimeOptions{})
	moveName := daggerheartDowntimeMoveToString(move)
	payload := daggerheart.DowntimeMoveAppliedPayload{
		CharacterID:  "char-1",
		Move:         moveName,
		HopeBefore:   &result.HopeBefore,
		HopeAfter:    &result.HopeAfter,
		StressBefore: &result.StressBefore,
		StressAfter:  &result.StressAfter,
		ArmorBefore:  &result.ArmorBefore,
		ArmorAfter:   &result.ArmorAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode downtime move payload: %v", err)
	}

	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.downtime_move.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.downtime_move_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-downtime-success",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-downtime-success")
	resp, err := svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDowntimeMove returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.Stress != 0 {
		t.Fatalf("stress = %d, want 0", resp.State.Stress)
	}
}

func TestApplyDowntimeMove_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	current := dhStore.States["camp-1:char-1"]
	profile := dhStore.Profiles["camp-1:char-1"]
	state := daggerheart.NewCharacterState(daggerheart.CharacterStateConfig{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HP:          current.Hp,
		HPMax:       profile.HpMax,
		Hope:        current.Hope,
		HopeMax:     current.HopeMax,
		Stress:      current.Stress,
		StressMax:   profile.StressMax,
		Armor:       current.Armor,
		ArmorMax:    profile.ArmorMax,
		LifeState:   current.LifeState,
	})
	move, err := daggerheartDowntimeMoveFromProto(pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS)
	if err != nil {
		t.Fatalf("map downtime move: %v", err)
	}
	result := daggerheart.ApplyDowntimeMove(state, move, daggerheart.DowntimeOptions{})
	moveName := daggerheartDowntimeMoveToString(move)
	payload := daggerheart.DowntimeMoveAppliedPayload{
		CharacterID:  "char-1",
		Move:         moveName,
		HopeBefore:   &result.HopeBefore,
		HopeAfter:    &result.HopeAfter,
		StressBefore: &result.StressBefore,
		StressAfter:  &result.StressAfter,
		ArmorBefore:  &result.ArmorBefore,
		ArmorAfter:   &result.ArmorAfter,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode downtime move payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.downtime_move.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.downtime_move_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-downtime-move",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-downtime-move")
	_, err = svc.ApplyDowntimeMove(ctx, &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDowntimeMove returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.downtime_move.apply") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.downtime_move.apply")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		CharacterID  string `json:"character_id"`
		Move         string `json:"move"`
		StressBefore *int   `json:"stress_before"`
		StressAfter  *int   `json:"stress_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode downtime move command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.Move != moveName {
		t.Fatalf("command move = %s, want %s", got.Move, moveName)
	}
	if got.StressBefore == nil || *got.StressBefore != result.StressBefore {
		t.Fatalf("command stress before = %v, want %d", got.StressBefore, result.StressBefore)
	}
	if got.StressAfter == nil || *got.StressAfter != result.StressAfter {
		t.Fatalf("command stress after = %v, want %d", got.StressAfter, result.StressAfter)
	}
}
