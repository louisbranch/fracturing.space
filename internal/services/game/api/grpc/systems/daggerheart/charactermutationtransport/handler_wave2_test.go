package charactermutationtransport

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTransformBeastformSuccessUsesEvolutionPayload(t *testing.T) {
	t.Parallel()

	store := &statefulDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Level:       3,
				ClassID:     "class.druid",
				StressMax:   6,
			},
		},
		stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
			"char-1": {
				{
					CharacterID: "char-1",
					Hope:        5,
					Stress:      1,
				},
				{
					CharacterID: "char-1",
					Hope:        2,
					Stress:      1,
					ClassState: projectionstore.DaggerheartClassState{
						ActiveBeastform: &projectionstore.DaggerheartActiveBeastformState{
							BeastformID:            "wolf",
							BaseTrait:              "agility",
							AttackTrait:            "instinct",
							EvolutionTraitOverride: "instinct",
						},
					},
				},
			},
		},
	}
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		Content: contentStoreStub{
			beastforms: map[string]contentstore.DaggerheartBeastformEntry{
				"wolf": {
					ID:    "wolf",
					Tier:  2,
					Trait: "agility",
					Attack: contentstore.DaggerheartBeastformAttack{
						Range:      "melee",
						DamageDice: []contentstore.DaggerheartDamageDie{{Count: 1, Sides: 8}},
					},
				},
			},
		},
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			return nil
		},
	})

	resp, err := handler.TransformBeastform(testContext(), &pb.DaggerheartTransformBeastformRequest{
		CampaignId:     "camp-1",
		CharacterId:    "char-1",
		SessionId:      "sess-explicit",
		BeastformId:    "wolf",
		UseEvolution:   true,
		EvolutionTrait: "instinct",
	})
	if err != nil {
		t.Fatalf("TransformBeastform() error = %v", err)
	}
	if resp.GetCharacterId() != "char-1" || resp.GetState() == nil {
		t.Fatalf("response = %#v", resp)
	}
	if commandInput.CommandType != commandids.DaggerheartBeastformTransform || commandInput.SessionID != "sess-explicit" {
		t.Fatalf("command input = %#v", commandInput)
	}

	var payload daggerheartpayload.BeastformTransformPayload
	decodeMutationPayload(t, commandInput.PayloadJSON, &payload)
	if string(payload.CharacterID) != "char-1" || payload.BeastformID != "wolf" || !payload.UseEvolution {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.HopeBefore == nil || *payload.HopeBefore != 5 || payload.HopeAfter == nil || *payload.HopeAfter != 2 {
		t.Fatalf("hope payload = %#v", payload)
	}
	if payload.ClassStateAfter == nil || payload.ClassStateAfter.ActiveBeastform == nil || payload.ClassStateAfter.ActiveBeastform.AttackTrait != "instinct" {
		t.Fatalf("class state after = %#v", payload.ClassStateAfter)
	}
}

func TestTransformBeastformRejectsNonDruidAndMissingEvolutionTrait(t *testing.T) {
	t.Parallel()

	t.Run("non druid", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{
			Daggerheart: &statefulDaggerheartStore{
				profiles: map[string]projectionstore.DaggerheartCharacterProfile{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Level: 2, ClassID: "class.guardian", StressMax: 6},
				},
			},
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error {
				t.Fatal("unexpected command execution")
				return nil
			},
		})

		_, err := handler.TransformBeastform(testContext(), &pb.DaggerheartTransformBeastformRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			BeastformId: "wolf",
		})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want FailedPrecondition", status.Code(err))
		}
	})

	t.Run("evolution requires trait", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{
			Daggerheart: &statefulDaggerheartStore{
				profiles: map[string]projectionstore.DaggerheartCharacterProfile{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Level: 2, ClassID: "class.druid", StressMax: 6},
				},
				stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
					"char-1": {{CharacterID: "char-1", Hope: 4}},
				},
			},
			Content: contentStoreStub{
				beastforms: map[string]contentstore.DaggerheartBeastformEntry{
					"wolf": {ID: "wolf", Tier: 1, Trait: "agility"},
				},
			},
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error {
				t.Fatal("unexpected command execution")
				return nil
			},
		})

		_, err := handler.TransformBeastform(testContext(), &pb.DaggerheartTransformBeastformRequest{
			CampaignId:   "camp-1",
			CharacterId:  "char-1",
			BeastformId:  "wolf",
			UseEvolution: true,
		})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want InvalidArgument", status.Code(err))
		}
	})
}

