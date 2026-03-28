package gateway

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestBatchCanCampaignActionMapsResults(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{
		Authorization: GRPCGatewayAuthorizationDeps{
			Client: fakeAuthorizationClient{
				batchCanResponse: &statev1.BatchCanResponse{Results: []*statev1.BatchCanResult{
					{CheckId: "char-a", Allowed: true, ReasonCode: "AUTHZ_ALLOW_RESOURCE_OWNER"},
					{CheckId: "char-b", Allowed: false, ReasonCode: "AUTHZ_DENY_NOT_RESOURCE_OWNER"},
				}},
			},
		},
	}

	decisions, err := gateway.BatchCanCampaignAction(context.Background(), "c1", []campaignapp.AuthorizationCheck{
		{
			CheckID:  "char-a",
			Action:   campaignapp.AuthorizationActionMutate,
			Resource: campaignapp.AuthorizationResourceCharacter,
			Target:   &campaignapp.AuthorizationTarget{ResourceID: "char-a"},
		},
		{
			CheckID:  "char-b",
			Action:   campaignapp.AuthorizationActionMutate,
			Resource: campaignapp.AuthorizationResourceCharacter,
			Target:   &campaignapp.AuthorizationTarget{ResourceID: "char-b"},
		},
	})
	if err != nil {
		t.Fatalf("BatchCanCampaignAction() error = %v", err)
	}
	if len(decisions) != 2 {
		t.Fatalf("len(decisions) = %d, want 2", len(decisions))
	}
	if decisions[0].CheckID != "char-a" || !decisions[0].Allowed || decisions[0].ReasonCode != "AUTHZ_ALLOW_RESOURCE_OWNER" {
		t.Fatalf("decisions[0] = %#v", decisions[0])
	}
	if decisions[1].CheckID != "char-b" || decisions[1].Allowed || decisions[1].ReasonCode != "AUTHZ_DENY_NOT_RESOURCE_OWNER" {
		t.Fatalf("decisions[1] = %#v", decisions[1])
	}
}

