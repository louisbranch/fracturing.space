package workflow

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
)

func TestServiceLoadPageAssemblesCreationView(t *testing.T) {
	t.Parallel()

	svc := NewPageService(&workflowAppStub{
		progress: Progress{NextStep: 2},
		catalog: Catalog{
			Classes: []Class{{ID: "warrior", Name: "Warrior"}},
		},
		profile: Profile{
			CharacterName: "Nox",
			ClassID:       "warrior",
		},
	}, Registry{GameSystemDaggerheart: testWorkflow{}})

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
		progress: Progress{NextStep: 3},
	}
	svc := NewMutationService(app, Registry{GameSystemDaggerheart: testWorkflow{
		parsed: &StepInput{
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

	svc := NewMutationService(&workflowAppStub{
		progress: Progress{Ready: true},
	}, Registry{GameSystemDaggerheart: testWorkflow{}})

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

	svc := NewPageService(&workflowAppStub{}, nil)
	_, err := svc.LoadPage(context.Background(), "c1", "char-1", language.AmericanEnglish, "unknown")
	if err == nil {
		t.Fatalf("LoadPage() error = nil, want invalid input")
	}
	if apperrors.LocalizationKey(err) != "error.web.message.character_creation_step_is_not_available" {
		t.Fatalf("LoadPage() localization key = %q", apperrors.LocalizationKey(err))
	}
}

type workflowAppStub struct {
	progress   Progress
	progErr    error
	catalog    Catalog
	catalogErr error
	profile    Profile
	profileErr error
	applyErr   error
	resetErr   error
	lastStep   *StepInput
}

func (w workflowAppStub) CampaignCharacterCreationProgress(context.Context, string, string) (Progress, error) {
	return w.progress, w.progErr
}

func (w workflowAppStub) CampaignCharacterCreationCatalog(context.Context, language.Tag) (Catalog, error) {
	return w.catalog, w.catalogErr
}

func (w workflowAppStub) CampaignCharacterCreationProfile(context.Context, string, string) (Profile, error) {
	return w.profile, w.profileErr
}

func (w *workflowAppStub) ApplyCharacterCreationStep(_ context.Context, _ string, _ string, step *StepInput) error {
	w.lastStep = step
	return w.applyErr
}

func (w workflowAppStub) ResetCharacterCreationWorkflow(context.Context, string, string) error {
	return w.resetErr
}

type testWorkflow struct {
	parsed *StepInput
}

func (w testWorkflow) BuildView(
	progress Progress,
	catalog Catalog,
	profile Profile,
) CharacterCreationView {
	return CharacterCreationView{
		NextStep: progress.NextStep,
		ClassID:  profile.ClassID,
		Classes: []CreationClassView{{
			ID: catalog.Classes[0].ID,
		}},
	}
}

func (w testWorkflow) ParseStepInput(url.Values, int32) (*StepInput, error) {
	if w.parsed == nil {
		return &StepInput{
			Details: &campaignapp.CampaignCharacterCreationStepDetails{Description: "parsed"},
		}, nil
	}
	return w.parsed, nil
}