func TestDropBeastformSuccessAndInactiveFailure(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {CampaignID: "camp-1", CharacterID: "char-1", ClassID: "class.druid"},
			},
			stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
				"char-1": {
					{
						CharacterID: "char-1",
						ClassState: projectionstore.DaggerheartClassState{
							ActiveBeastform: &projectionstore.DaggerheartActiveBeastformState{BeastformID: "wolf", AttackTrait: "instinct"},
						},
					},
					{CharacterID: "char-1"},
				},
			},
		}
		var commandInput CharacterCommandInput
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				commandInput = in
				return nil
			},
		})

		resp, err := handler.DropBeastform(testContext(), &pb.DaggerheartDropBeastformRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			SessionId:   "sess-explicit",
		})
		if err != nil {
			t.Fatalf("DropBeastform() error = %v", err)
		}
		if resp.GetCharacterId() != "char-1" || resp.GetState() == nil {
			t.Fatalf("response = %#v", resp)
		}
		if commandInput.CommandType != commandids.DaggerheartBeastformDrop || commandInput.SessionID != "sess-explicit" {
			t.Fatalf("command input = %#v", commandInput)
		}

		var payload daggerheartpayload.BeastformDropPayload
		decodeMutationPayload(t, commandInput.PayloadJSON, &payload)
		if payload.BeastformID != "wolf" || payload.Source != "beastform.drop" {
			t.Fatalf("payload = %#v", payload)
		}
		if payload.ClassStateBefore == nil || payload.ClassStateBefore.ActiveBeastform == nil {
			t.Fatalf("class state before = %#v", payload.ClassStateBefore)
		}
		if payload.ClassStateAfter == nil || payload.ClassStateAfter.ActiveBeastform != nil {
			t.Fatalf("class state after = %#v", payload.ClassStateAfter)
		}
	})

	t.Run("not active", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{
			Daggerheart: &statefulDaggerheartStore{
				profiles: map[string]projectionstore.DaggerheartCharacterProfile{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", ClassID: "class.druid"},
				},
				stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
					"char-1": {{CharacterID: "char-1"}},
				},
			},
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error {
				t.Fatal("unexpected command execution")
				return nil
			},
		})

		_, err := handler.DropBeastform(testContext(), &pb.DaggerheartDropBeastformRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
		})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want FailedPrecondition", status.Code(err))
		}
	})
}

