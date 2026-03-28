package sessionrolltransport

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
)

func TestHandlerSessionActionRollSuccess(t *testing.T) {
	var spendCalls []HopeSpendInput
	var resolveInput RollResolveInput
	var countdownID string

	handler := newTestHandler(Dependencies{
		ExecuteHopeSpend: func(_ context.Context, in HopeSpendInput) error {
			spendCalls = append(spendCalls, in)
			return nil
		},
		ExecuteActionRollResolve: func(_ context.Context, in RollResolveInput) (uint64, error) {
			resolveInput = in
			return 9, nil
		},
		AdvanceBreathCountdown: func(_ context.Context, _, _, id string, _ bool) error {
			countdownID = id
			return nil
		},
	})

	ctx := grpcmeta.WithInvocationID(grpcmeta.WithRequestID(context.Background(), "req-1"), "inv-1")
	resp, err := handler.SessionActionRoll(ctx, &pb.SessionActionRollRequest{
		CampaignId:             "camp-1",
		SessionId:              "sess-1",
		CharacterId:            "char-1",
		Trait:                  "agility",
		Difficulty:             10,
		BreathSceneCountdownId: "countdown-1",
		Modifiers: []*pb.ActionRollModifier{
			{Value: 2, Source: "experience"},
		},
		HopeSpends: []*pb.ActionRollHopeSpend{
			{Amount: 1, Source: "experience"},
		},
	})
	if err != nil {
		t.Fatalf("SessionActionRoll returned error: %v", err)
	}
	if resp.GetRollSeq() != 9 {
		t.Fatalf("roll_seq = %d, want 9", resp.GetRollSeq())
	}
	if len(spendCalls) != 1 {
		t.Fatalf("hope spend calls = %d, want 1", len(spendCalls))
	}
	if spendCalls[0].HopeBefore != 3 || spendCalls[0].HopeAfter != 2 {
		t.Fatalf("hope spend before/after = %d/%d, want 3/2", spendCalls[0].HopeBefore, spendCalls[0].HopeAfter)
	}
	if countdownID != "countdown-1" {
		t.Fatalf("countdown id = %q, want countdown-1", countdownID)
	}
	if resolveInput.EntityType != "roll" || resolveInput.EntityID != "req-1" {
		t.Fatalf("resolve entity = %s/%s", resolveInput.EntityType, resolveInput.EntityID)
	}

	var payload map[string]any
	if err := json.Unmarshal(resolveInput.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got := payload["request_id"]; got != "req-1" {
		t.Fatalf("payload request_id = %v, want req-1", got)
	}
}
