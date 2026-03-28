package damagetransport

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
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

func TestHandlerApplyDamageReturnsMitigationChoiceWhenRequired(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{
			profile: projectionstore.DaggerheartCharacterProfile{
				CampaignID:      "camp-1",
				CharacterID:     "char-1",
				EquippedArmorID: "armor-1",
				ArmorMax:        1,
				MajorThreshold:  5,
				SevereThreshold: 8,
			},
			state: projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Hp:          6,
				Armor:       1,
			},
		},
		Content: testContentStore{
			adversaryEntries: map[string]contentstore.DaggerheartAdversaryEntry{
				"entry-goblin": {ID: "entry-goblin"},
			},
			armors: map[string]contentstore.DaggerheartArmor{
				"armor-1": {ID: "armor-1"},
			},
		},
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	result, err := handler.ApplyDamage(testContextWithSessionID("sess-1"), &pb.DaggerheartApplyDamageRequest{
		CampaignId:              "camp-1",
		CharacterId:             "char-1",
		RequireMitigationChoice: true,
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     8,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if result.Choice == nil {
		t.Fatal("expected mitigation choice")
	}
	if got := result.Choice.GetStage(); got != pb.DaggerheartCombatChoiceStage_DAGGERHEART_COMBAT_CHOICE_STAGE_DAMAGE_MITIGATION {
		t.Fatalf("choice stage = %v", got)
	}
	if got := result.Choice.GetCharacterId(); got != "char-1" {
		t.Fatalf("choice character_id = %q, want char-1", got)
	}
	if got := result.Choice.GetOptionCodes(); len(got) != 2 || got[0] != "armor.base_slot" || got[1] != "armor.decline" {
		t.Fatalf("choice option_codes = %v", got)
	}
	if result.Choice.GetDeclinePreview() == nil || result.Choice.GetSpendBaseArmorPreview() == nil {
		t.Fatal("expected both damage previews")
	}
}

func TestHandlerApplyDamageExplicitDeclineSkipsArmorSpend(t *testing.T) {
	var payload daggerheartpayload.DamageApplyPayload
	commandCalls := 0
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{
			profile: projectionstore.DaggerheartCharacterProfile{
				CampaignID:      "camp-1",
				CharacterID:     "char-1",
				EquippedArmorID: "armor-1",
				ArmorMax:        1,
				MajorThreshold:  5,
				SevereThreshold: 8,
			},
			state: projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Hp:          6,
				Armor:       1,
			},
		},
		Content: testContentStore{
			adversaryEntries: map[string]contentstore.DaggerheartAdversaryEntry{
				"entry-goblin": {ID: "entry-goblin"},
			},
			armors: map[string]contentstore.DaggerheartArmor{
				"armor-1": {ID: "armor-1"},
			},
		},
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			commandCalls++
			if in.CommandType != commandids.DaggerheartDamageApply {
				return nil
			}
			if err := json.Unmarshal(in.PayloadJSON, &payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			return nil
		},
	})

	result, err := handler.ApplyDamage(testContextWithSessionID("sess-1"), &pb.DaggerheartApplyDamageRequest{
		CampaignId:              "camp-1",
		CharacterId:             "char-1",
		RequireMitigationChoice: true,
		MitigationDecision: &pb.DaggerheartDamageMitigationDecision{
			BaseArmor: pb.DaggerheartBaseArmorDecision_DAGGERHEART_BASE_ARMOR_DECISION_DECLINE,
		},
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     8,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if result.Choice != nil {
		t.Fatalf("choice = %#v, want nil after explicit decision", result.Choice)
	}
	if commandCalls != 1 {
		t.Fatalf("command calls = %d, want 1", commandCalls)
	}
	if payload.ArmorBefore == nil || payload.ArmorAfter == nil {
		t.Fatalf("payload armor pointers missing: %+v", payload)
	}
	if got := *payload.ArmorBefore; got != 1 {
		t.Fatalf("armor_before = %d, want 1", got)
	}
	if got := *payload.ArmorAfter; got != 1 {
		t.Fatalf("armor_after = %d, want 1", got)
	}
	if payload.ArmorSpent != 0 {
		t.Fatalf("armor_spent = %d, want 0", payload.ArmorSpent)
	}
}