func TestCompanionHandlersShapePayloadsAndPreconditions(t *testing.T) {
	t.Parallel()

	t.Run("begin experience success", func(t *testing.T) {
		t.Parallel()

		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {
					CampaignID:  "camp-1",
					CharacterID: "char-1",
					CompanionSheet: &projectionstore.DaggerheartCompanionSheet{
						Experiences: []projectionstore.DaggerheartCompanionExperience{{ExperienceID: "exp-1"}},
					},
				},
			},
			stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
				"char-1": {
					{CharacterID: "char-1"},
					{CharacterID: "char-1", CompanionState: &projectionstore.DaggerheartCompanionState{Status: "away", ActiveExperienceID: "exp-1"}},
				},
			},
		}
		var commandInput CharacterCommandInput
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				commandInput = in
				return nil
			},
		})

		resp, err := handler.BeginCompanionExperience(testContext(), &pb.DaggerheartBeginCompanionExperienceRequest{
			CampaignId:   "camp-1",
			CharacterId:  "char-1",
			SessionId:    "sess-explicit",
			ExperienceId: "exp-1",
		})
		if err != nil {
			t.Fatalf("BeginCompanionExperience() error = %v", err)
		}
		if resp.GetCharacterId() != "char-1" || resp.GetState() == nil {
			t.Fatalf("response = %#v", resp)
		}
		if commandInput.CommandType != commandids.DaggerheartCompanionExperienceBegin || commandInput.SessionID != "sess-explicit" {
			t.Fatalf("command input = %#v", commandInput)
		}

		var payload daggerheartpayload.CompanionExperienceBeginPayload
		decodeMutationPayload(t, commandInput.PayloadJSON, &payload)
		if payload.ExperienceID != "exp-1" || payload.CompanionStateBefore == nil || payload.CompanionStateBefore.Status != "present" {
			t.Fatalf("payload = %#v", payload)
		}
		if payload.CompanionStateAfter == nil || payload.CompanionStateAfter.Status != "away" || payload.CompanionStateAfter.ActiveExperienceID != "exp-1" {
			t.Fatalf("companion state after = %#v", payload.CompanionStateAfter)
		}
	})

	t.Run("begin experience rejects missing experience on sheet", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{
			Daggerheart: &statefulDaggerheartStore{
				profiles: map[string]projectionstore.DaggerheartCharacterProfile{
					"char-1": {
						CampaignID:  "camp-1",
						CharacterID: "char-1",
						CompanionSheet: &projectionstore.DaggerheartCompanionSheet{
							Experiences: []projectionstore.DaggerheartCompanionExperience{{ExperienceID: "exp-2"}},
						},
					},
				},
			},
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error {
				t.Fatal("unexpected command execution")
				return nil
			},
		})

		_, err := handler.BeginCompanionExperience(testContext(), &pb.DaggerheartBeginCompanionExperienceRequest{
			CampaignId:   "camp-1",
			CharacterId:  "char-1",
			ExperienceId: "exp-1",
		})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want FailedPrecondition", status.Code(err))
		}
	})

	t.Run("return companion success reduces stress", func(t *testing.T) {
		t.Parallel()

		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {
					CampaignID:  "camp-1",
					CharacterID: "char-1",
					CompanionSheet: &projectionstore.DaggerheartCompanionSheet{
						Experiences: []projectionstore.DaggerheartCompanionExperience{{ExperienceID: "exp-1"}},
					},
				},
			},
			stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
				"char-1": {
					{
						CharacterID: "char-1",
						Stress:      2,
						CompanionState: &projectionstore.DaggerheartCompanionState{
							Status:             "away",
							ActiveExperienceID: "exp-1",
						},
					},
					{
						CharacterID:    "char-1",
						Stress:         1,
						CompanionState: &projectionstore.DaggerheartCompanionState{Status: "present"},
					},
				},
			},
		}
		var commandInput CharacterCommandInput
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				commandInput = in
				return nil
			},
		})

		resp, err := handler.ReturnCompanion(testContext(), &pb.DaggerheartReturnCompanionRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			SessionId:   "sess-explicit",
			Resolution:  pb.DaggerheartCompanionReturnResolution_DAGGERHEART_COMPANION_RETURN_RESOLUTION_EXPERIENCE_COMPLETED,
		})
		if err != nil {
			t.Fatalf("ReturnCompanion() error = %v", err)
		}
		if resp.GetCharacterId() != "char-1" || resp.GetState() == nil {
			t.Fatalf("response = %#v", resp)
		}

		var payload daggerheartpayload.CompanionReturnPayload
		decodeMutationPayload(t, commandInput.PayloadJSON, &payload)
		if payload.Resolution != "experience_completed" || payload.StressBefore == nil || *payload.StressBefore != 2 || payload.StressAfter == nil || *payload.StressAfter != 1 {
			t.Fatalf("payload = %#v", payload)
		}
		if payload.CompanionStateAfter == nil || payload.CompanionStateAfter.Status != "present" {
			t.Fatalf("companion state after = %#v", payload.CompanionStateAfter)
		}
	})

	t.Run("return companion rejects when not away", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{
			Daggerheart: &statefulDaggerheartStore{
				profiles: map[string]projectionstore.DaggerheartCharacterProfile{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1", CompanionSheet: &projectionstore.DaggerheartCompanionSheet{}},
				},
				stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
					"char-1": {{CharacterID: "char-1", CompanionState: &projectionstore.DaggerheartCompanionState{Status: "present"}}},
				},
			},
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error {
				t.Fatal("unexpected command execution")
				return nil
			},
		})

		_, err := handler.ReturnCompanion(testContext(), &pb.DaggerheartReturnCompanionRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			Resolution:  pb.DaggerheartCompanionReturnResolution_DAGGERHEART_COMPANION_RETURN_RESOLUTION_EARLY_RETURN,
		})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want FailedPrecondition", status.Code(err))
		}
	})
}

