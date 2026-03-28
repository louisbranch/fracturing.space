package overview

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/text/message"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

type testLocalizer map[string]string

func (l testLocalizer) Sprintf(key message.Reference, args ...any) string {
	ref := fmt.Sprint(key)
	if value, ok := l[ref]; ok {
		return value
	}
	return ref
}

func TestOverviewViewBuildersAndRoutes(t *testing.T) {
	t.Parallel()

	page := &campaigndetail.PageContext{
		Workspace: campaignapp.CampaignWorkspace{
			ID:        "camp-1",
			Name:      "Starfall",
			Locale:    "pt-BR",
			AIAgentID: "agent-1",
		},
		Loc: testLocalizer{
			"game.campaign.menu.overview":    "Overview",
			"game.campaign.action_edit":      "Edit campaign",
			"game.campaign.ai_binding.title": "AI binding",
		},
	}
	summary := campaignapp.CampaignAIBindingSummary{Status: "bound", CanManage: true}
	settings := campaignapp.CampaignAIBindingSettings{
		CurrentID: "agent-1",
		Options: []campaignapp.CampaignAIAgentOption{
			{ID: "agent-1", Label: "Narrator", Enabled: true, Selected: true},
		},
	}

	if view := overviewView(page, "camp-1", true, summary); !view.CanEditCampaign || !view.CanManageAIBinding || view.AIBindingStatus != "bound" {
		t.Fatalf("overviewView() = %#v", view)
	}
	if view := campaignEditView(page, "camp-1"); view.LocaleValue != "pt-BR" {
		t.Fatalf("campaignEditView() = %#v", view)
	}
	if view := campaignAIBindingView(page, "camp-1", settings); view.AIBindingSettings.CurrentID != "agent-1" || len(view.AIBindingSettings.Options) != 1 {
		t.Fatalf("campaignAIBindingView() = %#v", view)
	}
	if got := campaignEditBreadcrumbs(page, "camp-1"); len(got) != 2 || got[0].URL != routepath.AppCampaign("camp-1") {
		t.Fatalf("campaignEditBreadcrumbs() = %#v", got)
	}
	if got := campaignAIBindingBreadcrumbs(page, "camp-1"); len(got) != 2 || got[1].Label != "AI binding" {
		t.Fatalf("campaignAIBindingBreadcrumbs() = %#v", got)
	}
	if got := mapAIBindingSettingsView(settings); got.CurrentID != "agent-1" || len(got.Options) != 1 || !got.Options[0].Selected {
		t.Fatalf("mapAIBindingSettingsView() = %#v", got)
	}
	if got := parseUpdateCampaignAIBindingInput(url.Values{"ai_agent_id": {"  agent-2  "}}); got.AIAgentID != "agent-2" {
		t.Fatalf("parseUpdateCampaignAIBindingInput() = %#v", got)
	}

	mux := http.NewServeMux()
	RegisterStableRoutes(mux, Handler{})
	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: routepath.AppCampaign("camp-1")},
		{method: http.MethodPost, path: routepath.AppCampaign("camp-1")},
		{method: http.MethodGet, path: routepath.AppCampaignEdit("camp-1")},
		{method: http.MethodPost, path: routepath.AppCampaignEdit("camp-1")},
		{method: http.MethodGet, path: routepath.AppCampaignAIBinding("camp-1")},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if _, pattern := mux.Handler(req); pattern == "" {
			t.Fatalf("route %s %s was not registered", tc.method, tc.path)
		}
	}
}
