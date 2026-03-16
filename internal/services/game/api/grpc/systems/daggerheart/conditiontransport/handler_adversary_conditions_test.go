package conditiontransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerApplyAdversaryConditionsSuccess(t *testing.T) {
	var commands []DomainCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commands = append(commands, in)
			return nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				Conditions:  []projectionstore.DaggerheartConditionState{{Standard: daggerheart.ConditionHidden}},
			}, nil
		},
	})

	ctx := grpcmeta.WithRequestID(testContextWithSessionID("sess-1"), "req-adv-1")
	resp, err := handler.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		AddConditions: []*pb.DaggerheartConditionState{{
			Id:       "vulnerable",
			Class:    pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
			Standard: pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE,
		}},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryConditions returned error: %v", err)
	}
	if resp.AdversaryID != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryID)
	}
	if len(commands) != 1 || commands[0].CommandType != commandids.DaggerheartAdversaryConditionChange {
		t.Fatalf("commands = %+v, want one adversary condition change", commands)
	}
}

func TestHandlerApplyAdversaryConditionsRejectsSessionMismatch(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Event: testEventStore{event: event.Event{SessionID: "other-session"}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				Conditions:  []projectionstore.DaggerheartConditionState{{Standard: daggerheart.ConditionHidden}},
			}, nil
		},
	})

	ctx := testContextWithSessionID("sess-1")
	rollSeq := uint64(7)
	_, err := handler.ApplyAdversaryConditions(ctx, &pb.DaggerheartApplyAdversaryConditionsRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		RollSeq:     &rollSeq,
		AddConditions: []*pb.DaggerheartConditionState{{
			Id:       "vulnerable",
			Class:    pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
			Standard: pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE,
		}},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.InvalidArgument, err)
	}
}