func TestEquipmentConsumableAndLevelUpHandlers(t *testing.T) {
	t.Parallel()

	t.Run("swap equipment armor enriches payload", func(t *testing.T) {
		t.Parallel()

		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {
					CampaignID:      "camp-1",
					CharacterID:     "char-1",
					Level:           2,
					ArmorMax:        2,
					ArmorScore:      1,
					Evasion:         11,
					MajorThreshold:  5,
					SevereThreshold: 10,
					EquippedArmorID: "armor.old",
				},
			},
			stateSeqs: map[string][]projectionstore.DaggerheartCharacterState{
				"char-1": {{CharacterID: "char-1", Armor: 1}},
			},
		}
		var commandInput CharacterCommandInput
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			Content: contentStoreStub{
				armors: map[string]contentstore.DaggerheartArmor{
					"armor.old": {ID: "armor.old", ArmorScore: 1, BaseMajorThreshold: 5, BaseSevereThreshold: 10},
					"armor.new": {ID: "armor.new", ArmorScore: 3, BaseMajorThreshold: 6, BaseSevereThreshold: 11},
				},
			},
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				commandInput = in
				return nil
			},
		})

		resp, err := handler.SwapEquipment(testContext(), &pb.DaggerheartSwapEquipmentRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			ItemId:      "armor.new",
			ItemType:    "armor",
			From:        "inventory",
			To:          "active",
		})
		if err != nil {
			t.Fatalf("SwapEquipment() error = %v", err)
		}
		if resp.GetCharacterId() != "char-1" {
			t.Fatalf("response = %#v", resp)
		}

		var payload daggerheartpayload.EquipmentSwapPayload
		decodeMutationPayload(t, commandInput.PayloadJSON, &payload)
		if payload.ItemID != "armor.new" || payload.EquippedArmorID != "armor.new" {
			t.Fatalf("payload = %#v", payload)
		}
		if payload.ArmorAfter == nil || payload.ArmorMaxAfter == nil || payload.ArmorScoreAfter == nil {
			t.Fatalf("armor enrichment missing = %#v", payload)
		}
	})

	t.Run("swap equipment armor requires content store", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{
			Daggerheart: &statefulDaggerheartStore{
				profiles: map[string]projectionstore.DaggerheartCharacterProfile{
					"char-1": {CampaignID: "camp-1", CharacterID: "char-1"},
				},
			},
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error {
				t.Fatal("unexpected command execution")
				return nil
			},
		})

		_, err := handler.SwapEquipment(testContext(), &pb.DaggerheartSwapEquipmentRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			ItemId:      "armor.new",
			ItemType:    "armor",
			To:          "active",
		})
		if status.Code(err) != codes.Internal {
			t.Fatalf("status code = %v, want Internal", status.Code(err))
		}
	})

	t.Run("consumable payloads are shaped", func(t *testing.T) {
		t.Parallel()

		handler := newTestHandler(Dependencies{
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				switch in.CommandType {
				case commandids.DaggerheartConsumableUse:
					var payload daggerheartpayload.ConsumableUsePayload
					decodeMutationPayload(t, in.PayloadJSON, &payload)
					if payload.ConsumableID != "cons-1" || payload.QuantityBefore != 2 || payload.QuantityAfter != 1 {
						t.Fatalf("use payload = %#v", payload)
					}
				case commandids.DaggerheartConsumableAcquire:
					var payload daggerheartpayload.ConsumableAcquirePayload
					decodeMutationPayload(t, in.PayloadJSON, &payload)
					if payload.ConsumableID != "cons-1" || payload.QuantityBefore != 1 || payload.QuantityAfter != 3 {
						t.Fatalf("acquire payload = %#v", payload)
					}
				default:
					t.Fatalf("unexpected command type %v", in.CommandType)
				}
				return nil
			},
		})

		if _, err := handler.UseConsumable(testContext(), &pb.DaggerheartUseConsumableRequest{
			CampaignId:     "camp-1",
			CharacterId:    "char-1",
			ConsumableId:   "cons-1",
			QuantityBefore: 2,
			QuantityAfter:  1,
		}); err != nil {
			t.Fatalf("UseConsumable() error = %v", err)
		}
		if _, err := handler.AcquireConsumable(testContext(), &pb.DaggerheartAcquireConsumableRequest{
			CampaignId:     "camp-1",
			CharacterId:    "char-1",
			ConsumableId:   "cons-1",
			QuantityBefore: 1,
			QuantityAfter:  3,
		}); err != nil {
			t.Fatalf("AcquireConsumable() error = %v", err)
		}
	})

	t.Run("domain card and gold payloads are shaped", func(t *testing.T) {
		t.Parallel()

		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {
					CampaignID:   "camp-1",
					CharacterID:  "char-1",
					GoldHandfuls: 4,
					GoldBags:     5,
					GoldChests:   6,
				},
			},
		}
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				switch in.CommandType {
				case commandids.DaggerheartDomainCardAcquire:
					var payload daggerheartpayload.DomainCardAcquirePayload
					decodeMutationPayload(t, in.PayloadJSON, &payload)
					if payload.CardID != "card-1" || payload.CardLevel != 2 || payload.Destination != "vault" {
						t.Fatalf("domain card payload = %#v", payload)
					}
				case commandids.DaggerheartGoldUpdate:
					var payload daggerheartpayload.GoldUpdatePayload
					decodeMutationPayload(t, in.PayloadJSON, &payload)
					if payload.HandfulsBefore != 1 || payload.HandfulsAfter != 4 || payload.BagsAfter != 5 || payload.ChestsAfter != 6 || payload.Reason != "loot" {
						t.Fatalf("gold payload = %#v", payload)
					}
					store.profiles["char-1"] = projectionstore.DaggerheartCharacterProfile{
						CampaignID:   "camp-1",
						CharacterID:  "char-1",
						GoldHandfuls: 4,
						GoldBags:     5,
						GoldChests:   6,
					}
				default:
					t.Fatalf("unexpected command type %v", in.CommandType)
				}
				return nil
			},
		})

		cardResp, err := handler.AcquireDomainCard(testContext(), &pb.DaggerheartAcquireDomainCardRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			CardId:      "card-1",
			CardLevel:   2,
		})
		if err != nil {
			t.Fatalf("AcquireDomainCard() error = %v", err)
		}
		if cardResp.GetCharacterId() != "char-1" {
			t.Fatalf("domain card response = %#v", cardResp)
		}

		goldResp, err := handler.UpdateGold(testContext(), &pb.DaggerheartUpdateGoldRequest{
			CampaignId:     "camp-1",
			CharacterId:    "char-1",
			HandfulsBefore: 1,
			HandfulsAfter:  4,
			BagsBefore:     2,
			BagsAfter:      5,
			ChestsBefore:   3,
			ChestsAfter:    6,
			Reason:         "loot",
		})
		if err != nil {
			t.Fatalf("UpdateGold() error = %v", err)
		}
		if goldResp.GetHandfuls() != 4 || goldResp.GetBags() != 5 || goldResp.GetChests() != 6 {
			t.Fatalf("gold response = %#v", goldResp)
		}
	})

	t.Run("level up payload and campaign guard", func(t *testing.T) {
		t.Parallel()

		store := &statefulDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Level: 1},
			},
		}
		var commandInput CharacterCommandInput
		handler := newTestHandler(Dependencies{
			Daggerheart: store,
			ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
				commandInput = in
				store.profiles["char-1"] = projectionstore.DaggerheartCharacterProfile{
					CampaignID:  "camp-1",
					CharacterID: "char-1",
					Level:       2,
				}
				return nil
			},
		})

		resp, err := handler.ApplyLevelUp(testContext(), &pb.DaggerheartApplyLevelUpRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			LevelAfter:  2,
			Advancements: []*pb.DaggerheartLevelUpAdvancement{
				{Type: "trait", Trait: "agility"},
			},
			Rewards: []*pb.DaggerheartLevelUpReward{
				{Type: "companion_bonus", CompanionBonusChoices: 1},
			},
		})
		if err != nil {
			t.Fatalf("ApplyLevelUp() error = %v", err)
		}
		if resp.GetLevel() != 2 || resp.GetTier() != 2 {
			t.Fatalf("response = %#v", resp)
		}
		if commandInput.CommandType != commandids.DaggerheartLevelUpApply || commandInput.SessionID != "sess-1" {
			t.Fatalf("command input = %#v", commandInput)
		}

		var payload daggerheartpayload.LevelUpApplyPayload
		decodeMutationPayload(t, commandInput.PayloadJSON, &payload)
		if payload.LevelBefore != 1 || payload.LevelAfter != 2 || len(payload.Advancements) != 1 || payload.Advancements[0].Trait != "agility" {
			t.Fatalf("payload = %#v", payload)
		}
		if len(payload.Rewards) != 1 || payload.Rewards[0].CompanionBonusChoices != 1 {
			t.Fatalf("rewards = %#v", payload.Rewards)
		}

		guarded := NewHandler(Dependencies{
			Campaign: testCampaignStore{record: storage.CampaignRecord{
				ID:     "camp-1",
				System: systembridge.SystemID("other"),
				Status: campaign.StatusActive,
			}},
			Daggerheart: store,
			ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error {
				t.Fatal("unexpected command execution")
				return nil
			},
		})
		_, err = guarded.ApplyLevelUp(testContext(), &pb.DaggerheartApplyLevelUpRequest{
			CampaignId:  "camp-1",
			CharacterId: "char-1",
			LevelAfter:  2,
		})
		if status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("guard status code = %v, want FailedPrecondition", status.Code(err))
		}
	})
}

func TestNilRequestGuards(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(Dependencies{})
	testCases := []struct {
		name string
		err  error
	}{
		{name: "use consumable", err: func() error {
			_, err := handler.UseConsumable(testContext(), nil)
			return err
		}()},
		{name: "acquire consumable", err: func() error {
			_, err := handler.AcquireConsumable(testContext(), nil)
			return err
		}()},
		{name: "acquire domain card", err: func() error {
			_, err := handler.AcquireDomainCard(testContext(), nil)
			return err
		}()},
		{name: "update gold", err: func() error {
			_, err := handler.UpdateGold(testContext(), nil)
			return err
		}()},
		{name: "apply class feature", err: func() error {
			_, err := handler.ApplyClassFeature(testContext(), nil)
			return err
		}()},
		{name: "apply subclass feature", err: func() error {
			_, err := handler.ApplySubclassFeature(testContext(), nil)
			return err
		}()},
	}

	for _, tc := range testCases {
		if status.Code(tc.err) != codes.InvalidArgument {
			t.Fatalf("%s status code = %v, want InvalidArgument", tc.name, status.Code(tc.err))
		}
	}
}

func decodeMutationPayload(t *testing.T, data []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", string(data), err)
	}
}