func TestBatchCanCampaignActionFallsBackToRequestCheckID(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{
		Authorization: GRPCGatewayAuthorizationDeps{
			Client: fakeAuthorizationClient{
				batchCanResponse: &statev1.BatchCanResponse{Results: []*statev1.BatchCanResult{{Allowed: true, ReasonCode: "AUTHZ_ALLOW_RESOURCE_OWNER"}}},
			},
		},
	}

	decisions, err := gateway.BatchCanCampaignAction(context.Background(), "c1", []campaignapp.AuthorizationCheck{
		{
			CheckID:  "char-a",
			Action:   campaignapp.AuthorizationActionMutate,
			Resource: campaignapp.AuthorizationResourceCharacter,
			Target:   &campaignapp.AuthorizationTarget{ResourceID: "char-a"},
		},
	})
	if err != nil {
		t.Fatalf("BatchCanCampaignAction() error = %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("len(decisions) = %d, want 1", len(decisions))
	}
	if decisions[0].CheckID != "char-a" {
		t.Fatalf("decisions[0].CheckID = %q, want %q", decisions[0].CheckID, "char-a")
	}
}

func TestBatchCanCampaignActionFailsWithClientError(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{Authorization: GRPCGatewayAuthorizationDeps{Client: fakeAuthorizationClient{batchCanErr: errors.New("auth unavailable")}}}
	_, err := gateway.BatchCanCampaignAction(context.Background(), "c1", []campaignapp.AuthorizationCheck{{CheckID: "char-a"}})
	if err == nil {
		t.Fatal("expected BatchCanCampaignAction() error")
	}
}

func TestCharacterCreationProgressMapsResponse(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{
		progressResp: &statev1.GetCharacterCreationProgressResponse{Progress: &statev1.CharacterCreationProgress{
			CampaignId:   "c1",
			CharacterId:  "char-1",
			Steps:        []*statev1.CharacterCreationStepProgress{{Step: 1, Key: "class_subclass", Complete: true}},
			NextStep:     2,
			Ready:        false,
			UnmetReasons: []string{"ancestry and community selection are required"},
		}},
	}
	gateway := GRPCGateway{CreationRead: GRPCGatewayCreationReadDeps{Character: characterClient}}

	progress, err := gateway.CharacterCreationProgress(context.Background(), "c1", "char-1")
	if err != nil {
		t.Fatalf("CharacterCreationProgress() error = %v", err)
	}
	if progress.NextStep != 2 {
		t.Fatalf("NextStep = %d, want 2", progress.NextStep)
	}
	if len(progress.Steps) != 1 || progress.Steps[0].Key != "class_subclass" || !progress.Steps[0].Complete {
		t.Fatalf("Steps = %#v", progress.Steps)
	}
	if len(progress.UnmetReasons) != 1 {
		t.Fatalf("UnmetReasons len = %d, want 1", len(progress.UnmetReasons))
	}
}

func TestCharacterCreationCatalogMapsContentCatalog(t *testing.T) {
	t.Parallel()

	contentClient := &fakeDaggerheartContentClient{
		resp: &daggerheartv1.GetDaggerheartContentCatalogResponse{
			Catalog: &daggerheartv1.DaggerheartContentCatalog{
				Classes:    []*daggerheartv1.DaggerheartClass{{Id: "warrior", Name: "Warrior", DomainIds: []string{"valor", "blade"}}},
				Subclasses: []*daggerheartv1.DaggerheartSubclass{{Id: "guardian", Name: "Guardian", ClassId: "warrior"}},
				Heritages:  []*daggerheartv1.DaggerheartHeritage{{Id: "human", Name: "Human", Kind: daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY}},
				Domains:    []*daggerheartv1.DaggerheartDomain{{Id: "valor", Name: "Valor"}},
				Weapons: []*daggerheartv1.DaggerheartWeapon{{
					Id:           "weapon.longsword",
					Name:         "Longsword",
					Category:     daggerheartv1.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_PRIMARY,
					Tier:         1,
					DisplayOrder: 7,
					DisplayGroup: daggerheartv1.DaggerheartWeaponDisplayGroup_DAGGERHEART_WEAPON_DISPLAY_GROUP_MAGIC,
				}},
				Armor:        []*daggerheartv1.DaggerheartArmor{{Id: "armor.chain", Name: "Chain", Tier: 1}},
				Items:        []*daggerheartv1.DaggerheartItem{{Id: "item.minor-health-potion", Name: "Minor Health Potion"}},
				DomainCards:  []*daggerheartv1.DaggerheartDomainCard{{Id: "card.guard", Name: "Guard", DomainId: "valor", Level: 1}},
				Adversaries:  []*daggerheartv1.DaggerheartAdversaryEntry{{Id: "adv.goblin", Name: "Goblin"}},
				Environments: []*daggerheartv1.DaggerheartEnvironment{{Id: "env.woods", Name: "Whispering Woods"}},
			},
		},
		assetMapResp: &daggerheartv1.GetDaggerheartAssetMapResponse{
			AssetMap: &daggerheartv1.DaggerheartAssetMap{
				Theme: "high_fantasy",
				Assets: []*daggerheartv1.DaggerheartAssetRef{
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "class",
						EntityId:   "warrior",
						SetId:      "daggerheart_class_set_v1",
						AssetId:    "class.warrior",
						CdnAssetId: "v1/high_fantasy/daggerheart_class_illustration/v1/class.warrior",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_CLASS_ICON,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_SET_DEFAULT,
						EntityType: "class",
						EntityId:   "warrior",
						SetId:      "daggerheart_class_icon_set_v1",
						AssetId:    "class.warrior",
						CdnAssetId: "v1/high_fantasy/daggerheart_class_icon/v1/class.warrior",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_SUBCLASS_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "subclass",
						EntityId:   "guardian",
						SetId:      "daggerheart_subclass_set_v1",
						AssetId:    "subclass.guardian",
						CdnAssetId: "v1/high_fantasy/daggerheart_subclass_illustration/v1/subclass.guardian",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ANCESTRY_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "ancestry",
						EntityId:   "human",
						SetId:      "daggerheart_ancestry_set_v1",
						AssetId:    "ancestry.human",
						CdnAssetId: "v1/high_fantasy/daggerheart_ancestry_illustration/v1/ancestry.human",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "domain",
						EntityId:   "valor",
						SetId:      "daggerheart_domain_set_v1",
						AssetId:    "domain.valor",
						CdnAssetId: "v1/high_fantasy/daggerheart_domain_illustration/v1/domain.valor",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_ICON,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "domain",
						EntityId:   "valor",
						SetId:      "daggerheart_domain_icon_set_v1",
						AssetId:    "domain.valor",
						CdnAssetId: "v1/high_fantasy/daggerheart_domain_icon/v1/domain.valor",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_DOMAIN_CARD_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "domain_card",
						EntityId:   "card.guard",
						SetId:      "daggerheart_domain_card_set_v1",
						AssetId:    "card.guard",
						CdnAssetId: "v1/high_fantasy/daggerheart_domain_card_illustration/v1/card.guard",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ADVERSARY_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "adversary",
						EntityId:   "adv.goblin",
						SetId:      "daggerheart_adversary_set_v1",
						AssetId:    "adv.goblin",
						CdnAssetId: "v1/high_fantasy/daggerheart_adversary_illustration/v1/adv.goblin",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ENVIRONMENT_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "environment",
						EntityId:   "env.woods",
						SetId:      "daggerheart_environment_set_v1",
						AssetId:    "env.woods",
						CdnAssetId: "v1/high_fantasy/daggerheart_environment_illustration/v1/env.woods",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_WEAPON_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "weapon",
						EntityId:   "weapon.longsword",
						SetId:      "daggerheart_weapon_set_v1",
						AssetId:    "weapon.longsword",
						CdnAssetId: "v1/high_fantasy/daggerheart_weapon_illustration/v1/weapon.longsword",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ARMOR_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "armor",
						EntityId:   "armor.chain",
						SetId:      "daggerheart_armor_set_v1",
						AssetId:    "armor.chain",
						CdnAssetId: "v1/high_fantasy/daggerheart_armor_illustration/v1/armor.chain",
					},
					{
						Type:       daggerheartv1.DaggerheartAssetType_DAGGERHEART_ASSET_TYPE_ITEM_ILLUSTRATION,
						Status:     daggerheartv1.DaggerheartAssetStatus_DAGGERHEART_ASSET_STATUS_MAPPED,
						EntityType: "item",
						EntityId:   "item.minor-health-potion",
						SetId:      "daggerheart_item_set_v1",
						AssetId:    "item.minor-health-potion",
						CdnAssetId: "v1/high_fantasy/daggerheart_item_illustration/v1/item.minor-health-potion",
					},
				},
			},
		},
	}
	gateway := GRPCGateway{
		CreationRead: GRPCGatewayCreationReadDeps{
			DaggerheartContent: contentClient,
			DaggerheartAsset:   contentClient,
		},
		AssetBaseURL: "https://res.cloudinary.com/fracturing-space/image/upload",
	}

	catalog, err := gateway.CharacterCreationCatalog(context.Background(), language.MustParse("pt-BR"))
	if err != nil {
		t.Fatalf("CharacterCreationCatalog() error = %v", err)
	}
	if contentClient.lastReq == nil {
		t.Fatalf("expected content catalog request")
	}
	if contentClient.lastReq.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("requested locale = %v, want %v", contentClient.lastReq.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
	if contentClient.lastAssetMapReq == nil {
		t.Fatalf("expected content asset map request")
	}
	if contentClient.lastAssetMapReq.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("asset-map locale = %v, want %v", contentClient.lastAssetMapReq.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
	if catalog.AssetTheme != "high_fantasy" {
		t.Fatalf("asset theme = %q, want %q", catalog.AssetTheme, "high_fantasy")
	}
	if len(catalog.Classes) != 1 || catalog.Classes[0].ID != "warrior" {
		t.Fatalf("Classes = %#v", catalog.Classes)
	}
	if catalog.Classes[0].Illustration.Status != "mapped" {
		t.Fatalf("class illustration status = %q, want mapped", catalog.Classes[0].Illustration.Status)
	}
	if catalog.Classes[0].Icon.Status != "set_default" {
		t.Fatalf("class icon status = %q, want set_default", catalog.Classes[0].Icon.Status)
	}
	if len(catalog.Subclasses) != 1 || catalog.Subclasses[0].ClassID != "warrior" {
		t.Fatalf("Subclasses = %#v", catalog.Subclasses)
	}
	if catalog.Subclasses[0].Illustration.Status != "mapped" {
		t.Fatalf("subclass illustration status = %q, want mapped", catalog.Subclasses[0].Illustration.Status)
	}
	if len(catalog.Heritages) != 1 || catalog.Heritages[0].Kind != "ancestry" {
		t.Fatalf("Heritages = %#v", catalog.Heritages)
	}
	if catalog.Heritages[0].Illustration.Status != "mapped" {
		t.Fatalf("heritage illustration status = %q, want mapped", catalog.Heritages[0].Illustration.Status)
	}
	if len(catalog.Domains) != 1 || catalog.Domains[0].Icon.Status != "mapped" {
		t.Fatalf("Domains = %#v", catalog.Domains)
	}
	if len(catalog.Weapons) != 1 || catalog.Weapons[0].Category != "primary" {
		t.Fatalf("Weapons = %#v", catalog.Weapons)
	}
	if catalog.Weapons[0].DisplayOrder != 7 || catalog.Weapons[0].DisplayGroup != "magic" {
		t.Fatalf("weapon display metadata = %#v, want order 7 and magic group", catalog.Weapons[0])
	}
	if catalog.Weapons[0].Illustration.Status != "mapped" {
		t.Fatalf("weapon illustration status = %q, want mapped", catalog.Weapons[0].Illustration.Status)
	}
	if len(catalog.Armor) != 1 || len(catalog.Items) != 1 || len(catalog.DomainCards) != 1 {
		t.Fatalf("catalog subsets = %#v", catalog)
	}
	if catalog.Armor[0].Illustration.Status != "mapped" {
		t.Fatalf("armor illustration status = %q, want mapped", catalog.Armor[0].Illustration.Status)
	}
	if catalog.Items[0].Illustration.Status != "mapped" {
		t.Fatalf("item illustration status = %q, want mapped", catalog.Items[0].Illustration.Status)
	}
	if catalog.DomainCards[0].Illustration.Status != "mapped" {
		t.Fatalf("domain card illustration status = %q, want mapped", catalog.DomainCards[0].Illustration.Status)
	}
	if len(catalog.Adversaries) != 1 || catalog.Adversaries[0].Illustration.Status != "mapped" {
		t.Fatalf("Adversaries = %#v", catalog.Adversaries)
	}
	if len(catalog.Environments) != 1 || catalog.Environments[0].Illustration.Status != "mapped" {
		t.Fatalf("Environments = %#v", catalog.Environments)
	}
}

func TestCharacterCreationCatalogDefaultsLocaleToEnglishUS(t *testing.T) {
	t.Parallel()

	contentClient := &fakeDaggerheartContentClient{}
	gateway := GRPCGateway{CreationRead: GRPCGatewayCreationReadDeps{DaggerheartContent: contentClient, DaggerheartAsset: contentClient}}

	if _, err := gateway.CharacterCreationCatalog(context.Background(), language.Und); err != nil {
		t.Fatalf("CharacterCreationCatalog() error = %v", err)
	}
	if contentClient.lastReq == nil {
		t.Fatalf("expected content catalog request")
	}
	if contentClient.lastReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("requested locale = %v, want %v", contentClient.lastReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
	if contentClient.lastAssetMapReq == nil {
		t.Fatalf("expected content asset map request")
	}
	if contentClient.lastAssetMapReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("asset-map locale = %v, want %v", contentClient.lastAssetMapReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
}

func TestCharacterCreationCatalog_ContinuesWhenAssetMapFails(t *testing.T) {
	t.Parallel()

	contentClient := &fakeDaggerheartContentClient{
		resp: &daggerheartv1.GetDaggerheartContentCatalogResponse{
			Catalog: &daggerheartv1.DaggerheartContentCatalog{
				Classes: []*daggerheartv1.DaggerheartClass{{Id: "warrior", Name: "Warrior"}},
			},
		},
		assetMapErr: errors.New("asset map unavailable"),
	}
	gateway := GRPCGateway{
		CreationRead: GRPCGatewayCreationReadDeps{
			DaggerheartContent: contentClient,
			DaggerheartAsset:   contentClient,
		},
		AssetBaseURL: "https://res.cloudinary.com/fracturing-space/image/upload",
	}

	catalog, err := gateway.CharacterCreationCatalog(context.Background(), language.AmericanEnglish)
	if err != nil {
		t.Fatalf("CharacterCreationCatalog() error = %v", err)
	}
	if len(catalog.Classes) != 1 {
		t.Fatalf("classes = %#v", catalog.Classes)
	}
	if catalog.Classes[0].Illustration.Status != "unavailable" {
		t.Fatalf("class illustration status = %q, want unavailable", catalog.Classes[0].Illustration.Status)
	}
}

func TestCharacterCreationProfileMapsDaggerheartFields(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{
		sheetResp: &statev1.GetCharacterSheetResponse{Profile: &statev1.CharacterProfile{SystemProfile: &statev1.CharacterProfile_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
			ClassId:    "warrior",
			SubclassId: "guardian",
			Heritage: &daggerheartv1.DaggerheartHeritageSelection{
				FirstFeatureAncestryId:  "human",
				FirstFeatureId:          "human.feature-1",
				SecondFeatureAncestryId: "human",
				SecondFeatureId:         "human.feature-2",
				CommunityId:             "loreborne",
			},
			Agility:              wrapperspb.Int32(1),
			Strength:             wrapperspb.Int32(2),
			Finesse:              wrapperspb.Int32(0),
			Instinct:             wrapperspb.Int32(-1),
			Presence:             wrapperspb.Int32(0),
			Knowledge:            wrapperspb.Int32(1),
			StartingWeaponIds:    []string{"weapon.longsword", "weapon.dagger"},
			StartingArmorId:      "armor.chain",
			StartingPotionItemId: "item.minor-health-potion",
			Background:           "Raised by sailors.",
			Experiences:          []*daggerheartv1.DaggerheartExperience{{Name: "Cartographer", Modifier: 2}},
			DomainCardIds:        []string{"card.guard", "card.cleave"},
			Connections:          "Bonded with the harbor watch.",
		}}}},
	}
	gateway := GRPCGateway{CreationRead: GRPCGatewayCreationReadDeps{Character: characterClient}}

	profile, err := gateway.CharacterCreationProfile(context.Background(), "c1", "char-1")
	if err != nil {
		t.Fatalf("CharacterCreationProfile() error = %v", err)
	}
	if profile.ClassID != "warrior" || profile.SubclassID != "guardian" ||
		profile.Heritage.FirstFeatureAncestryID != "human" || profile.Heritage.CommunityID != "loreborne" {
		t.Fatalf("profile = %#v", profile)
	}
	if profile.Agility != "1" || profile.Strength != "2" || profile.Finesse != "0" || profile.Instinct != "-1" || profile.Presence != "0" || profile.Knowledge != "1" {
		t.Fatalf("trait values = %#v", profile)
	}
	if profile.PrimaryWeaponID != "weapon.longsword" || profile.SecondaryWeaponID != "weapon.dagger" {
		t.Fatalf("weapon values = %#v", profile)
	}
	if profile.ArmorID != "armor.chain" || profile.PotionItemID != "item.minor-health-potion" {
		t.Fatalf("equipment values = %#v", profile)
	}
	if profile.Background != "Raised by sailors." || profile.Connections != "Bonded with the harbor watch." {
		t.Fatalf("text values = %#v", profile)
	}
	if len(profile.Experiences) != 1 || profile.Experiences[0].Name != "Cartographer" || profile.Experiences[0].Modifier != "2" {
		t.Fatalf("experience values = %#v", profile.Experiences)
	}
	if len(profile.DomainCardIDs) != 2 || profile.DomainCardIDs[0] != "card.guard" || profile.DomainCardIDs[1] != "card.cleave" {
		t.Fatalf("domain cards = %#v", profile.DomainCardIDs)
	}
}

func TestApplyAndResetCharacterCreationWorkflowForwardRequests(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{}
	gateway := GRPCGateway{CreationMutation: GRPCGatewayCreationMutationDeps{Character: characterClient}}

	step := &campaignapp.CampaignCharacterCreationStepInput{Details: &campaignapp.CampaignCharacterCreationStepDetails{}}
	if err := gateway.ApplyCharacterCreationStep(context.Background(), "c1", "char-1", step); err != nil {
		t.Fatalf("ApplyCharacterCreationStep() error = %v", err)
	}
	if characterClient.applyReq == nil {
		t.Fatalf("expected ApplyCharacterCreationStep request")
	}
	if characterClient.applyReq.GetCampaignId() != "c1" || characterClient.applyReq.GetCharacterId() != "char-1" {
		t.Fatalf("apply request = %#v", characterClient.applyReq)
	}
	if characterClient.applyReq.GetDaggerheart() == nil {
		t.Fatalf("expected daggerheart system step: %#v", characterClient.applyReq)
	}
	if _, ok := characterClient.applyReq.GetDaggerheart().GetStep().(*daggerheartv1.DaggerheartCreationStepInput_DetailsInput); !ok {
		t.Fatalf("system step type = %T", characterClient.applyReq.GetDaggerheart().GetStep())
	}

	if err := gateway.ResetCharacterCreationWorkflow(context.Background(), "c1", "char-1"); err != nil {
		t.Fatalf("ResetCharacterCreationWorkflow() error = %v", err)
	}
	if characterClient.resetReq == nil {
		t.Fatalf("expected ResetCharacterCreationWorkflow request")
	}
	if characterClient.resetReq.GetCampaignId() != "c1" || characterClient.resetReq.GetCharacterId() != "char-1" {
		t.Fatalf("reset request = %#v", characterClient.resetReq)
	}
}

func TestMapCampaignCharacterCreationStepToProtoRejectsNilOrAmbiguousInputs(t *testing.T) {
	t.Parallel()

	if _, err := mapCampaignCharacterCreationStepToProto(nil); err == nil {
		t.Fatalf("expected error for nil step")
	}

	if _, err := mapCampaignCharacterCreationStepToProto(&campaignapp.CampaignCharacterCreationStepInput{
		ClassSubclass: &campaignapp.CampaignCharacterCreationStepClassSubclass{
			ClassID:    "warrior",
			SubclassID: "guardian",
		},
		Details: &campaignapp.CampaignCharacterCreationStepDetails{},
	}); err == nil {
		t.Fatalf("expected error for ambiguous step")
	}
}

func TestMapCampaignCharacterCreationStepToProtoTrimsWhitespace(t *testing.T) {
	t.Parallel()

	step := &campaignapp.CampaignCharacterCreationStepInput{
		ClassSubclass: &campaignapp.CampaignCharacterCreationStepClassSubclass{
			ClassID:    "  warrior  ",
			SubclassID: "  guardian  ",
		},
	}
	protoStep, err := mapCampaignCharacterCreationStepToProto(step)
	if err != nil {
		t.Fatalf("mapCampaignCharacterCreationStepToProto() error = %v", err)
	}

	classStep, ok := protoStep.GetStep().(*daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput)
	if !ok {
		t.Fatalf("proto step type = %T", protoStep.GetStep())
	}
	if classStep.ClassSubclassInput == nil {
		t.Fatalf("class subclass input = nil")
	}
	if classStep.ClassSubclassInput.GetClassId() != "warrior" {
		t.Fatalf("class id = %q, want %q", classStep.ClassSubclassInput.GetClassId(), "warrior")
	}
	if classStep.ClassSubclassInput.GetSubclassId() != "guardian" {
		t.Fatalf("subclass id = %q, want %q", classStep.ClassSubclassInput.GetSubclassId(), "guardian")
	}
}

func TestMapCampaignCharacterCreationStepToProtoFiltersWhitespaceItems(t *testing.T) {
	t.Parallel()

	step := &campaignapp.CampaignCharacterCreationStepInput{
		Equipment: &campaignapp.CampaignCharacterCreationStepEquipment{
			WeaponIDs:    []string{"  weapon.longsword  ", "   ", "weapon.dagger"},
			ArmorID:      "  armor.chain  ",
			PotionItemID: "  item.minor-health-potion  ",
		},
	}
	protoStep, err := mapCampaignCharacterCreationStepToProto(step)
	if err != nil {
		t.Fatalf("mapCampaignCharacterCreationStepToProto() error = %v", err)
	}

	equipmentStep, ok := protoStep.GetStep().(*daggerheartv1.DaggerheartCreationStepInput_EquipmentInput)
	if !ok {
		t.Fatalf("proto step type = %T", protoStep.GetStep())
	}
	if equipmentStep.EquipmentInput == nil {
		t.Fatalf("equipment input = nil")
	}
	equipmentInput := equipmentStep.EquipmentInput
	if len(equipmentInput.GetWeaponIds()) != 2 {
		t.Fatalf("weapon ids = %#v, want 2", equipmentInput.GetWeaponIds())
	}
	if equipmentInput.GetWeaponIds()[0] != "weapon.longsword" {
		t.Fatalf("weapon id[0] = %q, want %q", equipmentInput.GetWeaponIds()[0], "weapon.longsword")
	}
	if equipmentInput.GetWeaponIds()[1] != "weapon.dagger" {
		t.Fatalf("weapon id[1] = %q, want %q", equipmentInput.GetWeaponIds()[1], "weapon.dagger")
	}
	if equipmentInput.GetArmorId() != "armor.chain" {
		t.Fatalf("armor id = %q, want %q", equipmentInput.GetArmorId(), "armor.chain")
	}
	if equipmentInput.GetPotionItemId() != "item.minor-health-potion" {
		t.Fatalf("potion item id = %q, want %q", equipmentInput.GetPotionItemId(), "item.minor-health-potion")
	}
}

func TestCreateCharacterForwardsRequestAndReturnsCharacterID(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{createResp: &statev1.CreateCharacterResponse{Character: &statev1.Character{Id: "char-42"}}}
	gateway := GRPCGateway{Mutation: GRPCGatewayMutationDeps{Character: characterClient}}

	created, err := gateway.CreateCharacter(context.Background(), "c1", campaignapp.CreateCharacterInput{Name: "Hero", Kind: campaignapp.CharacterKindPC})
	if err != nil {
		t.Fatalf("CreateCharacter() error = %v", err)
	}
	if characterClient.createReq == nil {
		t.Fatalf("expected CreateCharacter request")
	}
	if characterClient.createReq.GetCampaignId() != "c1" || characterClient.createReq.GetName() != "Hero" || characterClient.createReq.GetKind() != statev1.CharacterKind_PC {
		t.Fatalf("create request = %#v", characterClient.createReq)
	}
	if created.CharacterID != "char-42" {
		t.Fatalf("created.CharacterID = %q, want %q", created.CharacterID, "char-42")
	}
}

func TestCreateCharacterRejectsEmptyCreatedCharacterID(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{createResp: &statev1.CreateCharacterResponse{Character: &statev1.Character{}}}
	gateway := GRPCGateway{Mutation: GRPCGatewayMutationDeps{Character: characterClient}}

	_, err := gateway.CreateCharacter(context.Background(), "c1", campaignapp.CreateCharacterInput{Name: "Hero", Kind: campaignapp.CharacterKindPC})
	if err == nil {
		t.Fatalf("expected empty created character id error")
	}
}

type fakeAuthorizationClient struct {
	batchCanResponse *statev1.BatchCanResponse
	batchCanErr      error
}

func (f fakeAuthorizationClient) Can(context.Context, *statev1.CanRequest, ...grpc.CallOption) (*statev1.CanResponse, error) {
	return &statev1.CanResponse{}, nil
}

func (f fakeAuthorizationClient) BatchCan(context.Context, *statev1.BatchCanRequest, ...grpc.CallOption) (*statev1.BatchCanResponse, error) {
	if f.batchCanErr != nil {
		return nil, f.batchCanErr
	}
	if f.batchCanResponse != nil {
		return f.batchCanResponse, nil
	}
	return &statev1.BatchCanResponse{}, nil
}

type fakeDaggerheartContentClient struct {
	resp            *daggerheartv1.GetDaggerheartContentCatalogResponse
	err             error
	lastReq         *daggerheartv1.GetDaggerheartContentCatalogRequest
	assetMapResp    *daggerheartv1.GetDaggerheartAssetMapResponse
	assetMapErr     error
	lastAssetMapReq *daggerheartv1.GetDaggerheartAssetMapRequest
}

func (f *fakeDaggerheartContentClient) GetContentCatalog(_ context.Context, req *daggerheartv1.GetDaggerheartContentCatalogRequest, _ ...grpc.CallOption) (*daggerheartv1.GetDaggerheartContentCatalogResponse, error) {
	f.lastReq = req
	if f.err != nil {
		return nil, f.err
	}
	if f.resp != nil {
		return f.resp, nil
	}
	return &daggerheartv1.GetDaggerheartContentCatalogResponse{Catalog: &daggerheartv1.DaggerheartContentCatalog{}}, nil
}

func (f *fakeDaggerheartContentClient) GetAssetMap(_ context.Context, req *daggerheartv1.GetDaggerheartAssetMapRequest, _ ...grpc.CallOption) (*daggerheartv1.GetDaggerheartAssetMapResponse, error) {
	f.lastAssetMapReq = req
	if f.assetMapErr != nil {
		return nil, f.assetMapErr
	}
	if f.assetMapResp != nil {
		return f.assetMapResp, nil
	}
	return &daggerheartv1.GetDaggerheartAssetMapResponse{}, nil
}

type fakeCharacterWorkflowClient struct {
	listResp        *statev1.ListCharactersResponse
	listErr         error
	profilesResp    *statev1.ListCharacterProfilesResponse
	profilesErr     error
	lastProfilesReq *statev1.ListCharacterProfilesRequest
	createReq       *statev1.CreateCharacterRequest
	createResp      *statev1.CreateCharacterResponse
	createErr       error
	sheetResp       *statev1.GetCharacterSheetResponse
	sheetErr        error
	progressResp    *statev1.GetCharacterCreationProgressResponse
	progressErr     error
	applyReq        *statev1.ApplyCharacterCreationStepRequest
	applyErr        error
	resetReq        *statev1.ResetCharacterCreationWorkflowRequest
	resetErr        error
}

func (f *fakeCharacterWorkflowClient) ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &statev1.ListCharactersResponse{}, nil
}

func (f *fakeCharacterWorkflowClient) ListCharacterProfiles(_ context.Context, req *statev1.ListCharacterProfilesRequest, _ ...grpc.CallOption) (*statev1.ListCharacterProfilesResponse, error) {
	f.lastProfilesReq = req
	if f.profilesErr != nil {
		return nil, f.profilesErr
	}
	if f.profilesResp != nil {
		return f.profilesResp, nil
	}
	return &statev1.ListCharacterProfilesResponse{}, nil
}

func (f *fakeCharacterWorkflowClient) CreateCharacter(_ context.Context, req *statev1.CreateCharacterRequest, _ ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
	f.createReq = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &statev1.CreateCharacterResponse{Character: &statev1.Character{Id: "char-created"}}, nil
}

func (f *fakeCharacterWorkflowClient) UpdateCharacter(_ context.Context, _ *statev1.UpdateCharacterRequest, _ ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error) {
	return &statev1.UpdateCharacterResponse{}, nil
}

func (f *fakeCharacterWorkflowClient) DeleteCharacter(_ context.Context, _ *statev1.DeleteCharacterRequest, _ ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error) {
	return &statev1.DeleteCharacterResponse{}, nil
}

func (f *fakeCharacterWorkflowClient) GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error) {
	if f.sheetErr != nil {
		return nil, f.sheetErr
	}
	if f.sheetResp != nil {
		return f.sheetResp, nil
	}
	return &statev1.GetCharacterSheetResponse{}, nil
}

