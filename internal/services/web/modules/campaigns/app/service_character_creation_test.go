package app

import (
	"context"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

func TestCampaignCharacterCreationDelegatesToWorkflow(t *testing.T) {
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

	creation, err := svc.campaignCharacterCreation(context.Background(), "c1", "char-1", language.AmericanEnglish, testCreationWorkflow{})
	if err != nil {
		t.Fatalf("campaignCharacterCreation() error = %v", err)
	}
	if creation.Progress.NextStep != 2 {
		t.Fatalf("NextStep = %d, want 2", creation.Progress.NextStep)
	}
	if creation.Profile.ClassID != "warrior" {
		t.Fatalf("ClassID = %q, want %q", creation.Profile.ClassID, "warrior")
	}
	if len(creation.Classes) != 1 || creation.Classes[0].ID != "warrior" {
		t.Fatalf("Classes = %#v, want single warrior class", creation.Classes)
	}
}

func TestCampaignCharacterCreationForwardsCatalogLocale(t *testing.T) {
	t.Parallel()

	gateway := &campaignGatewayStub{}
	svc := newService(gateway)

	ptBR := language.MustParse("pt-BR")
	_, err := svc.campaignCharacterCreation(context.Background(), "c1", "char-1", ptBR, testCreationWorkflow{})
	if err != nil {
		t.Fatalf("campaignCharacterCreation() error = %v", err)
	}
	if gateway.characterCreationCatalogLocale != ptBR {
		t.Fatalf("catalog locale = %v, want %v", gateway.characterCreationCatalogLocale, ptBR)
	}
}

func TestCampaignCharacterCreationRejectsNilWorkflow(t *testing.T) {
	t.Parallel()

	svc := newService(&campaignGatewayStub{})
	_, err := svc.campaignCharacterCreation(context.Background(), "c1", "char-1", language.AmericanEnglish, nil)
	if err == nil {
		t.Fatalf("campaignCharacterCreation() error = nil, want invalid input")
	}
	if apperrors.HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("campaignCharacterCreation() status = %d, want %d", apperrors.HTTPStatus(err), http.StatusBadRequest)
	}
	if apperrors.LocalizationKey(err) != "error.web.message.character_creation_step_is_not_available" {
		t.Fatalf("campaignCharacterCreation() localization key = %q", apperrors.LocalizationKey(err))
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
