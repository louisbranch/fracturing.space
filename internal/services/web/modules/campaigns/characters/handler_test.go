package characters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/text/language"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

type characterWorkspaceService struct {
	workspace campaignapp.CampaignWorkspace
}

func (s characterWorkspaceService) CampaignName(context.Context, string) string {
	return s.workspace.Name
}
func (s characterWorkspaceService) CampaignWorkspace(context.Context, string) (campaignapp.CampaignWorkspace, error) {
	return s.workspace, nil
}

type characterSessionReads struct{}

func (characterSessionReads) CampaignSessions(context.Context, string) ([]campaignapp.CampaignSession, error) {
	return nil, nil
}
func (characterSessionReads) CampaignSessionReadiness(context.Context, string, language.Tag) (campaignapp.CampaignSessionReadiness, error) {
	return campaignapp.CampaignSessionReadiness{}, nil
}

type characterAuth struct{}

func (characterAuth) RequireManageCampaign(context.Context, string) error     { return nil }
func (characterAuth) RequireManageSession(context.Context, string) error      { return nil }
func (characterAuth) RequireManageParticipants(context.Context, string) error { return nil }
func (characterAuth) RequireManageInvites(context.Context, string) error      { return nil }
func (characterAuth) RequireMutateCharacters(context.Context, string) error   { return nil }

type characterReads struct {
	items     []campaignapp.CampaignCharacter
	character campaignapp.CampaignCharacter
	editor    campaignapp.CampaignCharacterEditor
}

func (r *characterReads) CampaignCharacters(_ context.Context, _ string, _ campaignapp.CharacterReadContext) ([]campaignapp.CampaignCharacter, error) {
	return append([]campaignapp.CampaignCharacter(nil), r.items...), nil
}
func (r *characterReads) CampaignCharacter(context.Context, string, string, campaignapp.CharacterReadContext) (campaignapp.CampaignCharacter, error) {
	return r.character, nil
}
func (r *characterReads) CampaignCharacterEditor(context.Context, string, string, campaignapp.CharacterReadContext) (campaignapp.CampaignCharacterEditor, error) {
	return r.editor, nil
}

type characterControl struct {
	control                campaignapp.CampaignCharacterControl
	lastSetCampaignID      string
	lastSetCharacterID     string
	lastSetParticipantID   string
	lastClaimCampaignID    string
	lastClaimCharacterID   string
	lastClaimUserID        string
	lastReleaseCampaignID  string
	lastReleaseCharacterID string
	lastReleaseUserID      string
}

func (c *characterControl) CampaignCharacterControl(context.Context, string, string, string, campaignapp.CharacterReadContext) (campaignapp.CampaignCharacterControl, error) {
	return c.control, nil
}
func (c *characterControl) SetCharacterController(_ context.Context, campaignID, characterID, participantID string) error {
	c.lastSetCampaignID = campaignID
	c.lastSetCharacterID = characterID
	c.lastSetParticipantID = participantID
	return nil
}
func (c *characterControl) ClaimCharacterControl(_ context.Context, campaignID, characterID, userID string) error {
	c.lastClaimCampaignID = campaignID
	c.lastClaimCharacterID = characterID
	c.lastClaimUserID = userID
	return nil
}
func (c *characterControl) ReleaseCharacterControl(_ context.Context, campaignID, characterID, userID string) error {
	c.lastReleaseCampaignID = campaignID
	c.lastReleaseCharacterID = characterID
	c.lastReleaseUserID = userID
	return nil
}

type characterMutation struct {
	createResult          campaignapp.CreateCharacterResult
	lastCreate            campaignapp.CreateCharacterInput
	lastUpdateCampaignID  string
	lastUpdateCharacterID string
	lastUpdate            campaignapp.UpdateCharacterInput
	lastDeleteCampaignID  string
	lastDeleteCharacterID string
}

func (m *characterMutation) CreateCharacter(_ context.Context, _ string, input campaignapp.CreateCharacterInput) (campaignapp.CreateCharacterResult, error) {
	m.lastCreate = input
	return m.createResult, nil
}
func (m *characterMutation) UpdateCharacter(_ context.Context, campaignID, characterID string, input campaignapp.UpdateCharacterInput) error {
	m.lastUpdateCampaignID = campaignID
	m.lastUpdateCharacterID = characterID
	m.lastUpdate = input
	return nil
}
func (m *characterMutation) DeleteCharacter(_ context.Context, campaignID, characterID string) error {
	m.lastDeleteCampaignID = campaignID
	m.lastDeleteCharacterID = characterID
	return nil
}

