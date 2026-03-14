package app

import (
	"context"
	"testing"

	"golang.org/x/text/language"
)

func TestCampaignCharacterCreationPageReadsLoadGenericInputs(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{
		characterCreationProgress: CampaignCharacterCreationProgress{
			Steps:    []CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: true}, {Step: 2, Key: "heritage", Complete: false}},
			NextStep: 2,
		},
		characterCreationProfile: CampaignCharacterCreationProfile{ClassID: "warrior", SubclassID: "guardian"},
		characterCreationCatalog: CampaignCharacterCreationCatalog{
			Classes: []CatalogClass{{ID: "warrior", Name: "Warrior"}},
		},
	})

	progress, err := svc.campaignCharacterCreationProgress(context.Background(), "c1", "char-1")
	if err != nil {
		t.Fatalf("campaignCharacterCreationProgress() error = %v", err)
	}
	if progress.NextStep != 2 {
		t.Fatalf("NextStep = %d, want 2", progress.NextStep)
	}
	profile, err := svc.campaignCharacterCreationProfile(context.Background(), "c1", "char-1")
	if err != nil {
		t.Fatalf("campaignCharacterCreationProfile() error = %v", err)
	}
	if profile.ClassID != "warrior" {
		t.Fatalf("ClassID = %q, want %q", profile.ClassID, "warrior")
	}
	catalog, err := svc.campaignCharacterCreationCatalog(context.Background(), language.AmericanEnglish)
	if err != nil {
		t.Fatalf("campaignCharacterCreationCatalog() error = %v", err)
	}
	if len(catalog.Classes) != 1 || catalog.Classes[0].ID != "warrior" {
		t.Fatalf("Classes = %#v, want single warrior class", catalog.Classes)
	}
}

func TestCampaignCharacterCreationCatalogForwardsLocale(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{}
	svc := newService(gateway)

	ptBR := language.MustParse("pt-BR")
	_, err := svc.campaignCharacterCreationCatalog(context.Background(), ptBR)
	if err != nil {
		t.Fatalf("campaignCharacterCreationCatalog() error = %v", err)
	}
	if gateway.characterCreationCatalogLocale != ptBR {
		t.Fatalf("catalog locale = %v, want %v", gateway.characterCreationCatalogLocale, ptBR)
	}
}

func TestCharacterCreationMutationMethodsDelegateToGateway(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{authorizationDecision: AuthorizationDecision{Evaluated: true, Allowed: true}}
	svc := newService(gateway)
	ctx := contextWithResolvedUserID("user-1")

	if err := svc.applyCharacterCreationStep(ctx, "c1", "char-1", &CampaignCharacterCreationStepInput{
		Details: &CampaignCharacterCreationStepDetails{},
	}); err != nil {
		t.Fatalf("applyCharacterCreationStep() error = %v", err)
	}
	if err := svc.resetCharacterCreationWorkflow(ctx, "c1", "char-1"); err != nil {
		t.Fatalf("resetCharacterCreationWorkflow() error = %v", err)
	}
	if len(gateway.calls) != 2 {
		t.Fatalf("calls = %v, want two workflow mutation calls", gateway.calls)
	}
	if gateway.calls[0] != "apply-character-creation-step" || gateway.calls[1] != "reset-character-creation-workflow" {
		t.Fatalf("calls = %v", gateway.calls)
	}
}
