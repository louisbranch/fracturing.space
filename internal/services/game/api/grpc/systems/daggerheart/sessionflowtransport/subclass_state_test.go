package sessionflowtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestSessionAttackFlowConsumesStoredSubclassBonuses(t *testing.T) {
	var (
		actionReq  *pb.SessionActionRollRequest
		damageReq  *pb.SessionDamageRollRequest
		patchCalls []CharacterStatePatchInput
	)
	handler := NewHandler(Dependencies{
		SessionActionRoll: func(_ context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
			cloned := *in
			actionReq = &cloned
			return &pb.SessionActionRollResponse{RollSeq: 11}, nil
		},
		SessionDamageRoll: func(_ context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
			cloned := *in
			damageReq = &cloned
			return &pb.SessionDamageRollResponse{RollSeq: 12, Total: 13}, nil
		},
		ApplyRollOutcome: func(context.Context, *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
			return &pb.ApplyRollOutcomeResponse{}, nil
		},
		ApplyAttackOutcome: func(context.Context, *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
			return &pb.DaggerheartApplyAttackOutcomeResponse{
				Result: &pb.DaggerheartAttackOutcomeResult{Success: true},
			}, nil
		},
		ApplyDamage: func(_ context.Context, in *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
			return &pb.DaggerheartApplyDamageResponse{CharacterId: in.GetCharacterId()}, nil
		},
		LoadCharacterProfile: func(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
			return projectionstore.DaggerheartCharacterProfile{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
			}, nil
		},
		LoadCharacterState: func(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
			return projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				SubclassState: &projectionstore.DaggerheartSubclassState{
					ContactsEverywhereActionDieBonus:       3,
					ContactsEverywhereDamageDiceBonusCount: 2,
					ElementalistActionBonus:                2,
					ElementalistDamageBonus:                3,
					TranscendenceActive:                    true,
					TranscendenceTraitBonusTarget:          "agility",
					TranscendenceTraitBonusValue:           1,
					TranscendenceProficiencyBonus:          1,
					ElementalChannel:                       daggerheart.ElementalChannelAir,
				},
			}, nil
		},
		LoadSubclass: func(context.Context, string) (contentstore.DaggerheartSubclass, error) {
			return contentstore.DaggerheartSubclass{}, nil
		},
		LoadArmor: func(context.Context, string) (contentstore.DaggerheartArmor, error) {
			return contentstore.DaggerheartArmor{}, nil
		},
		ExecuteCharacterStatePatch: func(_ context.Context, in CharacterStatePatchInput) error {
			patchCalls = append(patchCalls, in)
			return nil
		},
	})

	_, err := handler.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CharacterId: "char-1",
		TargetId:    "char-2",
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC,
		},
		AttackProfile: &pb.SessionAttackFlowRequest_StandardAttack{
			StandardAttack: &pb.SessionStandardAttackProfile{
				Trait:       "agility",
				AttackRange: pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE,
				DamageDice:  []*pb.DiceSpec{{Sides: 6, Count: 1}},
			},
		},
	})
	if err != nil {
		t.Fatalf("SessionAttackFlow returned error: %v", err)
	}
	if actionReq == nil {
		t.Fatal("expected action roll request")
	}
	if actionReq.GetAdvantage() != 1 {
		t.Fatalf("action advantage = %d, want 1", actionReq.GetAdvantage())
	}
	if got := actionReq.GetModifiers(); len(got) != 3 {
		t.Fatalf("action modifiers = %+v, want 3 modifiers", got)
	}
	if got := damageReq; got == nil {
		t.Fatal("expected damage roll request")
	}
	if damageReq.GetModifier() != 4 {
		t.Fatalf("damage modifier = %d, want 4", damageReq.GetModifier())
	}
	if got := damageReq.GetDice(); len(got) != 2 || got[1].GetSides() != 8 || got[1].GetCount() != 2 {
		t.Fatalf("damage dice = %+v, want base die plus 2d8 bonus", got)
	}
	if len(patchCalls) != 2 {
		t.Fatalf("patch calls = %d, want 2", len(patchCalls))
	}
	if patchCalls[0].SubclassStateAfter == nil || patchCalls[0].SubclassStateAfter.ContactsEverywhereActionDieBonus != 0 || patchCalls[0].SubclassStateAfter.ElementalistActionBonus != 0 {
		t.Fatalf("first patch subclass state after = %+v, want action bonuses cleared", patchCalls[0].SubclassStateAfter)
	}
	if patchCalls[1].SubclassStateAfter == nil || patchCalls[1].SubclassStateAfter.ContactsEverywhereDamageDiceBonusCount != 0 || patchCalls[1].SubclassStateAfter.ElementalistDamageBonus != 0 {
		t.Fatalf("second patch subclass state after = %+v, want damage bonuses cleared", patchCalls[1].SubclassStateAfter)
	}
}