type characterCreationApp struct {
	progress      campaignworkflow.Progress
	catalog       campaignworkflow.Catalog
	profile       campaignworkflow.Profile
	lastStep      *campaignworkflow.StepInput
	lastResetCID  string
	lastResetChar string
}

func (a *characterCreationApp) CampaignCharacterCreationProgress(context.Context, string, string) (campaignworkflow.Progress, error) {
	return a.progress, nil
}
func (a *characterCreationApp) CampaignCharacterCreationCatalog(context.Context, language.Tag) (campaignworkflow.Catalog, error) {
	return a.catalog, nil
}
func (a *characterCreationApp) CampaignCharacterCreationProfile(context.Context, string, string) (campaignworkflow.Profile, error) {
	return a.profile, nil
}
func (a *characterCreationApp) ApplyCharacterCreationStep(_ context.Context, _, _ string, step *campaignworkflow.StepInput) error {
	a.lastStep = step
	return nil
}
func (a *characterCreationApp) ResetCharacterCreationWorkflow(_ context.Context, campaignID, characterID string) error {
	a.lastResetCID = campaignID
	a.lastResetChar = characterID
	return nil
}

type characterWorkflow struct{}

func (characterWorkflow) BuildView(progress campaignworkflow.Progress, catalog campaignworkflow.Catalog, profile campaignworkflow.Profile) campaignworkflow.CharacterCreationView {
	view := campaignworkflow.CharacterCreationView{
		NextStep: progress.NextStep,
		ClassID:  profile.ClassID,
	}
	if len(catalog.Classes) > 0 {
		view.Classes = []campaignworkflow.CreationClassView{{ID: catalog.Classes[0].ID, Name: catalog.Classes[0].Name}}
	}
	return view
}

func (characterWorkflow) ParseStepInput(form url.Values, _ int32) (*campaignworkflow.StepInput, error) {
	return &campaignworkflow.StepInput{
		Details: &campaignapp.CampaignCharacterCreationStepDetails{
			Description: strings.TrimSpace(form.Get("description")),
		},
	}, nil
}

func newCharacterHandler(t *testing.T, system string) (Handler, *characterMutation, *characterControl, *characterCreationApp) {
	t.Helper()

	if system == "" {
		system = "Daggerheart"
	}

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		func(*http.Request) module.Viewer { return module.Viewer{} },
	)
	detailHandler := campaigndetail.NewHandler(
		campaigndetail.NewSupport(base, requestmeta.SchemePolicy{}, nil),
		campaigndetail.PageServices{
			Workspace: characterWorkspaceService{workspace: campaignapp.CampaignWorkspace{
				ID:     "camp-1",
				Name:   "The Guildhouse",
				System: system,
			}},
			SessionReads:  characterSessionReads{},
			Authorization: characterAuth{},
		},
	)

	character := campaignapp.CampaignCharacter{
		ID:            "char-1",
		Name:          "Aria",
		Kind:          "pc",
		Controller:    "human",
		Pronouns:      "she/her",
		OwnedByViewer: true,
		CanEdit:       true,
		Daggerheart: &campaignapp.CampaignCharacterDaggerheartSummary{
			Level:         2,
			ClassName:     "Warrior",
			SubclassName:  "Guardian",
			HeritageName:  "Drakona",
			CommunityName: "Wanderborne",
		},
	}

	reads := &characterReads{
		items:     []campaignapp.CampaignCharacter{character},
		character: character,
		editor:    campaignapp.CampaignCharacterEditor{Character: character},
	}
	control := &characterControl{
		control: campaignapp.CampaignCharacterControl{
			CurrentParticipantName: "Ariadne",
			CanSelfClaim:           true,
			CanSelfRelease:         true,
			CanManageControl:       true,
			Options: []campaignapp.CampaignCharacterControlOption{
				{ParticipantID: "part-1", Label: "Ariadne", Selected: true},
			},
		},
	}
	mutation := &characterMutation{
		createResult: campaignapp.CreateCharacterResult{CharacterID: "char-2"},
	}
	creationApp := &characterCreationApp{
		progress: campaignworkflow.Progress{
			Steps:    []campaignworkflow.Step{{Step: 1, Key: "details"}},
			NextStep: 1,
		},
		catalog: campaignworkflow.Catalog{
			Classes: []campaignworkflow.Class{{ID: "warrior", Name: "Warrior"}},
		},
		profile: campaignworkflow.Profile{
			CharacterName: "Aria",
			ClassID:       "warrior",
		},
	}
	registry := campaignworkflow.Install(campaignworkflow.Installation{
		ID:                "daggerheart",
		Aliases:           []string{"Daggerheart"},
		CharacterCreation: characterWorkflow{},
	})

	return NewHandler(detailHandler, HandlerServices{
		reads:            reads,
		control:          control,
		mutation:         mutation,
		creationPages:    campaignworkflow.NewPageService(creationApp, registry),
		creationMutation: campaignworkflow.NewMutationService(creationApp, registry),
	}), mutation, control, creationApp
}

