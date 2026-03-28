package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

// --- ApplyGmMove tests ---

func TestApplyGmMove_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "c1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestApplyGmMove_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		SessionId: "sess-1",
		FearSpent: 1,
		SpendTarget: &pb.DaggerheartApplyGmMoveRequest_DirectMove{
			DirectMove: &pb.DaggerheartDirectGmMoveTarget{
				Kind:  pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE,
				Shape: pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT,
			},
		},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		FearSpent:  1,
		SpendTarget: &pb.DaggerheartApplyGmMoveRequest_DirectMove{
			DirectMove: &pb.DaggerheartDirectGmMoveTarget{
				Kind:  pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE,
				Shape: pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT,
			},
		},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_MissingKind(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		FearSpent:  1,
		SpendTarget: &pb.DaggerheartApplyGmMoveRequest_DirectMove{
			DirectMove: &pb.DaggerheartDirectGmMoveTarget{
				Shape: pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT,
			},
		},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_NegativeFearSpent(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		FearSpent:  -1,
		SpendTarget: &pb.DaggerheartApplyGmMoveRequest_DirectMove{
			DirectMove: &pb.DaggerheartDirectGmMoveTarget{
				Kind:  pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE,
				Shape: pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT,
			},
		},
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestApplyGmMove_SuccessAdditionalMove(t *testing.T) {
	svc := newActionTestService()
	domain := &fakeDomainEngine{}
	svc.stores.Write.Executor = domain
	ctx := context.Background()
	resp, err := svc.ApplyGmMove(ctx, &pb.DaggerheartApplyGmMoveRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		FearSpent:   1,
		SpendTarget: directAdditionalMoveRequest("camp-1", "sess-1", 1).GetSpendTarget(),
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
	if resp != nil {
		t.Fatal("expected nil response when fear is unavailable")
	}
	if domain.calls != 0 {
		t.Fatalf("expected no domain calls, got %d", domain.calls)
	}
}

func TestApplyGmMove_WithFearSpent(t *testing.T) {
	svc := newActionTestService()
	// Pre-populate GM fear in snapshot
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Snapshots["camp-1"] = projectionstore.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 3}
	eventStore := svc.stores.Event.(*fakeEventStore)
	gmMovePayloadJSON, err := json.Marshal(daggerheartpayload.GMMoveAppliedPayload{
		Target: daggerheartpayload.GMMoveTarget{
			Type:  rules.GMMoveTargetTypeDirectMove,
			Kind:  rules.GMMoveKindAdditionalMove,
			Shape: rules.GMMoveShapeShiftEnvironment,
		},
		FearSpent: 1,
	})
	if err != nil {
		t.Fatalf("encode gm move payload: %v", err)
	}
	gmFearPayloadJSON, err := json.Marshal(daggerheartpayload.GMFearChangedPayload{Value: 2, Reason: "gm_move"})
	if err != nil {
		t.Fatalf("encode gm fear payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_move.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_move_applied"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-gm-move-fear",
				EntityType:    "session",
				EntityID:      "sess-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   gmMovePayloadJSON,
			}, event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-gm-move-fear",
				EntityType:    "campaign",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   gmFearPayloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	ctx := context.Background()
	resp, err := svc.ApplyGmMove(ctx, &pb.DaggerheartApplyGmMoveRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		FearSpent:   1,
		SpendTarget: directAdditionalMoveRequest("camp-1", "sess-1", 1).GetSpendTarget(),
	})
	if err != nil {
		t.Fatalf("ApplyGmMove returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.GetGmFearBefore() != 3 {
		t.Fatalf("expected gm fear before = 3, got %d", resp.GetGmFearBefore())
	}
	if resp.GetGmFearAfter() != 2 {
		t.Fatalf("expected gm fear after = 2, got %d", resp.GetGmFearAfter())
	}
	if len(eventStore.Events["camp-1"]) != 2 {
		t.Fatalf("expected 2 events, got %d", len(eventStore.Events["camp-1"]))
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("sys.daggerheart.gm_move_applied") {
		t.Fatalf("first event type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("sys.daggerheart.gm_move_applied"))
	}
	if eventStore.Events["camp-1"][1].Type != event.Type("sys.daggerheart.gm_fear_changed") {
		t.Fatalf("second event type = %s, want %s", eventStore.Events["camp-1"][1].Type, event.Type("sys.daggerheart.gm_fear_changed"))
	}
}

func TestApplyGmMove_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Snapshots["camp-1"] = projectionstore.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 2}
	now := testTimestamp

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.gm_move.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_move_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "session",
				EntityID:      "sess-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   []byte(`{"target":{"type":"direct_move","kind":"additional_move","shape":"shift_environment"},"fear_spent":1}`),
			}, event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.gm_fear_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "campaign",
				EntityID:      "camp-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   []byte(`{"after":1,"reason":"gm_move"}`),
			}),
		},
	}}

	svc.stores.Write.Executor = domain

	_, err := svc.ApplyGmMove(context.Background(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		FearSpent:   1,
		SpendTarget: directAdditionalMoveRequest("camp-1", "sess-1", 1).GetSpendTarget(),
	})
	if err != nil {
		t.Fatalf("ApplyGmMove returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.gm_move.apply") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.gm_move.apply")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		Target struct {
			Type  string `json:"type"`
			Kind  string `json:"kind"`
			Shape string `json:"shape"`
		} `json:"target"`
		FearSpent int `json:"fear_spent"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode gm move command payload: %v", err)
	}
	if got.Target.Kind != "additional_move" {
		t.Fatalf("command kind = %q, want additional_move", got.Target.Kind)
	}
	if got.Target.Shape != "shift_environment" {
		t.Fatalf("command shape = %q, want shift_environment", got.Target.Shape)
	}
	if got.FearSpent != 1 {
		t.Fatalf("command fear_spent = %d, want 1", got.FearSpent)
	}
	if got := len(eventStore.Events["camp-1"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("sys.daggerheart.gm_move_applied") {
		t.Fatalf("first event type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("sys.daggerheart.gm_move_applied"))
	}
	if eventStore.Events["camp-1"][1].Type != event.Type("sys.daggerheart.gm_fear_changed") {
		t.Fatalf("second event type = %s, want %s", eventStore.Events["camp-1"][1].Type, event.Type("sys.daggerheart.gm_fear_changed"))
	}
}