func TestSessionReactionFlowAppliesSubclassTraitAndAirBonuses(t *testing.T) {
	var actionReq *pb.SessionActionRollRequest
	handler := NewHandler(Dependencies{
		SessionActionRoll: func(_ context.Context, in *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
			cloned := *in
			actionReq = &cloned
			return &pb.SessionActionRollResponse{RollSeq: 21}, nil
		},
		ApplyRollOutcome: func(context.Context, *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
			return &pb.ApplyRollOutcomeResponse{}, nil
		},
		ApplyReactionOutcome: func(context.Context, *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error) {
			return &pb.DaggerheartApplyReactionOutcomeResponse{}, nil
		},
		LoadCharacterState: func(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
			return projectionstore.DaggerheartCharacterState{
				SubclassState: &projectionstore.DaggerheartSubclassState{
					TranscendenceActive:           true,
					TranscendenceTraitBonusTarget: "agility",
					TranscendenceTraitBonusValue:  1,
					ElementalChannel:              daggerheart.ElementalChannelAir,
				},
			}, nil
		},
	})

	_, err := handler.SessionReactionFlow(context.Background(), &pb.SessionReactionFlowRequest{
		CampaignId:   "camp-1",
		SessionId:    "sess-1",
		SceneId:      "scene-1",
		CharacterId:  "char-1",
		Trait:        "agility",
		Advantage:    2,
		Disadvantage: 1,
	})
	if err != nil {
		t.Fatalf("SessionReactionFlow returned error: %v", err)
	}
	if actionReq == nil {
		t.Fatal("expected action roll request")
	}
	if actionReq.GetAdvantage() != 3 {
		t.Fatalf("advantage = %d, want 3", actionReq.GetAdvantage())
	}
	if got := actionReq.GetModifiers(); len(got) != 1 || got[0].GetValue() != 1 {
		t.Fatalf("modifiers = %+v, want transcendence trait +1", got)
	}
}

func TestSessionAdversaryAttackFlowAddsTranscendenceEvasionBonus(t *testing.T) {
	var outcomeReq *pb.DaggerheartApplyAdversaryAttackOutcomeRequest
	handler := NewHandler(Dependencies{
		LoadAdversary: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				AdversaryID:      "adv-1",
				AdversaryEntryID: "adversary.goblin",
			}, nil
		},
		LoadAdversaryEntry: func(context.Context, string) (contentstore.DaggerheartAdversaryEntry, error) {
			return contentstore.DaggerheartAdversaryEntry{
				AttackModifier: 1,
				StandardAttack: contentstore.DaggerheartAdversaryAttack{
					DamageDice: []contentstore.DaggerheartDamageDie{{Sides: 6, Count: 1}},
					DamageType: "physical",
				},
			}, nil
		},
		LoadCharacterProfile: func(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
			return projectionstore.DaggerheartCharacterProfile{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
			}, nil
		},
		LoadCharacterState: func(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
			return projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				SubclassState: &projectionstore.DaggerheartSubclassState{
					TranscendenceActive:       true,
					TranscendenceEvasionBonus: 2,
				},
			}, nil
		},
		LoadSubclass: func(context.Context, string) (contentstore.DaggerheartSubclass, error) {
			return contentstore.DaggerheartSubclass{}, nil
		},
		SessionAdversaryAttackRoll: func(context.Context, *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
			return &pb.SessionAdversaryAttackRollResponse{RollSeq: 41}, nil
		},
		ApplyAdversaryAttackOutcome: func(_ context.Context, in *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
			outcomeReq = in
			return &pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
				Result: &pb.DaggerheartAdversaryAttackOutcomeResult{Success: false},
			}, nil
		},
		SessionDamageRoll: func(context.Context, *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
			return &pb.SessionDamageRollResponse{RollSeq: 42, Total: 5}, nil
		},
		ApplyDamage: func(context.Context, *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
			return &pb.DaggerheartApplyDamageResponse{}, nil
		},
	})

	_, err := handler.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		AdversaryId: "adv-1",
		TargetId:    "char-1",
		Difficulty:  10,
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackFlow returned error: %v", err)
	}
	if outcomeReq == nil {
		t.Fatal("expected adversary attack outcome request")
	}
	if outcomeReq.GetDifficulty() != 12 {
		t.Fatalf("difficulty = %d, want 12", outcomeReq.GetDifficulty())
	}
}

