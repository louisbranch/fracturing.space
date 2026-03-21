package charactertransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestApplyCharacterCreationWorkflow_Success(t *testing.T) {
	svc := newWorkflowCharacterService(t, projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
	})

	resp, err := svc.ApplyCharacterCreationWorkflow(gametest.ContextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationWorkflowRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemWorkflow: &statev1.ApplyCharacterCreationWorkflowRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationWorkflowInput{
				ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{ClassId: "class.guardian", SubclassId: "subclass.stalwart"},
				HeritageInput:      &daggerheartv1.DaggerheartCreationStepHeritageInput{Heritage: testCreationHeritageInput()},
				TraitsInput:        &daggerheartv1.DaggerheartCreationStepTraitsInput{Agility: 2, Strength: 1, Finesse: 1, Instinct: 0, Presence: 0, Knowledge: -1},
				DetailsInput:       &daggerheartv1.DaggerheartCreationStepDetailsInput{Description: "A stalwart guardian of the realm."},
				EquipmentInput:     &daggerheartv1.DaggerheartCreationStepEquipmentInput{WeaponIds: []string{"weapon.longsword"}, ArmorId: "armor.gambeson-armor", PotionItemId: "item.minor-health-potion"},
				BackgroundInput:    &daggerheartv1.DaggerheartCreationStepBackgroundInput{Background: "City watch veteran"},
				ExperiencesInput: &daggerheartv1.DaggerheartCreationStepExperiencesInput{Experiences: []*daggerheartv1.DaggerheartExperience{
					{Name: "Shield wall", Modifier: 2},
					{Name: "Patrol routes", Modifier: 2},
				}},
				DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{DomainCardIds: []string{"domain-card.ward", "domain-card.blade-strike"}},
				ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{Connections: "Trusted by the quartermaster"},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyCharacterCreationWorkflow returned error: %v", err)
	}
	if resp.GetProfile().GetDaggerheart().GetClassId() != "class.guardian" {
		t.Fatalf("class_id = %q, want %q", resp.GetProfile().GetDaggerheart().GetClassId(), "class.guardian")
	}
	if got := resp.GetProfile().GetDaggerheart().GetEquippedArmorId(); got != "armor.gambeson-armor" {
		t.Fatalf("equipped_armor_id = %q, want %q", got, "armor.gambeson-armor")
	}
	if got := resp.GetProfile().GetDaggerheart().GetArmorScore().GetValue(); got != 1 {
		t.Fatalf("armor_score = %d, want 1", got)
	}
	if got := resp.GetProfile().GetDaggerheart().GetArmorMax().GetValue(); got != 1 {
		t.Fatalf("armor_max = %d, want 1", got)
	}
	if !resp.GetProgress().GetReady() {
		t.Fatal("ready = false, want true")
	}
	if got := resp.GetProgress().GetNextStep(); got != 0 {
		t.Fatalf("next_step = %d, want 0", got)
	}
}
