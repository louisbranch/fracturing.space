package damagetransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerApplyAdversaryDamageSuccess(t *testing.T) {
	var commandInput SystemCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			commandInput = in
			return nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				HP:          10,
				Armor:       1,
				Major:       5,
				Severe:      8,
			}, nil
		},
	})

	ctx := grpcmeta.WithInvocationID(grpcmeta.WithRequestID(testContextWithSessionID("sess-1"), "req-adv-1"), "inv-1")
	resp, err := handler.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if resp.AdversaryID != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryID)
	}
	if commandInput.CommandType != commandids.DaggerheartAdversaryDamageApply {
		t.Fatalf("command type = %q", commandInput.CommandType)
	}
}

func TestHandlerApplyAdversaryDamageRequiresLoader(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error { return nil },
	})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.Internal, err)
	}
}

func TestHandlerApplyAdversaryDamageRejectsOpenGate(t *testing.T) {
	handler := newTestHandler(Dependencies{
		SessionGate: testOpenGateStore{},
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			t.Fatal("unexpected adversary load")
			return projectionstore.DaggerheartAdversary{}, nil
		},
	})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.FailedPrecondition, err)
	}
}
