package campaigns

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestBatchCanCampaignActionMapsResults(t *testing.T) {
	t.Parallel()

	gateway := grpcGateway{
		authorizationClient: fakeAuthorizationClient{
			batchCanResponse: &statev1.BatchCanResponse{Results: []*statev1.BatchCanResult{
				{CheckId: "char-a", Allowed: true, ReasonCode: "AUTHZ_ALLOW_RESOURCE_OWNER"},
				{CheckId: "char-b", Allowed: false, ReasonCode: "AUTHZ_DENY_NOT_RESOURCE_OWNER"},
			}},
		},
	}

	decisions, err := gateway.BatchCanCampaignAction(context.Background(), "c1", []campaignAuthorizationCheck{
		{
			CheckID:  "char-a",
			Action:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
			Resource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
			Target:   &statev1.AuthorizationTarget{ResourceId: "char-a"},
		},
		{
			CheckID:  "char-b",
			Action:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
			Resource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
			Target:   &statev1.AuthorizationTarget{ResourceId: "char-b"},
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

	gateway := grpcGateway{
		authorizationClient: fakeAuthorizationClient{
			batchCanResponse: &statev1.BatchCanResponse{Results: []*statev1.BatchCanResult{{Allowed: true, ReasonCode: "AUTHZ_ALLOW_RESOURCE_OWNER"}}},
		},
	}

	decisions, err := gateway.BatchCanCampaignAction(context.Background(), "c1", []campaignAuthorizationCheck{
		{
			CheckID:  "char-a",
			Action:   statev1.AuthorizationAction_AUTHORIZATION_ACTION_MUTATE,
			Resource: statev1.AuthorizationResource_AUTHORIZATION_RESOURCE_CHARACTER,
			Target:   &statev1.AuthorizationTarget{ResourceId: "char-a"},
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

	gateway := grpcGateway{authorizationClient: fakeAuthorizationClient{batchCanErr: errors.New("auth unavailable")}}
	_, err := gateway.BatchCanCampaignAction(context.Background(), "c1", []campaignAuthorizationCheck{{CheckID: "char-a"}})
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
	gateway := grpcGateway{characterClient: characterClient}

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

	contentClient := &fakeDaggerheartContentClient{resp: &daggerheartv1.GetDaggerheartContentCatalogResponse{Catalog: &daggerheartv1.DaggerheartContentCatalog{
		Classes:     []*daggerheartv1.DaggerheartClass{{Id: "warrior", Name: "Warrior", DomainIds: []string{"valor", "blade"}}},
		Subclasses:  []*daggerheartv1.DaggerheartSubclass{{Id: "guardian", Name: "Guardian", ClassId: "warrior"}},
		Heritages:   []*daggerheartv1.DaggerheartHeritage{{Id: "human", Name: "Human", Kind: daggerheartv1.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY}},
		Weapons:     []*daggerheartv1.DaggerheartWeapon{{Id: "weapon.longsword", Name: "Longsword", Category: daggerheartv1.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_PRIMARY, Tier: 1}},
		Armor:       []*daggerheartv1.DaggerheartArmor{{Id: "armor.chain", Name: "Chain", Tier: 1}},
		Items:       []*daggerheartv1.DaggerheartItem{{Id: "item.minor-health-potion", Name: "Minor Health Potion"}},
		DomainCards: []*daggerheartv1.DaggerheartDomainCard{{Id: "card.guard", Name: "Guard", DomainId: "valor", Level: 1}},
	}}}
	gateway := grpcGateway{daggerheartClient: contentClient}

	catalog, err := gateway.CharacterCreationCatalog(context.Background(), commonv1.Locale_LOCALE_PT_BR)
	if err != nil {
		t.Fatalf("CharacterCreationCatalog() error = %v", err)
	}
	if contentClient.lastReq == nil {
		t.Fatalf("expected content catalog request")
	}
	if contentClient.lastReq.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("requested locale = %v, want %v", contentClient.lastReq.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
	if len(catalog.Classes) != 1 || catalog.Classes[0].ID != "warrior" {
		t.Fatalf("Classes = %#v", catalog.Classes)
	}
	if len(catalog.Subclasses) != 1 || catalog.Subclasses[0].ClassID != "warrior" {
		t.Fatalf("Subclasses = %#v", catalog.Subclasses)
	}
	if len(catalog.Heritages) != 1 || catalog.Heritages[0].Kind != "ancestry" {
		t.Fatalf("Heritages = %#v", catalog.Heritages)
	}
	if len(catalog.Weapons) != 1 || catalog.Weapons[0].Category != "primary" {
		t.Fatalf("Weapons = %#v", catalog.Weapons)
	}
	if len(catalog.Armor) != 1 || len(catalog.Items) != 1 || len(catalog.DomainCards) != 1 {
		t.Fatalf("catalog subsets = %#v", catalog)
	}
}

func TestCharacterCreationCatalogDefaultsLocaleToEnglishUS(t *testing.T) {
	t.Parallel()

	contentClient := &fakeDaggerheartContentClient{}
	gateway := grpcGateway{daggerheartClient: contentClient}

	if _, err := gateway.CharacterCreationCatalog(context.Background(), commonv1.Locale_LOCALE_UNSPECIFIED); err != nil {
		t.Fatalf("CharacterCreationCatalog() error = %v", err)
	}
	if contentClient.lastReq == nil {
		t.Fatalf("expected content catalog request")
	}
	if contentClient.lastReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("requested locale = %v, want %v", contentClient.lastReq.GetLocale(), commonv1.Locale_LOCALE_EN_US)
	}
}

func TestCharacterCreationProfileMapsDaggerheartFields(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{
		sheetResp: &statev1.GetCharacterSheetResponse{Profile: &statev1.CharacterProfile{SystemProfile: &statev1.CharacterProfile_Daggerheart{Daggerheart: &daggerheartv1.DaggerheartProfile{
			ClassId:              "warrior",
			SubclassId:           "guardian",
			AncestryId:           "human",
			CommunityId:          "loreborne",
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
	gateway := grpcGateway{characterClient: characterClient}

	profile, err := gateway.CharacterCreationProfile(context.Background(), "c1", "char-1")
	if err != nil {
		t.Fatalf("CharacterCreationProfile() error = %v", err)
	}
	if profile.ClassID != "warrior" || profile.SubclassID != "guardian" || profile.AncestryID != "human" || profile.CommunityID != "loreborne" {
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
	if profile.Background != "Raised by sailors." || profile.ExperienceName != "Cartographer" || profile.ExperienceModifier != "2" || profile.Connections != "Bonded with the harbor watch." {
		t.Fatalf("text values = %#v", profile)
	}
	if len(profile.DomainCardIDs) != 2 || profile.DomainCardIDs[0] != "card.guard" || profile.DomainCardIDs[1] != "card.cleave" {
		t.Fatalf("domain cards = %#v", profile.DomainCardIDs)
	}
}

func TestApplyAndResetCharacterCreationWorkflowForwardRequests(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{}
	gateway := grpcGateway{characterClient: characterClient}

	step := &daggerheartv1.DaggerheartCreationStepInput{Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{DetailsInput: &daggerheartv1.DaggerheartCreationStepDetailsInput{}}}
	if err := gateway.ApplyCharacterCreationStep(context.Background(), "c1", "char-1", step); err != nil {
		t.Fatalf("ApplyCharacterCreationStep() error = %v", err)
	}
	if characterClient.applyReq == nil {
		t.Fatalf("expected ApplyCharacterCreationStep request")
	}
	if characterClient.applyReq.GetCampaignId() != "c1" || characterClient.applyReq.GetCharacterId() != "char-1" {
		t.Fatalf("apply request = %#v", characterClient.applyReq)
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

func TestCreateCharacterForwardsRequestAndReturnsCharacterID(t *testing.T) {
	t.Parallel()

	characterClient := &fakeCharacterWorkflowClient{createResp: &statev1.CreateCharacterResponse{Character: &statev1.Character{Id: "char-42"}}}
	gateway := grpcGateway{characterClient: characterClient}

	created, err := gateway.CreateCharacter(context.Background(), "c1", CreateCharacterInput{Name: "Hero", Kind: statev1.CharacterKind_PC})
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
	gateway := grpcGateway{characterClient: characterClient}

	_, err := gateway.CreateCharacter(context.Background(), "c1", CreateCharacterInput{Name: "Hero", Kind: statev1.CharacterKind_PC})
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
	resp    *daggerheartv1.GetDaggerheartContentCatalogResponse
	err     error
	lastReq *daggerheartv1.GetDaggerheartContentCatalogRequest
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

type fakeCharacterWorkflowClient struct {
	listResp     *statev1.ListCharactersResponse
	listErr      error
	createReq    *statev1.CreateCharacterRequest
	createResp   *statev1.CreateCharacterResponse
	createErr    error
	sheetResp    *statev1.GetCharacterSheetResponse
	sheetErr     error
	progressResp *statev1.GetCharacterCreationProgressResponse
	progressErr  error
	applyReq     *statev1.ApplyCharacterCreationStepRequest
	applyErr     error
	resetReq     *statev1.ResetCharacterCreationWorkflowRequest
	resetErr     error
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
	CreateCharacter(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error)
	GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error)
	GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error)
	ApplyCharacterCreationStep(context.Context, *statev1.ApplyCharacterCreationStepRequest, ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error)
	ResetCharacterCreationWorkflow(context.Context, *statev1.ResetCharacterCreationWorkflowRequest, ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error)
}

var _ moduleDaggerheartContentClientContract = (*fakeDaggerheartContentClient)(nil)

type moduleDaggerheartContentClientContract interface {
	GetContentCatalog(context.Context, *daggerheartv1.GetDaggerheartContentCatalogRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartContentCatalogResponse, error)
}
