package damagetransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerApplyDamageSuccess(t *testing.T) {
	var commandInput SystemCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			commandInput = in
			return nil
		},
	})

	ctx := grpcmeta.WithInvocationID(grpcmeta.WithRequestID(testContextWithSessionID("sess-1"), "req-1"), "inv-1")
	resp, err := handler.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if resp.CharacterID != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterID)
	}
	if commandInput.CommandType != commandids.DaggerheartDamageApply {
		t.Fatalf("command type = %q", commandInput.CommandType)
	}
}

func TestHandlerApplyDamageRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.Internal, err)
	}
}

func TestHandlerApplyDamageRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.ApplyDamage(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerApplyDamageRequiresRollSeqWhenFlagged(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error { return nil },
	})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:        "camp-1",
		CharacterId:       "char-1",
		RequireDamageRoll: true,
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestHandlerApplyDamageRejectsNonRollEvent(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Event: testEventStore{event: event.Event{Type: event.Type("other.event")}},
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	ctx := testContextWithSessionID("sess-1")
	rollSeq := uint64(7)
	_, err := handler.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		RollSeq:     &rollSeq,
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.InvalidArgument, err)
	}
}
