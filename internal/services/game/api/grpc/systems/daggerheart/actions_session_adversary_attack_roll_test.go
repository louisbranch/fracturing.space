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
	"google.golang.org/grpc/codes"
)

// --- SessionAdversaryAttackRoll tests ---

func TestSessionAdversaryAttackRoll_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionAdversaryAttackRoll_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		SessionId: "sess-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackRoll_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackRoll_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
		CampaignId: "camp-1", SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionAdversaryAttackRoll_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	payload := action.RollResolvePayload{
		RequestID: "req-adv-roll-success",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":                       []int{7},
			workflowtransport.KeyRoll:     7,
			workflowtransport.KeyModifier: 0,
			workflowtransport.KeyTotal:    7,
			"advantage":                   0,
			"disadvantage":                0,
		},
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "adv-1",
			workflowtransport.KeyAdversaryID: "adv-1",
			workflowtransport.KeyRollKind:    "adversary_roll",
			workflowtransport.KeyRoll:        7,
			workflowtransport.KeyModifier:    0,
			workflowtransport.KeyTotal:       7,
			"advantage":                      0,
			"disadvantage":                   0,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary roll payload: %v", err)
	}

	svc.stores.Write.Executor = &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.roll_resolved"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-roll-success",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	ctx := grpcmeta.WithRequestID(context.Background(), "req-adv-roll-success")
	resp, err := svc.SessionAdversaryAttackRoll(ctx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
}

func TestSessionAdversaryAttackRoll_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)

	payload := action.RollResolvePayload{
		RequestID: "req-adv-roll",
		RollSeq:   1,
		Results: map[string]any{
			"rolls":                       []int{12, 18},
			workflowtransport.KeyRoll:     18,
			workflowtransport.KeyModifier: 2,
			workflowtransport.KeyTotal:    20,
			"advantage":                   1,
			"disadvantage":                0,
		},
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "adv-1",
			workflowtransport.KeyAdversaryID: "adv-1",
			workflowtransport.KeyRollKind:    "adversary_roll",
			workflowtransport.KeyRoll:        18,
			workflowtransport.KeyModifier:    2,
			workflowtransport.KeyTotal:       20,
			"advantage":                      1,
			"disadvantage":                   0,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary roll payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("action.roll.resolve"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("action.roll_resolved"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adv-roll",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}

	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(context.Background(), "req-adv-roll")
	resp, err := svc.SessionAdversaryAttackRoll(ctx, &pb.SessionAdversaryAttackRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackRoll returned error: %v", err)
	}
	if resp.RollSeq == 0 {
		t.Fatal("expected non-zero roll seq")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("action.roll.resolve") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "action.roll.resolve")
	}
	if got := len(eventStore.Events["camp-1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["camp-1"][0].Type != event.Type("action.roll_resolved") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["camp-1"][0].Type, event.Type("action.roll_resolved"))
	}
}

// --- SessionAdversaryActionCheck tests ---
