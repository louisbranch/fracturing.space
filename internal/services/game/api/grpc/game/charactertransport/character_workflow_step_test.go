package charactertransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
)

func TestApplyCharacterCreationStep_RequiresNextStep(t *testing.T) {
	svc := newWorkflowCharacterService(t, projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
	})

	_, err := svc.ApplyCharacterCreationStep(gametest.ContextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStep: &statev1.ApplyCharacterCreationStepRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{
					HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{Heritage: testCreationHeritageInput()},
				},
			},
		},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyCharacterCreationStep_ClassStepSuccess(t *testing.T) {
	svc := newWorkflowCharacterService(t, projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
	})

	resp, err := svc.ApplyCharacterCreationStep(gametest.ContextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStep: &statev1.ApplyCharacterCreationStepRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput{
					ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{ClassId: "class.guardian", SubclassId: "subclass.stalwart"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyCharacterCreationStep returned error: %v", err)
	}
	if resp.GetProfile().GetDaggerheart().GetClassId() != "class.guardian" {
		t.Fatalf("class_id = %q, want %q", resp.GetProfile().GetDaggerheart().GetClassId(), "class.guardian")
	}
	if resp.GetProfile().GetDaggerheart().GetSubclassId() != "subclass.stalwart" {
		t.Fatalf("subclass_id = %q, want %q", resp.GetProfile().GetDaggerheart().GetSubclassId(), "subclass.stalwart")
	}
	if got := resp.GetProgress().GetNextStep(); got != 2 {
		t.Fatalf("next_step = %d, want 2", got)
	}
}

func TestApplyCharacterCreationStep_TraitsRejectsInvalidDistribution(t *testing.T) {
	svc := newWorkflowCharacterService(t, projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
		ClassID:     "class.guardian",
		SubclassID:  "subclass.stalwart",
		Heritage:    testProjectionHeritage(),
	})

	_, err := svc.ApplyCharacterCreationStep(gametest.ContextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStep: &statev1.ApplyCharacterCreationStepRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_TraitsInput{
					TraitsInput: &daggerheartv1.DaggerheartCreationStepTraitsInput{Agility: 0, Strength: 0, Finesse: 0, Instinct: 0, Presence: 0, Knowledge: 0},
				},
			},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyCharacterCreationStep_EquipmentRejectsInvalidPotion(t *testing.T) {
	svc := newWorkflowCharacterService(t, projectionstore.DaggerheartCharacterProfile{
		CampaignID:      "c1",
		CharacterID:     "ch1",
		Level:           1,
		HpMax:           7,
		StressMax:       6,
		Evasion:         9,
		MajorThreshold:  8,
		SevereThreshold: 12,
		ClassID:         "class.guardian",
		SubclassID:      "subclass.stalwart",
		Heritage:        testProjectionHeritage(),
		TraitsAssigned:  true,
		DetailsRecorded: true,
		Agility:         2,
		Strength:        1,
		Finesse:         1,
		Instinct:        0,
		Presence:        0,
		Knowledge:       -1,
	})

	_, err := svc.ApplyCharacterCreationStep(gametest.ContextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStep: &statev1.ApplyCharacterCreationStepRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{
					EquipmentInput: &daggerheartv1.DaggerheartCreationStepEquipmentInput{
						WeaponIds:    []string{"weapon.longsword"},
						ArmorId:      "armor.gambeson-armor",
						PotionItemId: "item.not-a-starting-potion",
					},
				},
			},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyCharacterCreationStep_DomainCardsRejectsClassDomainMismatch(t *testing.T) {
	svc := newWorkflowCharacterService(t, projectionstore.DaggerheartCharacterProfile{
		CampaignID:           "c1",
		CharacterID:          "ch1",
		Level:                1,
		HpMax:                7,
		StressMax:            6,
		Evasion:              9,
		MajorThreshold:       9,
		SevereThreshold:      14,
		Proficiency:          1,
		ArmorScore:           1,
		ArmorMax:             1,
		ClassID:              "class.guardian",
		SubclassID:           "subclass.stalwart",
		Heritage:             testProjectionHeritage(),
		TraitsAssigned:       true,
		DetailsRecorded:      true,
		StartingWeaponIDs:    []string{"weapon.longsword"},
		StartingArmorID:      "armor.gambeson-armor",
		EquippedArmorID:      "armor.gambeson-armor",
		StartingPotionItemID: daggerheart.StartingPotionMinorHealthID,
		Agility:              2,
		Strength:             1,
		Finesse:              1,
		Instinct:             0,
		Presence:             0,
		Knowledge:            -1,
		Background:           "Watch captain",
		Experiences:          []projectionstore.DaggerheartExperience{{Name: "Shield wall", Modifier: 2}, {Name: "Patrol routes", Modifier: 2}},
	})

	_, err := svc.ApplyCharacterCreationStep(gametest.ContextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStep: &statev1.ApplyCharacterCreationStepRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{
					DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{DomainCardIds: []string{"domain-card.arcana-bolt", "domain-card.arcana-bolt"}},
				},
			},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}