func TestSessionAttackFlowAppliesWaterSplashToNearbyAdversaries(t *testing.T) {
	var (
		adversaryDamageReq *pb.DaggerheartApplyAdversaryDamageRequest
		adversaryUpdates   []AdversaryUpdateInput
	)
	handler := NewHandler(Dependencies{
		SessionActionRoll: func(context.Context, *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error) {
			return &pb.SessionActionRollResponse{RollSeq: 51}, nil
		},
		SessionDamageRoll: func(context.Context, *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
			return &pb.SessionDamageRollResponse{RollSeq: 52, Total: 9}, nil
		},
		ApplyRollOutcome: func(context.Context, *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error) {
			return &pb.ApplyRollOutcomeResponse{}, nil
		},
		ApplyAttackOutcome: func(context.Context, *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error) {
			return &pb.DaggerheartApplyAttackOutcomeResponse{
				Result: &pb.DaggerheartAttackOutcomeResult{Success: true},
			}, nil
		},
		ApplyDamage: func(context.Context, *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
			return &pb.DaggerheartApplyDamageResponse{}, nil
		},
		ApplyAdversaryDamage: func(_ context.Context, in *pb.DaggerheartApplyAdversaryDamageRequest) (*pb.DaggerheartApplyAdversaryDamageResponse, error) {
			adversaryDamageReq = in
			return &pb.DaggerheartApplyAdversaryDamageResponse{
				AdversaryId: in.GetAdversaryId(),
				Adversary: &pb.DaggerheartAdversary{
					Id:     in.GetAdversaryId(),
					Hp:     4,
					HpMax:  8,
					Stress: 0,
				},
			}, nil
		},
		LoadCharacterProfile: func(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
			return projectionstore.DaggerheartCharacterProfile{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
			}, nil
		},
		LoadCharacterState: func(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
			return projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				SubclassState: &projectionstore.DaggerheartSubclassState{
					ElementalChannel: daggerheart.ElementalChannelWater,
				},
			}, nil
		},
		LoadAdversary: func(_ context.Context, _ string, adversaryID string, _ string) (projectionstore.DaggerheartAdversary, error) {
			switch adversaryID {
			case "adv-1":
				return projectionstore.DaggerheartAdversary{AdversaryID: "adv-1", AdversaryEntryID: "adversary.water-target", HP: 8, HPMax: 8, Stress: 0, StressMax: 4}, nil
			case "adv-2":
				return projectionstore.DaggerheartAdversary{AdversaryID: "adv-2", AdversaryEntryID: "adversary.water-target", HP: 8, HPMax: 8, Stress: 1, StressMax: 4}, nil
			case "adv-3":
				return projectionstore.DaggerheartAdversary{AdversaryID: "adv-3", AdversaryEntryID: "adversary.water-target", HP: 8, HPMax: 8, Stress: 4, StressMax: 4}, nil
			default:
				return projectionstore.DaggerheartAdversary{}, nil
			}
		},
		LoadAdversaryEntry: func(context.Context, string) (contentstore.DaggerheartAdversaryEntry, error) {
			return contentstore.DaggerheartAdversaryEntry{}, nil
		},
		LoadSubclass: func(context.Context, string) (contentstore.DaggerheartSubclass, error) {
			return contentstore.DaggerheartSubclass{}, nil
		},
		LoadArmor: func(context.Context, string) (contentstore.DaggerheartArmor, error) {
			return contentstore.DaggerheartArmor{}, nil
		},
		ExecuteAdversaryUpdate: func(_ context.Context, in AdversaryUpdateInput) error {
			adversaryUpdates = append(adversaryUpdates, in)
			return nil
		},
	})

	resp, err := handler.SessionAttackFlow(context.Background(), &pb.SessionAttackFlowRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		CharacterId:       "char-1",
		TargetId:          "adv-1",
		TargetIsAdversary: true,
		NearbyAdversaryIds: []string{
			"adv-1",
			"adv-2",
			"adv-2",
			"adv-3",
		},
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC,
		},
		AttackProfile: &pb.SessionAttackFlowRequest_StandardAttack{
			StandardAttack: &pb.SessionStandardAttackProfile{
				Trait:       "instinct",
				AttackRange: pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE,
				DamageDice:  []*pb.DiceSpec{{Sides: 8, Count: 1}},
			},
		},
	})
	if err != nil {
		t.Fatalf("SessionAttackFlow returned error: %v", err)
	}
	if adversaryDamageReq == nil {
		t.Fatal("expected adversary damage request")
	}
	if adversaryDamageReq.GetAdversaryId() != "adv-1" {
		t.Fatalf("adversary damage target = %q, want adv-1", adversaryDamageReq.GetAdversaryId())
	}
	if resp.GetAdversaryDamageApplied() == nil {
		t.Fatal("expected adversary damage response")
	}
	if len(adversaryUpdates) != 1 {
		t.Fatalf("adversary updates = %d, want 1", len(adversaryUpdates))
	}
	if adversaryUpdates[0].Adversary.AdversaryID != "adv-2" || adversaryUpdates[0].UpdatedStress != 2 {
		t.Fatalf("water splash update = %+v, want adv-2 stress 2", adversaryUpdates[0])
	}
}

