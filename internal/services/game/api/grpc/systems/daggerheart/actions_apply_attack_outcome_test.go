package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestApplyAttackOutcome_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

	rollPayload := action.RollResolvePayload{
		RequestID: "req-atk-outcome-1",
		RollSeq:   1,
		Results:   map[string]any{"d20": 18},
		Outcome:   pb.Outcome_SUCCESS_WITH_HOPE.String(),
		SystemData: map[string]any{
			workflowtransport.KeyCharacterID: "char-1",
			workflowtransport.KeyRollKind:    pb.RollKind_ROLL_KIND_ACTION.String(),
			workflowtransport.KeyHopeFear:    true,
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
		RequestID:   "req-atk-outcome-1",
		ActorType:   event.ActorTypeSystem,
		EntityType:  "roll",
		EntityID:    "req-atk-outcome-1",
		PayloadJSON: rollJSON,
	})
	if err != nil {
		t.Fatalf("append roll event: %v", err)
	}

	ctx := grpcmeta.WithRequestID(
		workflowtransport.WithCampaignSessionMetadata(context.Background(), "camp-1", "sess-1"),
		"req-atk-outcome-1",
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
		t.Fatalf("expected attacker char-1, got %s", resp.CharacterId)
	}
	if resp.Result.GetOutcome() != pb.Outcome_SUCCESS_WITH_HOPE {
		t.Fatalf("expected outcome SUCCESS_WITH_HOPE, got %s", resp.Result.GetOutcome())
	}
	if !resp.Result.GetSuccess() {
		t.Fatal("expected successful attack outcome")
	}
}
