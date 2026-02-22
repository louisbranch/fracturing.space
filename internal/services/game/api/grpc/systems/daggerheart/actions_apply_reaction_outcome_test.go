package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

// --- ApplyRollOutcome tests ---
// --- ApplyReactionOutcome tests ---

func TestApplyReactionOutcome_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyReactionOutcome_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1", RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyReactionOutcome(context.Background(), &pb.DaggerheartApplyReactionOutcomeRequest{
		RollSeq: 1,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_MissingRollSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1")
	_, err := svc.ApplyReactionOutcome(ctx, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyReactionOutcome_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	rollCtx := grpcmeta.WithRequestID(context.Background(), "req-react-outcome-required")
	configureActionRollDomain(t, svc, "req-react-outcome-required")
	rollResp, err := svc.SessionActionRoll(rollCtx, &pb.SessionActionRollRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CharacterId: "char-1",
		Trait:       "agility",
		RollKind:    pb.RollKind_ROLL_KIND_REACTION,
		Difficulty:  10,
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	svc.stores.Domain = nil

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-react-outcome-required",
	)
	_, err = svc.ApplyReactionOutcome(ctx, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollResp.RollSeq,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyReactionOutcome_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	configureNoopDomain(svc)

	rollPayload := action.RollResolvePayload{
		RequestID: "req-react-outcome-legacy",
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
	rollEvent, err := eventStore.AppendEvent(context.Background(), event.Event{
		CampaignID:  "camp-1",
		Timestamp:   now,
		Type:        event.Type("action.roll_resolved"),
		SessionID:   "sess-1",
		RequestID:   "req-react-outcome-legacy",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-react-outcome-legacy",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		withCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-react-outcome-legacy",
	)
	resp, err := svc.ApplyReactionOutcome(ctx, &pb.DaggerheartApplyReactionOutcomeRequest{
		SessionId: "sess-1",
		RollSeq:   rollEvent.Seq,
	})
	if err != nil {
		t.Fatalf("ApplyReactionOutcome returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result in response")
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("expected character_id char-1, got %s", resp.CharacterId)
	}
	if resp.Result.GetOutcome() != pb.Outcome_SUCCESS_WITH_HOPE {
		t.Fatalf("expected outcome SUCCESS_WITH_HOPE, got %s", resp.Result.GetOutcome())
	}
	if !resp.Result.GetSuccess() {
		t.Fatal("expected reaction success")
	}
}
