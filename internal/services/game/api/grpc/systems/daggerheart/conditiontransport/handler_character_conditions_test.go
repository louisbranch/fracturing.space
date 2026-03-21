package conditiontransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerApplyConditionsSuccess(t *testing.T) {
	var commands []DomainCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commands = append(commands, in)
			return nil
		},
	})

	ctx := grpcmeta.WithInvocationID(grpcmeta.WithRequestID(testContextWithSessionID("sess-1"), "req-1"), "inv-1")
	resp, err := handler.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		AddConditions: []*pb.DaggerheartConditionState{{
			Id:       "vulnerable",
			Class:    pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
			Standard: pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE,
		}},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if resp.CharacterID != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterID)
	}
	if len(resp.Added) != 1 || resp.Added[0].Standard != pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE {
		t.Fatalf("added = %v, want vulnerable", resp.Added)
	}
	if len(commands) != 1 || commands[0].CommandType != commandids.DaggerheartConditionChange {
		t.Fatalf("commands = %+v, want one condition change", commands)
	}
}

func TestHandlerApplyConditionsLifeStatePatchOnly(t *testing.T) {
	var commands []DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{
			state: projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				LifeState:   daggerheart.LifeStateAlive,
			},
		},
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commands = append(commands, in)
			return nil
		},
	})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if len(commands) != 1 || commands[0].CommandType != commandids.DaggerheartCharacterStatePatch {
		t.Fatalf("commands = %+v, want one character state patch", commands)
	}
}

func TestHandlerApplyConditionsRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		AddConditions: []*pb.DaggerheartConditionState{{
			Id:       "hidden",
			Class:    pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
			Standard: pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN,
		}},
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
