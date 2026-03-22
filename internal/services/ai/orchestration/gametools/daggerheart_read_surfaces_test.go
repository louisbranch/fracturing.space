package gametools

import (
	"fmt"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func TestBuildCharacterSheetPayloadIncludesCapabilitiesAndState(t *testing.T) {
	payload := buildCharacterSheetPayload(&statev1.GetCharacterSheetResponse{
		Character: &statev1.Character{
			Id:            "char-1",
			CampaignId:    "camp-1",
			Name:          "Aria",
			Kind:          statev1.CharacterKind_PC,
			ParticipantId: wrapperspb.String("player-1"),
			Aliases:       []string{"The Lantern"},
			Notes:         "Keeps the harbor oath.",
		},
		Profile: &statev1.CharacterProfile{
			SystemProfile: &statev1.CharacterProfile_Daggerheart{
				Daggerheart: &pb.DaggerheartProfile{
					Level:         1,
					ClassId:       "class.guardian",
					SubclassId:    "subclass.stalwart",
					HpMax:         10,
					StressMax:     wrapperspb.Int32(6),
					ArmorMax:      wrapperspb.Int32(3),
					Evasion:       wrapperspb.Int32(12),
					Agility:       wrapperspb.Int32(2),
					Strength:      wrapperspb.Int32(1),
					Heritage:      &pb.DaggerheartHeritageSelection{AncestryName: "Human", CommunityName: "Highborne"},
					DomainCardIds: []string{"domain_card.valor-shield-wall"},
					ActiveClassFeatures: []*pb.DaggerheartActiveClassFeature{{
						Id:          "feature.guardian.hold-the-line",
						Name:        "Hold the Line",
						Description: "Protect allies in the breach.",
						HopeFeature: true,
					}},
					PrimaryWeapon: &pb.DaggerheartSheetWeaponSummary{
						Id:         "weapon.longsword",
						Name:       "Longsword",
						Trait:      "Strength",
						DamageDice: "1d10",
					},
					ActiveArmor: &pb.DaggerheartSheetArmorSummary{
						Id:        "armor.gambeson-armor",
						Name:      "Gambeson Armor",
						BaseScore: 2,
					},
					StartingPotionItemId: "item.minor-health-potion",
					CompanionSheet: &pb.DaggerheartCompanionSheet{
						Name: "Moss",
					},
				},
			},
		},
		State: &statev1.CharacterState{
			SystemState: &statev1.CharacterState_Daggerheart{
				Daggerheart: &pb.DaggerheartCharacterState{
					Hp:        8,
					Hope:      3,
					HopeMax:   6,
					Stress:    2,
					Armor:     1,
					LifeState: pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
					ConditionStates: []*pb.DaggerheartConditionState{{
						Label: "Vulnerable",
					}},
					ClassState: &pb.DaggerheartClassState{
						ActiveBeastform: &pb.DaggerheartActiveBeastformState{
							BaseTrait:   "Instinct",
							AttackTrait: "Strength",
						},
					},
					CompanionState: &pb.DaggerheartCompanionState{
						Status: "ready",
					},
				},
			},
		},
	})

	if payload.Character.Name != "Aria" {
		t.Fatalf("character name = %q, want Aria", payload.Character.Name)
	}
	if payload.Daggerheart == nil {
		t.Fatal("expected daggerheart payload")
	}
	if payload.Daggerheart.Class == nil || payload.Daggerheart.Class.Name != "Guardian" {
		t.Fatalf("class = %#v", payload.Daggerheart.Class)
	}
	if got := payload.Daggerheart.Resources.Hope; got != 3 {
		t.Fatalf("resources.hope = %d, want 3", got)
	}
	if len(payload.Daggerheart.DomainCards) != 1 || payload.Daggerheart.DomainCards[0].Name != "Shield Wall" {
		t.Fatalf("domain cards = %#v", payload.Daggerheart.DomainCards)
	}
	if len(payload.Daggerheart.ActiveClassFeatures) != 1 || !payload.Daggerheart.ActiveClassFeatures[0].HopeFeature {
		t.Fatalf("active class features = %#v", payload.Daggerheart.ActiveClassFeatures)
	}
	if payload.Daggerheart.Equipment == nil || payload.Daggerheart.Equipment.PrimaryWeapon == nil || payload.Daggerheart.Equipment.PrimaryWeapon.Name != "Longsword" {
		t.Fatalf("equipment = %#v", payload.Daggerheart.Equipment)
	}
	if payload.Daggerheart.Companion == nil || payload.Daggerheart.Companion.Status != "ready" {
		t.Fatalf("companion = %#v", payload.Daggerheart.Companion)
	}
	if payload.Daggerheart.ClassState == nil || payload.Daggerheart.ClassState.ActiveBeastform == nil {
		t.Fatalf("class state = %#v", payload.Daggerheart.ClassState)
	}
}

func TestBuildDaggerheartCombatBoardPayloadIncludesFearSpotlightAndAdversaries(t *testing.T) {
	payload := applyCombatBoardDiagnostics(buildDaggerheartCombatBoardPayload(
		"sess-1",
		"scene-1",
		&statev1.GetSnapshotResponse{
			Snapshot: &statev1.Snapshot{
				SystemSnapshot: &statev1.Snapshot_Daggerheart{
					Daggerheart: &pb.DaggerheartSnapshot{GmFear: 4},
				},
			},
		},
		&statev1.GetSessionSpotlightResponse{
			Spotlight: &statev1.SessionSpotlight{
				Type:        statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER,
				CharacterId: "char-1",
			},
		},
		&pb.DaggerheartListSceneCountdownsResponse{
			Countdowns: []*pb.DaggerheartSceneCountdown{{
				CountdownId:       "cd-1",
				SessionId:         "sess-1",
				SceneId:           "scene-1",
				Name:              "Breach",
				Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE,
				AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
				StartingValue:     4,
				RemainingValue:    2,
				LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
				Status:            pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
			}},
		},
		&pb.DaggerheartListAdversariesResponse{
			Adversaries: []*pb.DaggerheartAdversary{{
				Id:             "adv-1",
				Name:           "Bandit",
				SceneId:        "scene-1",
				Hp:             5,
				SpotlightCount: 1,
				FeatureStates: []*pb.DaggerheartAdversaryFeatureState{{
					FeatureId: "feature.volley",
					Status:    pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_READY,
				}},
			}, {
				Id:      "adv-offscene",
				Name:    "Offscreen Threat",
				SceneId: "scene-2",
			}},
		},
	), nil)

	if payload.GmFear != 4 {
		t.Fatalf("gm_fear = %d, want 4", payload.GmFear)
	}
	if payload.Status != "READY" {
		t.Fatalf("status = %q, want READY", payload.Status)
	}
	if payload.Spotlight == nil || payload.Spotlight.Type != "CHARACTER" {
		t.Fatalf("spotlight = %#v", payload.Spotlight)
	}
	if payload.SceneID != "scene-1" {
		t.Fatalf("scene_id = %q, want scene-1", payload.SceneID)
	}
	if len(payload.Countdowns) != 1 || payload.Countdowns[0].ID != "cd-1" || payload.Countdowns[0].Name != "Breach" {
		t.Fatalf("countdowns = %#v", payload.Countdowns)
	}
	if len(payload.Adversaries) != 1 || payload.Adversaries[0].Name != "Bandit" {
		t.Fatalf("adversaries = %#v", payload.Adversaries)
	}
	if len(payload.Adversaries[0].Features) != 1 || payload.Adversaries[0].Features[0].Status != "READY" {
		t.Fatalf("adversary features = %#v", payload.Adversaries[0].Features)
	}
}

func TestApplyCombatBoardDiagnosticsNoActiveScene(t *testing.T) {
	payload := applyCombatBoardDiagnostics(daggerheartCombatBoardPayload{
		SessionID: "sess-1",
		GmFear:    1,
	}, fmt.Errorf("no active scene"))

	if payload.Status != "NO_ACTIVE_SCENE" {
		t.Fatalf("status = %q, want NO_ACTIVE_SCENE", payload.Status)
	}
	if len(payload.Issues) != 1 || payload.Issues[0].Code != "no_active_scene" {
		t.Fatalf("issues = %#v", payload.Issues)
	}
	if len(payload.RecommendedTools) == 0 || payload.RecommendedTools[0] != "interaction_state_read" {
		t.Fatalf("recommended_tools = %#v", payload.RecommendedTools)
	}
}

func TestApplyCombatBoardDiagnosticsEmptyBoardAndNoVisibleAdversary(t *testing.T) {
	emptyBoard := applyCombatBoardDiagnostics(daggerheartCombatBoardPayload{
		SessionID: "sess-1",
		SceneID:   "scene-1",
	}, nil)
	if emptyBoard.Status != "EMPTY_BOARD" {
		t.Fatalf("empty board status = %q, want EMPTY_BOARD", emptyBoard.Status)
	}

	countdownOnly := applyCombatBoardDiagnostics(daggerheartCombatBoardPayload{
		SessionID:  "sess-1",
		SceneID:    "scene-1",
		Countdowns: []countdownSummary{{ID: "cd-1"}},
	}, nil)
	if countdownOnly.Status != "NO_VISIBLE_ADVERSARY" {
		t.Fatalf("countdown-only status = %q, want NO_VISIBLE_ADVERSARY", countdownOnly.Status)
	}
}
