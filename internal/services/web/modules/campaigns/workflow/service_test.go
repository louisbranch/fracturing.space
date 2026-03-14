package workflow

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

func TestServiceLoadPageAssemblesCreationView(t *testing.T) {
	t.Parallel()

	svc := NewService(&workflowAppStub{
		data: campaignapp.CampaignCharacterCreationData{
			Progress: campaignapp.CampaignCharacterCreationProgress{NextStep: 2},
			Catalog: campaignapp.CampaignCharacterCreationCatalog{
				Classes: []campaignapp.CatalogClass{{ID: "warrior", Name: "Warrior"}},
			},
			Profile: campaignapp.CampaignCharacterCreationProfile{
				CharacterName: "Nox",
				ClassID:       "warrior",
			},
		},
	}, Registry{campaignapp.GameSystemDaggerheart: testWorkflow{}})

	page, err := svc.LoadPage(context.Background(), "c1", "char-1", language.AmericanEnglish, "Daggerheart")
	if err != nil {
		t.Fatalf("LoadPage() error = %v", err)
	}
	if page.CharacterName != "Nox" {
		t.Fatalf("CharacterName = %q, want %q", page.CharacterName, "Nox")
	}
	if page.Creation.NextStep != 2 {
		t.Fatalf("NextStep = %d, want 2", page.Creation.NextStep)
	}
	if len(page.Creation.Classes) != 1 || page.Creation.Classes[0].ID != "warrior" {
		t.Fatalf("Classes = %#v, want single warrior class", page.Creation.Classes)
	}
}

func TestServiceApplyStepParsesWorkflowInputAndDelegatesMutation(t *testing.T) {
	t.Parallel()

	app := &workflowAppStub{
		progress: campaignapp.CampaignCharacterCreationProgress{NextStep: 3},
	}
	svc := NewService(app, Registry{campaignapp.GameSystemDaggerheart: testWorkflow{
		parsed: &campaignapp.CampaignCharacterCreationStepInput{
			Details: &campaignapp.CampaignCharacterCreationStepDetails{Description: "done"},
		},
	}})

	err := svc.ApplyStep(context.Background(), "c1", "char-1", "daggerheart", url.Values{"description": {"done"}})
	if err != nil {
		t.Fatalf("ApplyStep() error = %v", err)
	}
	if app.lastStep == nil || app.lastStep.Details == nil || app.lastStep.Details.Description != "done" {
		t.Fatalf("lastStep = %#v, want parsed details step", app.lastStep)
	}
}

func TestServiceApplyStepRejectsReadyWorkflow(t *testing.T) {
	t.Parallel()

	svc := NewService(&workflowAppStub{
		progress: campaignapp.CampaignCharacterCreationProgress{Ready: true},
	}, Registry{campaignapp.GameSystemDaggerheart: testWorkflow{}})

	err := svc.ApplyStep(context.Background(), "c1", "char-1", "daggerheart", url.Values{})
	if err == nil {
		t.Fatalf("ApplyStep() error = nil, want invalid input")
	}
	if apperrors.HTTPStatus(err) != http.StatusBadRequest {
		t.Fatalf("ApplyStep() status = %d, want %d", apperrors.HTTPStatus(err), http.StatusBadRequest)
	}
}

func TestServiceRejectsUnsupportedWorkflow(t *testing.T) {
	t.Parallel()

	svc := NewService(&workflowAppStub{}, nil)
	_, err := svc.LoadPage(context.Background(), "c1", "char-1", language.AmericanEnglish, "unknown")
	if err == nil {
		t.Fatalf("LoadPage() error = nil, want invalid input")
	}
	if apperrors.LocalizationKey(err) != "error.web.message.character_creation_step_is_not_available" {
		t.Fatalf("LoadPage() localization key = %q", apperrors.LocalizationKey(err))
	}
}

type workflowAppStub struct {
	data     campaignapp.CampaignCharacterCreationData
	dataErr  error
	progress campaignapp.CampaignCharacterCreationProgress
	progErr  error
	applyErr error
	resetErr error
	lastStep *campaignapp.CampaignCharacterCreationStepInput
}

func (w workflowAppStub) CampaignCharacterCreationData(context.Context, string, string, language.Tag) (campaignapp.CampaignCharacterCreationData, error) {
	return w.data, w.dataErr
}

func (w workflowAppStub) CampaignCharacterCreationProgress(context.Context, string, string) (campaignapp.CampaignCharacterCreationProgress, error) {
	return w.progress, w.progErr
}

func (w *workflowAppStub) ApplyCharacterCreationStep(_ context.Context, _ string, _ string, step *campaignapp.CampaignCharacterCreationStepInput) error {
	w.lastStep = step
	return w.applyErr
}

func (w workflowAppStub) ResetCharacterCreationWorkflow(context.Context, string, string) error {
	return w.resetErr
}

type testWorkflow struct {
	parsed *campaignapp.CampaignCharacterCreationStepInput
}

func (w testWorkflow) AssembleCatalog(
	progress campaignapp.CampaignCharacterCreationProgress,
	catalog campaignapp.CampaignCharacterCreationCatalog,
	profile campaignapp.CampaignCharacterCreationProfile,
) campaignapp.CampaignCharacterCreation {
	return campaignapp.CampaignCharacterCreation{
		Progress: progress,
		Profile:  profile,
		Classes:  append([]campaignapp.CatalogClass(nil), catalog.Classes...),
	}
}

func (w testWorkflow) CreationView(creation campaignapp.CampaignCharacterCreation) campaignrender.CampaignCharacterCreationView {
	return campaignrender.CampaignCharacterCreationView{
		NextStep: creation.Progress.NextStep,
		ClassID:  creation.Profile.ClassID,
		Classes: []campaignrender.CampaignCreationClassView{{
			ID: creation.Classes[0].ID,
		}},
	}
}

func (w testWorkflow) ParseStepInput(url.Values, int32) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	if w.parsed == nil {
		return &campaignapp.CampaignCharacterCreationStepInput{
			Details: &campaignapp.CampaignCharacterCreationStepDetails{Description: "parsed"},
		}, nil
	}
	return w.parsed, nil
}
