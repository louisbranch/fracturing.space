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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

// --- ApplyRollOutcome tests ---

func TestApplyRollOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyRollOutcome(context.Background(), &pb.ApplyRollOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRollOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyRollOutcome(context.Background(), &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "")
	_, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyRollOutcome_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	svc.stores.Domain = nil

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-outcome-required",
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
		RequestID:   "req-roll-outcome-required",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-outcome-required",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-outcome-required")
	_, err = svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyRollOutcome_IdempotentWhenAlreadyAppliedEvenWithOpenGate(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{
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

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-duplicate",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
			"gm_move":      true,
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
		RequestID:   "req-roll-duplicate",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-duplicate",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
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

	domain := svc.stores.Domain.(*fakeDomainEngine)
	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-duplicate")
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
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     2,
	}

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-gate-retry",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
			"gm_move":      true,
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
		RequestID:   "req-roll-gate-retry",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-gate-retry",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
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
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-gate-retry")
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

func TestApplyRollOutcome_PartialRetrySkipsRepeatedGMFearSet(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{
		CampaignID: "camp-1",
		GMFear:     1,
	}

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-partial-retry",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
			"gm_move":      true,
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
		RequestID:   "req-roll-partial-retry",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-partial-retry",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:    "camp-1",
		Timestamp:     now.Add(time.Second),
		Type:          event.Type("sys.daggerheart.gm_fear_changed"),
		SessionID:     "sess-1",
		RequestID:     "req-roll-partial-retry",
		ActorType:     event.ActorTypeSystem,
		EntityType:    "campaign",
		EntityID:      "camp-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   []byte(`{"before":0,"after":1}`),
	})
	if err != nil {
		t.Fatalf("append gm fear event: %v", err)
	}

	gatePayload := session.GateOpenedPayload{
		GateID:   "gate-1",
		GateType: "gm_consequence",
		Reason:   "gm_consequence",
		Metadata: map[string]any{"roll_seq": uint64(rollEvent.Seq), "request_id": "req-roll-partial-retry"},
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
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("action.outcome_applied"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-partial-retry",
					EntityType:  "outcome",
					EntityID:    "req-roll-partial-retry",
					PayloadJSON: []byte(`{"request_id":"req-roll-partial-retry","roll_seq":1}`),
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.gate_opened"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-partial-retry",
					EntityType:  "session_gate",
					EntityID:    "gate-1",
					PayloadJSON: gateJSON,
				},
				event.Event{
					CampaignID:  "camp-1",
					Type:        event.Type("session.spotlight_set"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "sess-1",
					RequestID:   "req-roll-partial-retry",
					EntityType:  "session_spotlight",
					EntityID:    "sess-1",
					PayloadJSON: spotlightJSON,
				},
			),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-partial-retry")
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
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	var payload action.OutcomeApplyPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode outcome command payload: %v", err)
	}
	for _, effect := range payload.PreEffects {
		if effect.Type == "sys.daggerheart.gm_fear_changed" {
			t.Fatal("did not expect gm fear pre_effect on partial retry")
		}
	}
	if snap := dhStore.Snapshots["camp-1"]; snap.GMFear != 1 {
		t.Fatalf("gm fear = %d, want %d", snap.GMFear, 1)
	}
}

func TestApplyRollOutcome_PartialRetrySkipsRepeatedCharacterPatch(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-patch-retry",
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
		RequestID:   "req-roll-patch-retry",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-patch-retry",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:    "camp-1",
		Timestamp:     now.Add(time.Second),
		Type:          event.Type("sys.daggerheart.character_state_patched"),
		SessionID:     "sess-1",
		RequestID:     "req-roll-patch-retry",
		ActorType:     event.ActorTypeSystem,
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   []byte(`{"character_id":"char-1","hope_before":2,"hope_after":3,"stress_before":3,"stress_after":3}`),
	})
	if err != nil {
		t.Fatalf("append patch event: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.outcome.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "camp-1",
				Type:        event.Type("action.outcome_applied"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "sess-1",
				RequestID:   "req-roll-patch-retry",
				EntityType:  "outcome",
				EntityID:    "req-roll-patch-retry",
				PayloadJSON: []byte(`{"request_id":"req-roll-patch-retry"}`),
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-patch-retry")
	resp, err := svc.ApplyRollOutcome(ctx, &pb.ApplyRollOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyRollOutcome returned error: %v", err)
	}
	if resp.RollSeq != rollEvent.Seq {
		t.Fatalf("roll seq = %d, want %d", resp.RollSeq, rollEvent.Seq)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 || domain.commands[0].Type != command.Type("action.outcome.apply") {
		t.Fatalf("expected only outcome apply command, got %+v", domain.commands)
	}
}

func TestApplyRollOutcome_AlreadyAppliedWithOpenGateRepairsSpotlight(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore.Snapshots["camp-1"] = storage.DaggerheartSnapshot{
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

	rollPayload := action.RollResolvePayload{
		RequestID: "req-roll-open-gate",
		RollSeq:   1,
		Results:   map[string]any{"d20": 1},
		Outcome:   pb.Outcome_FAILURE_WITH_FEAR.String(),
		SystemData: map[string]any{
			"character_id": "char-1",
			"roll_kind":    pb.RollKind_ROLL_KIND_ACTION.String(),
			"hope_fear":    true,
			"gm_move":      true,
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
		RequestID:   "req-roll-open-gate",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-roll-open-gate",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	_, err = eventStore.AppendEvent(context.Background(), event.Event{
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
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"), "req-roll-open-gate")
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
