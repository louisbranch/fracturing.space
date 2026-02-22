package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func TestApplyAttackOutcome_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	svc.stores.Domain = nil

	rollPayload := action.RollResolvePayload{
		RequestID: "req-atk-outcome-required",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-atk-outcome-required",
		ActorType:   event.ActorTypeSystem,
		EntityID:    "req-atk-outcome-required",
		EntityType:  "roll",
		PayloadJSON: rollPayloadJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-atk-outcome-required",
	)
	_, err = svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
		Targets:   []string{"char-2"},
	})
	if err != nil {
		t.Fatalf("ApplyAttackOutcome returned error: %v", err)
	}
}

func TestApplyAttackOutcome_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-atk-outcome-legacy",
		RollSeq:   1,
		Results:   map[string]any{"d20": 20},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
		},
	}
	rollJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-atk-outcome-legacy",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-atk-outcome-legacy",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	svc.stores.Domain = nil

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-atk-outcome-legacy",
	)
	resp, err := svc.ApplyAttackOutcome(ctx, &pb.DaggerheartApplyAttackOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
		Targets:   []string{"char-2"},
	})
	if err != nil {
		t.Fatalf("ApplyAttackOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("expected character_id char-1, got %s", resp.CharacterId)
	}
	if len(resp.Targets) != 1 || resp.Targets[0] != "char-2" {
		t.Fatalf("expected targets [char-2], got %v", resp.Targets)
	}
	if resp.Result.GetOutcome() != pb.Outcome_SUCCESS_WITH_HOPE {
		t.Fatalf("expected outcome SUCCESS_WITH_HOPE, got %s", resp.Result.GetOutcome())
	}
	if !resp.Result.GetSuccess() {
		t.Fatal("expected attack outcome success")
	}
}

// --- ApplyAdversaryAttackOutcome tests ---

func TestApplyAdversaryAttackOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryAttackOutcome(context.Background(), &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId: "s1",
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

func TestApplyAdversaryAttackOutcome_RequiresDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-atk-outcome-required",
		RollSeq:   1,
		Results:   map[string]any{"rolls": []int{4}, "roll": 4, "modifier": 0, "total": 4, "advantage": 0, "disadvantage": 0},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "adv-1",
			"adversary_id": "adv-1",
			"roll_kind":    "adversary_roll",
			"roll":         4,
			"modifier":     0,
			"total":        4,
			"advantage":    0,
			"disadvantage": 0,
		},
	}
	rollPayloadJSON, err := json.Marshal(rollPayload)
	if err != nil {
		t.Fatalf("encode roll payload: %v", err)
	}
	outcomeJSON, err := json.Marshal(action.OutcomeApplyPayload{
		RequestID:      "req-adv-attack-1",
		RollSeq:        1,
		Targets:        []string{"char-1"},
		AppliedChanges: []action.OutcomeAppliedChange{},
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
				RequestID:   "req-adv-atk-outcome-required",
				EntityType:  "adversary",
				EntityID:    "adv-1",
				PayloadJSON: rollPayloadJSON,
			}),
		},
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.outcome_applied"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-attack-1",
				EntityType:    "outcome",
				EntityID:      "req-adv-attack-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   outcomeJSON,
			}),
		},
	}}

	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-adv-atk-outcome-required")
	rollResp, err := svc.SessionAdversaryAttackRoll(rollCtx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}
	noDomainSvc := &DaggerheartService{stores: svc.stores, seedFunc: svc.seedFunc}
	noDomainSvc.stores.Domain = nil

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-adv-atk-outcome-required",
	)
	resp, err := noDomainSvc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  "sess-1",
		RollSeq:    rollResp.RollSeq,
		Targets:    []string{"char-1"},
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryAttackOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
}

func TestApplyAdversaryAttackOutcome_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-atk-outcome-legacy",
		RollSeq:   1,
		Results:   map[string]any{"rolls": []int{4}, "roll": 4, "modifier": 0, "total": 4, "advantage": 0, "disadvantage": 0},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "adv-1",
			"adversary_id": "adv-1",
			"roll_kind":    "adversary_roll",
			"roll":         4,
			"modifier":     0,
			"total":        4,
			"advantage":    0,
			"disadvantage": 0,
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
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
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

	svc.stores.Domain = domain

	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-adv-atk-outcome-legacy")
	rollResp, err := svc.SessionAdversaryAttackRoll(rollCtx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-adv-atk-outcome-legacy",
	)
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
