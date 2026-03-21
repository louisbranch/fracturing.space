package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestApplyRollOutcome_IdempotentWhenAlreadyAppliedEvenWithOpenGate(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Snapshots["camp-1"] = projectionstore.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     3,
	}

	svc.stores.SessionGate = &fakeOpenSessionGateStore{
		gate: storage.SessionGate{
			CampaignID: "camp-1",
			SessionID:  "sess-1",
			GateID:     "gate-open",
			GateType:   "gm_consequence",
			Reason:     "gm_consequence",
			Status:     session.GateStatusOpen,
			CreatedAt:  now,
		},
	}
	svc.stores.SessionSpotlight = &fakeSessionSpotlightStateStore{
		exists: true,
		spotlight: storage.SessionSpotlight{
			CampaignID:    "camp-1",
			SessionID:     "sess-1",
			SpotlightType: session.SpotlightTypeGM,
		},
	}

	rollEvent := fearRollEvent(t, "req-roll-duplicate").appendTo(eventStore)

	_, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now.Add(time.Second),
		Type:        event.Type("action.outcome_applied"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-duplicate",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "outcome",
		EntityID:    "req-roll-duplicate",
		PayloadJSON: []byte(`{"request_id":"req-roll-duplicate","roll_seq":1}`),
	})
	if err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	domain := svc.stores.Write.Executor.(*fakeDomainEngine)
	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-duplicate")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.RequiresComplication {
		t.Fatal("expected requires complication to be true")
	}
	if resp.Updated == nil || resp.Updated.GmFear == nil {
		t.Fatal("expected gm fear in idempotent response")
	}
	if got, want := int(resp.Updated.GetGmFear()), 3; got != want {
		t.Fatalf("gm fear = %d, want %d", got, want)
	}
	if domain.calls != 0 {
		t.Fatalf("expected no new domain commands for duplicate outcome, got %d", domain.calls)
	}
}

func TestApplyRollOutcome_AlreadyAppliedStillEnsuresComplicationGate(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := testTimestamp

	dhStore.Snapshots["camp-1"] = projectionstore.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     2,
	}

	rollEvent := fearRollEvent(t, "req-roll-gate-retry").appendTo(eventStore)

	_, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now.Add(time.Second),
		Type:        event.Type("action.outcome_applied"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-gate-retry",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "outcome",
		EntityID:    "req-roll-gate-retry",
		PayloadJSON: []byte(`{"request_id":"req-roll-gate-retry","roll_seq":1}`),
	})
	if err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	gatePayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "gm_consequence",
		Reason:   "gm_consequence",
		Metadata: map[string]any{"roll_seq": uint64(rollEvent.Seq), "request_id": "req-roll-gate-retry"},
	}
	gateJSON, err := json.Marshal(gatePayload)
	if err != nil {
		t.Fatalf("encode gate payload: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	spotlightJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		t.Fatalf("encode spotlight payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("session.gate_open"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("session.gate_opened"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-gate-retry",
				EntityType:  "session_gate",
				EntityID:    "gate-1",
				PayloadJSON: gateJSON,
			}),
		},
		command.Type("session.spotlight_set"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("session.spotlight_set"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-gate-retry",
				EntityType:  "session_spotlight",
				EntityID:    "sess-1",
				PayloadJSON: spotlightJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-gate-retry")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.RequiresComplication {
		t.Fatal("expected requires complication to be true")
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice for gate recovery, got %d", domain.calls)
	}
	var foundGate bool
	var foundSpotlight bool
	for _, cmd := range domain.commands {
		switch cmd.Type {
		case command.Type("session.gate_open"):
			foundGate = true
		case command.Type("session.spotlight_set"):
			foundSpotlight = true
		}
	}
	if !foundGate {
		t.Fatal("expected session gate open command")
	}
	if !foundSpotlight {
		t.Fatal("expected session spotlight set command")
	}
}

func TestApplyRollOutcome_AlreadyAppliedWithOpenGateRepairsSpotlight(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := testTimestamp

	dhStore.Snapshots["camp-1"] = projectionstore.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     2,
	}

	svc.stores.SessionGate = &fakeOpenSessionGateStore{
		gate: storage.SessionGate{
			CampaignID: "camp-1",
			SessionID:  "sess-1",
			GateID:     "gate-open",
			GateType:   "gm_consequence",
			Reason:     "gm_consequence",
			Status:     session.GateStatusOpen,
			CreatedAt:  now,
		},
	}

	rollEvent := fearRollEvent(t, "req-roll-open-gate").appendTo(eventStore)

	_, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now.Add(time.Second),
		Type:        event.Type("action.outcome_applied"),
		SessionID:   "sess-1",
		RequestID:   "req-roll-open-gate",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "outcome",
		EntityID:    "req-roll-open-gate",
		PayloadJSON: []byte(`{"request_id":"req-roll-open-gate","roll_seq":1}`),
	})
	if err != nil {
		t.Fatalf("append outcome event: %v", err)
	}

	spotlightPayload := session.SpotlightSetPayload{SpotlightType: string(session.SpotlightTypeGM)}
	spotlightJSON, err := json.Marshal(spotlightPayload)
	if err != nil {
		t.Fatalf("encode spotlight payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("session.spotlight_set"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("session.spotlight_set"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-open-gate",
				EntityType:  "session_spotlight",
				EntityID:    "sess-1",
				PayloadJSON: spotlightJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := testSessionCtx("camp-1", "sess-1", "req-roll-open-gate")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if !resp.RequiresComplication {
		t.Fatal("expected requires complication to be true")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once for spotlight repair, got %d", domain.calls)
	}
	if len(domain.commands) != 1 || domain.commands[0].Type != command.Type("session.spotlight_set") {
		t.Fatalf("expected spotlight set command, got %+v", domain.commands)
	}
}