func TestSessionAdversaryAttackFlowAppliesFireRetaliation(t *testing.T) {
	var (
		damageRollReqs    []*pb.SessionDamageRollRequest
		retaliationDamage *pb.DaggerheartApplyAdversaryDamageRequest
	)
	handler := NewHandler(Dependencies{
		LoadAdversary: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				AdversaryID:      "adv-1",
				AdversaryEntryID: "adversary.goblin",
			}, nil
		},
		LoadAdversaryEntry: func(context.Context, string) (contentstore.DaggerheartAdversaryEntry, error) {
			return contentstore.DaggerheartAdversaryEntry{
				AttackModifier: 1,
				StandardAttack: contentstore.DaggerheartAdversaryAttack{
					Range:      "melee",
					DamageDice: []contentstore.DaggerheartDamageDie{{Sides: 6, Count: 1}},
					DamageType: "physical",
				},
			}, nil
		},
		LoadCharacterProfile: func(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
			return projectionstore.DaggerheartCharacterProfile{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
			}, nil
		},
		LoadCharacterState: func(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
			return projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Hp:          6,
				Armor:       2,
				SubclassState: &projectionstore.DaggerheartSubclassState{
					ElementalChannel: daggerheart.ElementalChannelFire,
				},
			}, nil
		},
		LoadSubclass: func(context.Context, string) (contentstore.DaggerheartSubclass, error) {
			return contentstore.DaggerheartSubclass{}, nil
		},
		SessionAdversaryAttackRoll: func(context.Context, *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
			return &pb.SessionAdversaryAttackRollResponse{RollSeq: 61}, nil
		},
		ApplyAdversaryAttackOutcome: func(context.Context, *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error) {
			return &pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
				Result: &pb.DaggerheartAdversaryAttackOutcomeResult{Success: true},
			}, nil
		},
		SessionDamageRoll: func(_ context.Context, in *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error) {
			cloned := *in
			damageRollReqs = append(damageRollReqs, &cloned)
			if len(damageRollReqs) == 1 {
				return &pb.SessionDamageRollResponse{RollSeq: 62, Total: 5}, nil
			}
			return &pb.SessionDamageRollResponse{RollSeq: 63, Total: 7}, nil
		},
		ApplyDamage: func(context.Context, *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error) {
			return &pb.DaggerheartApplyDamageResponse{
				CharacterId: "char-1",
				State: &pb.DaggerheartCharacterState{
					Hp:     5,
					Armor:  2,
					Stress: 0,
				},
			}, nil
		},
		ApplyAdversaryDamage: func(_ context.Context, in *pb.DaggerheartApplyAdversaryDamageRequest) (*pb.DaggerheartApplyAdversaryDamageResponse, error) {
			retaliationDamage = in
			return &pb.DaggerheartApplyAdversaryDamageResponse{
				AdversaryId: in.GetAdversaryId(),
				Adversary:   &pb.DaggerheartAdversary{Id: in.GetAdversaryId(), Hp: 3},
			}, nil
		},
	})

	_, err := handler.SessionAdversaryAttackFlow(context.Background(), &pb.SessionAdversaryAttackFlowRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		AdversaryId: "adv-1",
		TargetId:    "char-1",
		Difficulty:  10,
		Damage: &pb.DaggerheartAttackDamageSpec{
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("SessionAdversaryAttackFlow returned error: %v", err)
	}
	if len(damageRollReqs) != 2 {
		t.Fatalf("damage roll calls = %d, want 2", len(damageRollReqs))
	}
	if got := damageRollReqs[1].GetDice(); len(got) != 1 || got[0].GetSides() != 10 || got[0].GetCount() != 1 {
		t.Fatalf("retaliation dice = %+v, want 1d10", got)
	}
	if retaliationDamage == nil {
		t.Fatal("expected retaliation adversary damage")
	}
	if retaliationDamage.GetDamage().GetDamageType() != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC {
		t.Fatalf("retaliation damage type = %v, want magic", retaliationDamage.GetDamage().GetDamageType())
	}
	if retaliationDamage.GetDamage().GetAmount() != 7 {
		t.Fatalf("retaliation amount = %d, want 7", retaliationDamage.GetDamage().GetAmount())
	}
}
