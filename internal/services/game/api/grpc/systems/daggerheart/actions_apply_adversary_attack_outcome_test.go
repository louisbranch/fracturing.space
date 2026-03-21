package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func TestApplyAdversaryAttackOutcome_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-atk-outcome-legacy",
		RollSeq:   1,
		Results:   map[string]any{"rolls": []int{4}, workflowtransport.KeyRoll: 4, workflowtransport.KeyModifier: 0, workflowtransport.KeyTotal: 4, "advantage": 0, "disadvantage": 0},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "adv-1",
			workflowtransport.KeyAdversaryID: "adv-1",
			workflowtransport.KeyRollKind:    "adversary_roll",
			workflowtransport.KeyRoll:        4,
			workflowtransport.KeyModifier:    0,
			workflowtransport.KeyTotal:       4,
			"advantage":                      0,
			"disadvantage":                   0,
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

func TestApplyAdversaryAttackOutcome_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	rollPayload := action.RollResolvePayload{
		RequestID: "req-adv-atk-outcome-1",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":                       []int{3},
			workflowtransport.KeyRoll:     3,
			workflowtransport.KeyModifier: 0,
			workflowtransport.KeyTotal:    3,
			"advantage":                   0,
			"disadvantage":                0,
		},
		Outcome: pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "adv-1",
			workflowtransport.KeyAdversaryID: "adv-1",
			workflowtransport.KeyRollKind:    "adversary_roll",
			workflowtransport.KeyRoll:        3,
			workflowtransport.KeyModifier:    0,
			workflowtransport.KeyTotal:       3,
			"advantage":                      0,
			"disadvantage":                   0,
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
		RequestID:   "req-adv-atk-outcome-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-adv-atk-outcome-1",
		PayloadJSON: rollPayloadJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		workflowtransport.WithCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-adv-atk-outcome-1",
	)
	resp, err := svc.ApplyAdversaryAttackOutcome(ctx, &pb.DaggerheartApplyAdversaryAttackOutcomeRequest{
		SessionId:  "sess-1",
		RollSeq:    rollEvent.Seq,
		Targets:    []string{"char-1"},
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryAttackOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("expected adversary adv-1, got %s", resp.AdversaryId)
	}
	if resp.Result.GetSuccess() {
		t.Fatal("expected adversary attack outcome failure")
	}
	if resp.Result.GetRoll() != 3 {
		t.Fatalf("expected roll=3, got %d", resp.Result.GetRoll())
	}
	if resp.Result.GetTotal() != 3 {
		t.Fatalf("expected total=3, got %d", resp.Result.GetTotal())
	}
	if resp.Result.GetDifficulty() != 10 {
		t.Fatalf("expected difficulty=10, got %d", resp.Result.GetDifficulty())
	}
}
