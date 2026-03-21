package charactertransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestGetCharacterCreationProgress_Success(t *testing.T) {
	svc := newWorkflowCharacterService(t, projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
		ClassID:     "class.guardian",
		SubclassID:  "subclass.stalwart",
	})

	resp, err := svc.GetCharacterCreationProgress(gametest.ContextWithParticipantID("manager-1"), &statev1.GetCharacterCreationProgressRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	if err != nil {
		t.Fatalf("GetCharacterCreationProgress returned error: %v", err)
	}
	if resp.GetProgress() == nil {
		t.Fatal("expected non-nil progress")
	}
	if got := resp.GetProgress().GetNextStep(); got != 2 {
		t.Fatalf("next_step = %d, want 2", got)
	}
}

func TestResetCharacterCreationWorkflow_Success(t *testing.T) {
	svc := newWorkflowCharacterService(t, projectionstore.DaggerheartCharacterProfile{
		CampaignID:           "c1",
		CharacterID:          "ch1",
		Level:                1,
		HpMax:                6,
		StressMax:            6,
		Evasion:              10,
		ClassID:              "class.guardian",
		SubclassID:           "subclass.stalwart",
		Heritage:             testProjectionHeritage(),
		TraitsAssigned:       true,
		DetailsRecorded:      true,
		StartingWeaponIDs:    []string{"weapon.longsword"},
		StartingArmorID:      "armor.gambeson-armor",
		StartingPotionItemID: daggerheart.StartingPotionMinorHealthID,
		Agility:              2,
		Strength:             1,
		Finesse:              1,
		Instinct:             0,
		Presence:             0,
		Knowledge:            -1,
		Background:           "Watch captain",
		Connections:          "Trusted by the quartermaster",
		DomainCardIDs:        []string{"domain-card.ward"},
		Experiences:          []projectionstore.DaggerheartExperience{{Name: "Tactics", Modifier: 2}},
		MajorThreshold:       8,
		SevereThreshold:      12,
	})

	resp, err := svc.ResetCharacterCreationWorkflow(gametest.ContextWithParticipantID("manager-1"), &statev1.ResetCharacterCreationWorkflowRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
	})
	if err != nil {
		t.Fatalf("ResetCharacterCreationWorkflow returned error: %v", err)
	}
	if resp.GetProgress().GetReady() {
		t.Fatal("ready = true, want false")
	}
	if got := resp.GetProgress().GetNextStep(); got != 1 {
		t.Fatalf("next_step = %d, want 1", got)
	}
}
