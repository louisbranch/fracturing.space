package charactertransport

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
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
		ArmorMax:             0,
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
	if !resp.GetProgress().GetReady() {
		t.Fatal("ready = false, want true")
	}
	if got := resp.GetProgress().GetNextStep(); got != 0 {
		t.Fatalf("next_step = %d, want 0", got)
	}
}

func newWorkflowCharacterService(t *testing.T, profile projectionstore.DaggerheartCharacterProfile) *Service {
	t.Helper()

	participantStore := characterManagerParticipantStore("c1")
	characterStore := gametest.NewFakeCharacterStore()
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch1": {
			ID:                 "ch1",
			CampaignID:         "c1",
			OwnerParticipantID: "manager-1",
			Name:               "Hero",
			Kind:               character.KindPC,
		},
	}

	dhStore := gametest.NewFakeDaggerheartStore()
	if profile.CharacterID != "" {
		dhStore.Profiles["c1"] = map[string]projectionstore.DaggerheartCharacterProfile{
			"ch1": profile,
		}
	}

	domain := &fakeDomainEngine{result: engine.Result{Decision: command.Accept()}}
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
		System: bridge.SystemIDDaggerheart,
	}

	return NewService(Deps{
		Auth:        authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:    campaignStore,
		Participant: participantStore,
		Character:   characterStore,
		Daggerheart: dhStore,
		DaggerheartContent: workflowContentStore{
			classes: map[string]contentstore.DaggerheartClass{
				"class.guardian": {
					ID:              "class.guardian",
					Name:            "Guardian",
					StartingEvasion: 9,
					StartingHP:      7,
					DomainIDs:       []string{"domain.valor", "domain.blade"},
				},
			},
			subclasses: map[string]contentstore.DaggerheartSubclass{
				"subclass.stalwart": {ID: "subclass.stalwart", ClassID: "class.guardian", Name: "Stalwart"},
			},
			heritages: map[string]contentstore.DaggerheartHeritage{
				"heritage.ancestry.clank": {
					ID:   "heritage.ancestry.clank",
					Kind: "ancestry",
					Name: "Clank",
					Features: []contentstore.DaggerheartFeature{
						{ID: "heritage.ancestry.clank.feature-1", Name: "Clank One"},
						{ID: "heritage.ancestry.clank.feature-2", Name: "Clank Two"},
					},
				},
				"heritage.community.farmer": {ID: "heritage.community.farmer", Kind: "community", Name: "Farmer"},
			},
			weapons: map[string]contentstore.DaggerheartWeapon{
				"weapon.longsword": {ID: "weapon.longsword", Tier: 1, Category: "primary", Burden: 2},
			},
			armors: map[string]contentstore.DaggerheartArmor{
				"armor.gambeson-armor": {ID: "armor.gambeson-armor", Tier: 1, ArmorScore: 1, BaseMajorThreshold: 8, BaseSevereThreshold: 14},
			},
			items: map[string]contentstore.DaggerheartItem{
				"item.minor-health-potion":  {ID: "item.minor-health-potion"},
				"item.minor-stamina-potion": {ID: "item.minor-stamina-potion"},
			},
			domainCards: map[string]contentstore.DaggerheartDomainCard{
				"domain-card.ward":         {ID: "domain-card.ward", DomainID: "domain.valor", Name: "Ward", Level: 1},
				"domain-card.arcana-bolt":  {ID: "domain-card.arcana-bolt", DomainID: "domain.arcana", Name: "Arcana Bolt", Level: 1},
				"domain-card.blade-strike": {ID: "domain-card.blade-strike", DomainID: "domain.blade", Name: "Blade Strike", Level: 1},
			},
		},
		Write: domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
	})
}

func testCreationHeritageInput() *daggerheartv1.DaggerheartCreationStepHeritageSelectionInput {
	return &daggerheartv1.DaggerheartCreationStepHeritageSelectionInput{
		FirstFeatureAncestryId:  "heritage.ancestry.clank",
		SecondFeatureAncestryId: "heritage.ancestry.clank",
		CommunityId:             "heritage.community.farmer",
	}
}

func testProjectionHeritage() projectionstore.DaggerheartHeritageSelection {
	return projectionstore.DaggerheartHeritageSelection{
		FirstFeatureAncestryID:  "heritage.ancestry.clank",
		FirstFeatureID:          "heritage.ancestry.clank.feature-1",
		SecondFeatureAncestryID: "heritage.ancestry.clank",
		SecondFeatureID:         "heritage.ancestry.clank.feature-2",
		CommunityID:             "heritage.community.farmer",
	}
}

type workflowContentStore struct {
	contentstore.DaggerheartContentReadStore
	classes     map[string]contentstore.DaggerheartClass
	subclasses  map[string]contentstore.DaggerheartSubclass
	heritages   map[string]contentstore.DaggerheartHeritage
	weapons     map[string]contentstore.DaggerheartWeapon
	armors      map[string]contentstore.DaggerheartArmor
	items       map[string]contentstore.DaggerheartItem
	domainCards map[string]contentstore.DaggerheartDomainCard
}

func (s workflowContentStore) GetDaggerheartClass(_ context.Context, id string) (contentstore.DaggerheartClass, error) {
	class, ok := s.classes[id]
	if !ok {
		return contentstore.DaggerheartClass{}, storage.ErrNotFound
	}
	return class, nil
}

func (s workflowContentStore) GetDaggerheartSubclass(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	subclass, ok := s.subclasses[id]
	if !ok {
		return contentstore.DaggerheartSubclass{}, storage.ErrNotFound
	}
	return subclass, nil
}

func (s workflowContentStore) GetDaggerheartHeritage(_ context.Context, id string) (contentstore.DaggerheartHeritage, error) {
	heritage, ok := s.heritages[id]
	if !ok {
		return contentstore.DaggerheartHeritage{}, storage.ErrNotFound
	}
	return heritage, nil
}

func (s workflowContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (contentstore.DaggerheartDomainCard, error) {
	card, ok := s.domainCards[id]
	if !ok {
		return contentstore.DaggerheartDomainCard{}, storage.ErrNotFound
	}
	return card, nil
}

func (s workflowContentStore) GetDaggerheartWeapon(_ context.Context, id string) (contentstore.DaggerheartWeapon, error) {
	weapon, ok := s.weapons[id]
	if !ok {
		return contentstore.DaggerheartWeapon{}, storage.ErrNotFound
	}
	return weapon, nil
}

func (s workflowContentStore) GetDaggerheartArmor(_ context.Context, id string) (contentstore.DaggerheartArmor, error) {
	armor, ok := s.armors[id]
	if !ok {
		return contentstore.DaggerheartArmor{}, storage.ErrNotFound
	}
	return armor, nil
}

func (s workflowContentStore) GetDaggerheartItem(_ context.Context, id string) (contentstore.DaggerheartItem, error) {
	item, ok := s.items[id]
	if !ok {
		return contentstore.DaggerheartItem{}, storage.ErrNotFound
	}
	return item, nil
}