func (f *fakeCharacterWorkflowClient) GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error) {
	if f.progressErr != nil {
		return nil, f.progressErr
	}
	if f.progressResp != nil {
		return f.progressResp, nil
	}
	return &statev1.GetCharacterCreationProgressResponse{Progress: &statev1.CharacterCreationProgress{}}, nil
}

func (f *fakeCharacterWorkflowClient) ApplyCharacterCreationStep(_ context.Context, req *statev1.ApplyCharacterCreationStepRequest, _ ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error) {
	f.applyReq = req
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return &statev1.ApplyCharacterCreationStepResponse{}, nil
}

func (f *fakeCharacterWorkflowClient) ResetCharacterCreationWorkflow(_ context.Context, req *statev1.ResetCharacterCreationWorkflowRequest, _ ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error) {
	f.resetReq = req
	if f.resetErr != nil {
		return nil, f.resetErr
	}
	return &statev1.ResetCharacterCreationWorkflowResponse{}, nil
}

var _ moduleCharacterClientContract = (*fakeCharacterWorkflowClient)(nil)

type moduleCharacterClientContract interface {
	ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error)
	ListCharacterProfiles(context.Context, *statev1.ListCharacterProfilesRequest, ...grpc.CallOption) (*statev1.ListCharacterProfilesResponse, error)
	CreateCharacter(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error)
	UpdateCharacter(context.Context, *statev1.UpdateCharacterRequest, ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error)
	DeleteCharacter(context.Context, *statev1.DeleteCharacterRequest, ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error)
	GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error)
	GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error)
	ApplyCharacterCreationStep(context.Context, *statev1.ApplyCharacterCreationStepRequest, ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error)
	ResetCharacterCreationWorkflow(context.Context, *statev1.ResetCharacterCreationWorkflowRequest, ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error)
}

var _ moduleDaggerheartContentClientContract = (*fakeDaggerheartContentClient)(nil)
var _ moduleDaggerheartAssetClientContract = (*fakeDaggerheartContentClient)(nil)

type moduleDaggerheartContentClientContract interface {
	GetContentCatalog(context.Context, *daggerheartv1.GetDaggerheartContentCatalogRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartContentCatalogResponse, error)
}

type moduleDaggerheartAssetClientContract interface {
	GetAssetMap(context.Context, *daggerheartv1.GetDaggerheartAssetMapRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartAssetMapResponse, error)
}
