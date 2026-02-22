package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

// --- SessionReactionFlow tests ---

func TestSessionReactionFlow_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionReactionFlow_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_MissingTrait(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId: "camp-1", SessionId: "sess-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionReactionFlow_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-reaction-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 12},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_REACTION.String(),
			"hope_fear":    false,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}

	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-reaction-1",
		RollSeq:   1,
		Targets:   []string{"char-1"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-reaction-1",
				EntityType:  "roll",
				EntityID:    "req-reaction-1",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-reaction-1",
				EntityType:  "outcome",
				EntityID:    "req-reaction-1",
				PayloadJSON: outcomeJSON,
			}),
		},
	}}
	ctx := grpcmeta.WithRequestID(context.Background(), "req-reaction-1")
	resp, err := svc.SessionReactionFlow(ctx, &pb.SessionReactionFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionReactionFlow returned error: %v", err)
	}
	if resp.ActionRoll == nil {
		t.Fatal("expected action roll in response")
	}
	if resp.RollOutcome == nil {
		t.Fatal("expected roll outcome in response")
	}
	if resp.ReactionOutcome == nil {
		t.Fatal("expected reaction outcome in response")
	}
}

func TestSessionReactionFlow_ForwardsAdvantageDisadvantage(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-reaction-forward-adv",
		RollSeq:   1,
		Results: map[string]any{
			"d20": 16,
		},
		Outcome: pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_REACTION.String(),
			"hope_fear":    false,
			"advantage":    0,
			"disadvantage": 0,
			"outcome":      pb.Outcome_SUCCESS_WITH_HOPE.String(),
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}

	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID: "req-reaction-forward-adv",
		RollSeq:   1,
		Targets:   []string{"char-1"},
	})
	if err != nil {
		t.Fatalf("encode outcome payload: %v", err)
	}

	svc.stores.Domain = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.roll_resolved"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-reaction-forward-adv",
				EntityType:  "roll",
				EntityID:    "req-reaction-forward-adv",
				PayloadJSON: rollJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-reaction-forward-adv",
				EntityType:  "outcome",
				EntityID:    "req-reaction-forward-adv",
				PayloadJSON: outcomeJSON,
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(context.Background(), "req-reaction-forward-adv")
	reactionSeed := uint64(11)
	_, err = svc.SessionReactionFlow(ctx, &pb.SessionReactionFlowRequest{
		CampaignId:   "camp-1",
		SessionId:    "sess-1",
		CharacterId:  "char-1",
		Trait:        "agility",
		Difficulty:   10,
		Advantage:    2,
		Disadvantage: 1,
		ReactionRng: &commonv1.RngRequest{
			Seed:     &reactionSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("SessionReactionFlow returned error: %v", err)
	}

	if len(svc.stores.Domain.(*fakeDomainEngine).commands) == 0 {
		t.Fatal("expected domain commands")
	}

	var commandPayload action.RollResolvePayload
	rollCommandPayload := svc.stores.Domain.(*fakeDomainEngine).commands[0].PayloadJSON
	if err := json.Unmarshal(rollCommandPayload, &commandPayload); err != nil {
		t.Fatalf("decode action roll command payload: %v", err)
	}

	advRaw, ok := commandPayload.SystemData["advantage"]
	if !ok {
		t.Fatal("expected advantage in system_data")
	}
	disRaw, ok := commandPayload.SystemData["disadvantage"]
	if !ok {
		t.Fatal("expected disadvantage in system_data")
	}
	advantage, ok := advRaw.(float64)
	if !ok || int(advantage) != 2 {
		t.Fatalf("advantage in command payload = %v, want 2", advRaw)
	}
	disadvantage, ok := disRaw.(float64)
	if !ok || int(disadvantage) != 1 {
		t.Fatalf("disadvantage in command payload = %v, want 1", disRaw)
	}
}
