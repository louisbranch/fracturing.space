package overview

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/text/language"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

type overviewWorkspaceService struct {
	workspace campaignapp.CampaignWorkspace
}

func (s overviewWorkspaceService) CampaignName(context.Context, string) string {
	return s.workspace.Name
}
func (s overviewWorkspaceService) CampaignWorkspace(context.Context, string) (campaignapp.CampaignWorkspace, error) {
	return s.workspace, nil
}

type overviewSessionReads struct{}

func (overviewSessionReads) CampaignSessions(context.Context, string) ([]campaignapp.CampaignSession, error) {
	return nil, nil
}
func (overviewSessionReads) CampaignSessionReadiness(context.Context, string, language.Tag) (campaignapp.CampaignSessionReadiness, error) {
	return campaignapp.CampaignSessionReadiness{}, nil
}

type overviewAuth struct {
	manageCampaignErr error
}

func (a overviewAuth) RequireManageCampaign(context.Context, string) error {
	return a.manageCampaignErr
}
func (overviewAuth) RequireManageSession(context.Context, string) error      { return nil }
func (overviewAuth) RequireManageParticipants(context.Context, string) error { return nil }
func (overviewAuth) RequireManageInvites(context.Context, string) error      { return nil }
func (overviewAuth) RequireMutateCharacters(context.Context, string) error   { return nil }

type overviewAutomationReads struct {
	summary  campaignapp.CampaignAIBindingSummary
	settings campaignapp.CampaignAIBindingSettings
}

func (r overviewAutomationReads) CampaignAIBindingSummary(context.Context, string, string, string) (campaignapp.CampaignAIBindingSummary, error) {
	return r.summary, nil
}
func (r overviewAutomationReads) CampaignAIBindingSettings(context.Context, string, string) (campaignapp.CampaignAIBindingSettings, error) {
	return r.settings, nil
}

type overviewAutomationMutation struct {
	lastCampaignID string
	lastInput      campaignapp.UpdateCampaignAIBindingInput
}

func (m *overviewAutomationMutation) UpdateCampaignAIBinding(_ context.Context, campaignID string, input campaignapp.UpdateCampaignAIBindingInput) error {
	m.lastCampaignID = campaignID
	m.lastInput = input
	return nil
}

type overviewConfiguration struct {
	lastCampaignID string
	lastInput      campaignapp.UpdateCampaignInput
}

func (c *overviewConfiguration) UpdateCampaign(_ context.Context, campaignID string, input campaignapp.UpdateCampaignInput) error {
	c.lastCampaignID = campaignID
	c.lastInput = input
	return nil
}

func newOverviewHandler(t *testing.T, authErr error) (Handler, *overviewAutomationMutation, *overviewConfiguration) {
	t.Helper()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		func(*http.Request) module.Viewer { return module.Viewer{} },
	)
	detailHandler := campaigndetail.NewHandler(
		campaigndetail.NewSupport(base, requestmeta.SchemePolicy{}, nil),
		campaigndetail.PageServices{
			Workspace: overviewWorkspaceService{workspace: campaignapp.CampaignWorkspace{
				ID:        "camp-1",
				Name:      "The Guildhouse",
				System:    "Daggerheart",
				GMMode:    "Human",
				Status:    "Active",
				Locale:    "English (US)",
				Intent:    "Standard",
				AIAgentID: "agent-1",
			}},
			SessionReads:  overviewSessionReads{},
			Authorization: overviewAuth{manageCampaignErr: authErr},
		},
	)
	automationMutation := &overviewAutomationMutation{}
	configuration := &overviewConfiguration{}
	return NewHandler(detailHandler, HandlerServices{
		automationReads: overviewAutomationReads{
			summary: campaignapp.CampaignAIBindingSummary{Status: "Not required", CanManage: true},
			settings: campaignapp.CampaignAIBindingSettings{
				CurrentID: "agent-1",
				Options: []campaignapp.CampaignAIAgentOption{
					{ID: "agent-1", Label: "Narrator", Enabled: true, Selected: true},
				},
			},
		},
		automationMutate: automationMutation,
		configuration:    configuration,
	}), automationMutation, configuration
}

func TestHandleOverviewMethodNotAllowedSetsAllowHeader(t *testing.T) {
	t.Parallel()

	h, _, _ := newOverviewHandler(t, nil)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaign("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleOverviewMethodNotAllowed(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodGet+", HEAD" {
		t.Fatalf("Allow = %q, want %q", got, http.MethodGet+", HEAD")
	}
}

func TestHandleOverviewRendersOwnedOverviewPage(t *testing.T) {
	t.Parallel()

	h, _, _ := newOverviewHandler(t, nil)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleOverview(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-overview-name="The Guildhouse"`,
		`data-campaign-overview-campaign-id="camp-1"`,
		`data-campaign-overview-system="Daggerheart"`,
		`data-campaign-overview-ai-binding-status="Not required"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing overview marker %q: %q", marker, body)
		}
	}
}

func TestHandleCampaignEditRendersOwnedEditPage(t *testing.T) {
	t.Parallel()

	h, _, _ := newOverviewHandler(t, nil)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignEdit("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleCampaignEdit(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`action="` + routepath.AppCampaignEdit("camp-1") + `"`,
		`value="The Guildhouse"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing campaign-edit marker %q: %q", marker, body)
		}
	}
}

func TestHandleCampaignAIBindingPageRendersOwnedPage(t *testing.T) {
	t.Parallel()

	h, _, _ := newOverviewHandler(t, nil)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignAIBinding("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleCampaignAIBindingPage(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-ai-binding-page="true"`,
		`Narrator`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing ai-binding marker %q: %q", marker, body)
		}
	}
}

func TestHandleCampaignUpdateRedirectsAndForwardsTrimmedInput(t *testing.T) {
	t.Parallel()

	h, _, configuration := newOverviewHandler(t, nil)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignEdit("camp-1"), strings.NewReader("name=  Updated  &theme_prompt=  Storm  &locale=  pt-BR  "))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleCampaignUpdate(rr, req, "camp-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaign("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaign("camp-1"))
	}
	if configuration.lastCampaignID != "camp-1" {
		t.Fatalf("campaign id = %q, want %q", configuration.lastCampaignID, "camp-1")
	}
	if configuration.lastInput.Name == nil || *configuration.lastInput.Name != "Updated" {
		t.Fatalf("name input = %#v", configuration.lastInput)
	}
}

func TestHandleCampaignAIBindingRedirectsAndForwardsInput(t *testing.T) {
	t.Parallel()

	h, automationMutation, _ := newOverviewHandler(t, nil)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignAIBinding("camp-1"), strings.NewReader("ai_agent_id=  agent-2  "))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleCampaignAIBinding(rr, req, "camp-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaign("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaign("camp-1"))
	}
	if automationMutation.lastInput.AIAgentID != "agent-2" {
		t.Fatalf("AIAgentID = %q, want %q", automationMutation.lastInput.AIAgentID, "agent-2")
	}
}