func TestHandleCharactersRendersOwnedCharactersPage(t *testing.T) {
	t.Parallel()

	h, _, _, _ := newCharacterHandler(t, "Daggerheart")
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleCharacters(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-create-link="true"`,
		`data-campaign-character-card-id="char-1"`,
		`data-campaign-character-name="Aria"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing character marker %q: %q", marker, body)
		}
	}
}

func TestHandleCharacterCreateRedirectsIntoWorkflowWhenEnabled(t *testing.T) {
	t.Parallel()

	h, mutation, _, _ := newCharacterHandler(t, "Daggerheart")
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreate("camp-1"), strings.NewReader("name=Nyx&kind=pc&pronouns=they%2Fthem"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleCharacterCreate(rr, req, "camp-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacterCreation("camp-1", "char-2") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacterCreation("camp-1", "char-2"))
	}
	if mutation.lastCreate.Name != "Nyx" || mutation.lastCreate.Pronouns != "they/them" {
		t.Fatalf("create input = %#v", mutation.lastCreate)
	}
}

func TestHandleCharacterDetailRendersOwnedDetailAndWorkflowCard(t *testing.T) {
	t.Parallel()

	h, _, _, _ := newCharacterHandler(t, "Daggerheart")
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacter("camp-1", "char-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleCharacterDetail(rr, req, "camp-1", "char-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-character-detail-id="char-1"`,
		`data-campaign-character-control-card="true"`,
		`data-character-creation-workflow="true"`,
		`data-character-creation-link="true"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing detail marker %q: %q", marker, body)
		}
	}
}

func TestHandleCharacterControlClaimRedirectsAndUsesViewer(t *testing.T) {
	t.Parallel()

	h, _, control, _ := newCharacterHandler(t, "Daggerheart")
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterControlClaim("camp-1", "char-1"), strings.NewReader("claim=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleCharacterControlClaim(rr, req, "camp-1", "char-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacter("camp-1", "char-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacter("camp-1", "char-1"))
	}
	if control.lastClaimCampaignID != "camp-1" || control.lastClaimCharacterID != "char-1" || control.lastClaimUserID != "user-1" {
		t.Fatalf("claim = %#v", control)
	}
}

func TestHandleCharacterDeleteRedirectsAndForwardsIdentity(t *testing.T) {
	t.Parallel()

	h, mutation, _, _ := newCharacterHandler(t, "Daggerheart")
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterDelete("camp-1", "char-1"), strings.NewReader("delete=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleCharacterDelete(rr, req, "camp-1", "char-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacters("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacters("camp-1"))
	}
	if mutation.lastDeleteCampaignID != "camp-1" || mutation.lastDeleteCharacterID != "char-1" {
		t.Fatalf("delete = %#v", mutation)
	}
}

func TestHandleCharacterCreationPageRendersOwnedWorkflowPage(t *testing.T) {
	t.Parallel()

	h, _, _, _ := newCharacterHandler(t, "Daggerheart")
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacterCreation("camp-1", "char-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleCharacterCreationPage(rr, req, "camp-1", "char-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-character-creation-page="true"`,
		`data-character-creation-form-step="1"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing creation marker %q: %q", marker, body)
		}
	}
}

func TestHandleCharacterCreationStepRedirectsAndDelegatesParsedStep(t *testing.T) {
	t.Parallel()

	h, _, _, creationApp := newCharacterHandler(t, "Daggerheart")
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationStep("camp-1", "char-1"), strings.NewReader("description=Completed"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleCharacterCreationStep(rr, req, "camp-1", "char-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacterCreation("camp-1", "char-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacterCreation("camp-1", "char-1"))
	}
	if creationApp.lastStep == nil || creationApp.lastStep.Details == nil || creationApp.lastStep.Details.Description != "Completed" {
		t.Fatalf("lastStep = %#v", creationApp.lastStep)
	}
}

func TestHandleCharacterCreationResetRedirectsAndDelegatesReset(t *testing.T) {
	t.Parallel()

	h, _, _, creationApp := newCharacterHandler(t, "Daggerheart")
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignCharacterCreationReset("camp-1", "char-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleCharacterCreationReset(rr, req, "camp-1", "char-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignCharacterCreation("camp-1", "char-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignCharacterCreation("camp-1", "char-1"))
	}
	if creationApp.lastResetCID != "camp-1" || creationApp.lastResetChar != "char-1" {
		t.Fatalf("reset app = %#v", creationApp)
	}
}
