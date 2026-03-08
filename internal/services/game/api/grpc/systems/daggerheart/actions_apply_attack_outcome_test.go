package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

// --- ApplyRollOutcome tests ---
// --- ApplyAttackOutcome tests ---

func TestApplyAttackOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "s1",
		RollSeq:   1,
		Targets:   []string{"char-1"},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAttackOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1, Targets: []string{"adv-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAttackOutcome(context.Background(), &pb.DaggerheartApplyAttackOutcomeRequest{
		RollSeq: 1, Targets: []string{"adv-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", Targets: []string{"adv-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAttackOutcome_MissingTargets(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- ApplyAdversaryAttackOutcome tests ---

func TestApplyAdversaryAttackOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "s1",
		RollSeq:   1,
		Targets:   []string{"char-1"},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryAttackOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1, Targets: []string{"char-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		RollSeq: 1, Targets: []string{"char-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", Targets: []string{"char-1"},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_MissingTargets(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryAttackOutcome_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-atk-outcome-legacy",
		RollSeq:   1,
		Results:   map[string]any{"rolls": []int{4}, sdKeyRoll: 4, sdKeyModifier: 0, sdKeyTotal: 4, "advantage": 0, "disadvantage": 0},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			sdKeyCharacterID: "adv-1",
			sdKeyAdversaryID: "adv-1",
			sdKeyRollKind:    "adversary_roll",
			sdKeyRoll:        4,
			sdKeyModifier:    0,
			sdKeyTotal:       4,
			"advantage":      0,
			"disadvantage":   0,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.roll_resolved"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-atk-outcome-legacy",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   rollPayloadJSON,
			}),
		},
	}}

	svc.stores.Write.Executor = domain

	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-adv-atk-outcome-legacy")
	rollResp, err := svc.SessionAdversaryAttackRoll(rollCtx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}

	ctx := testSessionCtx("camp-1", "sess-1", "req-adv-atk-outcome-legacy")
	resp, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  "sess-1",
		RollSeq:    rollResp.RollSeq,
		Targets:    []string{"char-1"},
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryAttackOutcome returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("expected adversary adv-1, got %s", resp.AdversaryId)
	}
}
