package game

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestGetCharacterCreationProgress_Success(t *testing.T) {
	svc := newWorkflowCharacterService(t, storage.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
		ClassID:     "class.guardian",
		SubclassID:  "subclass.stalwart",
	})

	resp, err := svc.GetCharacterCreationProgress(contextWithParticipantID("manager-1"), &statev1.GetCharacterCreationProgressRequest{
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

func TestApplyCharacterCreationStep_RequiresNextStep(t *testing.T) {
	svc := newWorkflowCharacterService(t, storage.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
	})

	_, err := svc.ApplyCharacterCreationStep(contextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStep: &statev1.ApplyCharacterCreationStepRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{
					HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{AncestryId: "heritage.ancestry.clank", CommunityId: "heritage.community.farmer"},
				},
			},
		},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyCharacterCreationStep_ClassStepSuccess(t *testing.T) {
	svc := newWorkflowCharacterService(t, storage.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
	})

	resp, err := svc.ApplyCharacterCreationStep(contextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
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
	svc := newWorkflowCharacterService(t, storage.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
		ClassID:     "class.guardian",
		SubclassID:  "subclass.stalwart",
		AncestryID:  "heritage.ancestry.clank",
		CommunityID: "heritage.community.farmer",
	})

	_, err := svc.ApplyCharacterCreationStep(contextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
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
	svc := newWorkflowCharacterService(t, storage.DaggerheartCharacterProfile{
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
		AncestryID:      "heritage.ancestry.clank",
		CommunityID:     "heritage.community.farmer",
		TraitsAssigned:  true,
		DetailsRecorded: true,
		Agility:         2,
		Strength:        1,
		Finesse:         1,
		Instinct:        0,
		Presence:        0,
		Knowledge:       -1,
	})

	_, err := svc.ApplyCharacterCreationStep(contextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
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
	svc := newWorkflowCharacterService(t, storage.DaggerheartCharacterProfile{
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
		ArmorMax:             0,
		ClassID:              "class.guardian",
		SubclassID:           "subclass.stalwart",
		AncestryID:           "heritage.ancestry.clank",
		CommunityID:          "heritage.community.farmer",
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
		Experiences:          []storage.DaggerheartExperience{{Name: "Shield wall", Modifier: 2}},
	})

	_, err := svc.ApplyCharacterCreationStep(contextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemStep: &statev1.ApplyCharacterCreationStepRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationStepInput{
				Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{
					DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{DomainCardIds: []string{"domain-card.arcana-bolt"}},
				},
			},
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestResetCharacterCreationWorkflow_Success(t *testing.T) {
	svc := newWorkflowCharacterService(t, storage.DaggerheartCharacterProfile{
		CampaignID:           "c1",
		CharacterID:          "ch1",
		Level:                1,
		HpMax:                6,
		StressMax:            6,
		Evasion:              10,
		ClassID:              "class.guardian",
		SubclassID:           "subclass.stalwart",
		AncestryID:           "heritage.clank",
		CommunityID:          "heritage.farmer",
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
		Experiences:          []storage.DaggerheartExperience{{Name: "Tactics", Modifier: 2}},
		MajorThreshold:       8,
		SevereThreshold:      12,
	})

	resp, err := svc.ResetCharacterCreationWorkflow(contextWithParticipantID("manager-1"), &statev1.ResetCharacterCreationWorkflowRequest{
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

func TestApplyCharacterCreationWorkflow_Success(t *testing.T) {
	svc := newWorkflowCharacterService(t, storage.DaggerheartCharacterProfile{
		CampaignID:  "c1",
		CharacterID: "ch1",
		Level:       1,
		HpMax:       6,
		StressMax:   6,
		Evasion:     10,
	})

	resp, err := svc.ApplyCharacterCreationWorkflow(contextWithParticipantID("manager-1"), &statev1.ApplyCharacterCreationWorkflowRequest{
		CampaignId:  "c1",
		CharacterId: "ch1",
		SystemWorkflow: &statev1.ApplyCharacterCreationWorkflowRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCreationWorkflowInput{
				ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{ClassId: "class.guardian", SubclassId: "subclass.stalwart"},
				HeritageInput:      &daggerheartv1.DaggerheartCreationStepHeritageInput{AncestryId: "heritage.ancestry.clank", CommunityId: "heritage.community.farmer"},
				TraitsInput:        &daggerheartv1.DaggerheartCreationStepTraitsInput{Agility: 2, Strength: 1, Finesse: 1, Instinct: 0, Presence: 0, Knowledge: -1},
				DetailsInput:       &daggerheartv1.DaggerheartCreationStepDetailsInput{},
				EquipmentInput:     &daggerheartv1.DaggerheartCreationStepEquipmentInput{WeaponIds: []string{"weapon.longsword"}, ArmorId: "armor.gambeson-armor", PotionItemId: "item.minor-health-potion"},
				BackgroundInput:    &daggerheartv1.DaggerheartCreationStepBackgroundInput{Background: "City watch veteran"},
				ExperiencesInput: &daggerheartv1.DaggerheartCreationStepExperiencesInput{Experiences: []*daggerheartv1.DaggerheartExperience{
					{Name: "Shield wall", Modifier: 2},
				}},
				DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{DomainCardIds: []string{"domain-card.ward"}},
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
	if !resp.GetProgress().GetReady() {
		t.Fatal("ready = false, want true")
	}
	if got := resp.GetProgress().GetNextStep(); got != 0 {
		t.Fatalf("next_step = %d, want 0", got)
	}
}

func newWorkflowCharacterService(t *testing.T, profile storage.DaggerheartCharacterProfile) *CharacterService {
	t.Helper()

	participantStore := characterManagerParticipantStore("c1")
	characterStore := newFakeCharacterStore()
	characterStore.characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {
			ID:                 "ch1",
			CampaignID:         "c1",
			OwnerParticipantID: "manager-1",
			Name:               "Hero",
			Kind:               character.KindPC,
		},
	}

	dhStore := newFakeDaggerheartStore()
	if profile.CharacterID != "" {
		dhStore.profiles["c1"] = map[string]storage.DaggerheartCharacterProfile{
			"ch1": profile,
		}
	}

	domain := &fakeDomainEngine{result: engine.Result{Decision: command.Accept()}}
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}

	return NewCharacterService(Stores{
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		SystemStores: systemmanifest.ProjectionStores{Daggerheart: dhStore},
		DaggerheartContent: workflowContentStore{
			classes: map[string]storage.DaggerheartClass{
				"class.guardian": {
					ID:              "class.guardian",
					Name:            "Guardian",
					StartingEvasion: 9,
					StartingHP:      7,
					DomainIDs:       []string{"domain.valor", "domain.blade"},
				},
			},
			subclasses: map[string]storage.DaggerheartSubclass{
				"subclass.stalwart": {ID: "subclass.stalwart", ClassID: "class.guardian", Name: "Stalwart"},
			},
			heritages: map[string]storage.DaggerheartHeritage{
				"heritage.ancestry.clank":   {ID: "heritage.ancestry.clank", Kind: "ancestry", Name: "Clank"},
				"heritage.community.farmer": {ID: "heritage.community.farmer", Kind: "community", Name: "Farmer"},
			},
			weapons: map[string]storage.DaggerheartWeapon{
				"weapon.longsword": {ID: "weapon.longsword", Tier: 1, Category: "primary"},
			},
			armors: map[string]storage.DaggerheartArmor{
				"armor.gambeson-armor": {ID: "armor.gambeson-armor", Tier: 1, ArmorScore: 1, BaseMajorThreshold: 8, BaseSevereThreshold: 14},
			},
			items: map[string]storage.DaggerheartItem{
				"item.minor-health-potion":  {ID: "item.minor-health-potion"},
				"item.minor-stamina-potion": {ID: "item.minor-stamina-potion"},
			},
			domainCards: map[string]storage.DaggerheartDomainCard{
				"domain-card.ward":        {ID: "domain-card.ward", DomainID: "domain.valor", Name: "Ward"},
				"domain-card.arcana-bolt": {ID: "domain-card.arcana-bolt", DomainID: "domain.arcana", Name: "Arcana Bolt"},
			},
		},
		Event:  newFakeEventStore(),
		Domain: domain,
	})
}

type workflowContentStore struct {
	storage.DaggerheartContentReadStore
	classes     map[string]storage.DaggerheartClass
	subclasses  map[string]storage.DaggerheartSubclass
	heritages   map[string]storage.DaggerheartHeritage
	weapons     map[string]storage.DaggerheartWeapon
	armors      map[string]storage.DaggerheartArmor
	items       map[string]storage.DaggerheartItem
	domainCards map[string]storage.DaggerheartDomainCard
}

func (s workflowContentStore) GetDaggerheartClass(_ context.Context, id string) (storage.DaggerheartClass, error) {
	class, ok := s.classes[id]
	if !ok {
		return storage.DaggerheartClass{}, storage.ErrNotFound
	}
	return class, nil
}

func (s workflowContentStore) GetDaggerheartSubclass(_ context.Context, id string) (storage.DaggerheartSubclass, error) {
	subclass, ok := s.subclasses[id]
	if !ok {
		return storage.DaggerheartSubclass{}, storage.ErrNotFound
	}
	return subclass, nil
}

func (s workflowContentStore) GetDaggerheartHeritage(_ context.Context, id string) (storage.DaggerheartHeritage, error) {
	heritage, ok := s.heritages[id]
	if !ok {
		return storage.DaggerheartHeritage{}, storage.ErrNotFound
	}
	return heritage, nil
}

func (s workflowContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (storage.DaggerheartDomainCard, error) {
	card, ok := s.domainCards[id]
	if !ok {
		return storage.DaggerheartDomainCard{}, storage.ErrNotFound
	}
	return card, nil
}

func (s workflowContentStore) GetDaggerheartWeapon(_ context.Context, id string) (storage.DaggerheartWeapon, error) {
	weapon, ok := s.weapons[id]
	if !ok {
		return storage.DaggerheartWeapon{}, storage.ErrNotFound
	}
	return weapon, nil
}

func (s workflowContentStore) GetDaggerheartArmor(_ context.Context, id string) (storage.DaggerheartArmor, error) {
	armor, ok := s.armors[id]
	if !ok {
		return storage.DaggerheartArmor{}, storage.ErrNotFound
	}
	return armor, nil
}

func (s workflowContentStore) GetDaggerheartItem(_ context.Context, id string) (storage.DaggerheartItem, error) {
	item, ok := s.items[id]
	if !ok {
		return storage.DaggerheartItem{}, storage.ErrNotFound
	}
	return item, nil
}
