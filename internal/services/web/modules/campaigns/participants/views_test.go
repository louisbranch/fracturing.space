package participants

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

func TestParticipantViewsMapWorkspaceState(t *testing.T) {
	t.Parallel()

	page := &campaigndetail.PageContext{
		Workspace: campaignapp.CampaignWorkspace{ID: "camp-1", Name: "Starfall"},
		Loc: testLocalizer{
			"game.participants.title":       "Participants",
			"game.participants.action_add":  "Add participant",
			"game.participants.action_edit": "Edit participant",
		},
	}
	creator := campaignapp.CampaignParticipantCreator{
		Name:           "Mira",
		Role:           "gm",
		CampaignAccess: "owner",
		AllowGMRole:    true,
		AccessOptions:  []campaignapp.CampaignParticipantAccessOption{{Value: "owner", Allowed: true}},
	}
	editor := campaignapp.CampaignParticipantEditor{
		Participant: campaignapp.CampaignParticipant{
			ID:             "part-1",
			Name:           "Mira",
			Role:           "gm",
			Controller:     "human",
			Pronouns:       "she/her",
			CampaignAccess: "owner",
		},
		AllowGMRole:   true,
		AccessOptions: []campaignapp.CampaignParticipantAccessOption{{Value: "owner", Allowed: true}},
	}

	list := participantsView(page, "camp-1", []campaignapp.CampaignParticipant{
		{ID: "part-1", Name: "Mira", UserID: "user-1", CanEdit: true},
	}, "user-1", true)
	if !list.CanManageParticipants || len(list.Participants) != 1 || !list.Participants[0].IsViewer {
		t.Fatalf("participantsView() = %#v", list)
	}

	createView := participantCreateView(page, "camp-1", creator)
	if !createView.CanManageParticipants || createView.ParticipantCreator.Name != "Mira" {
		t.Fatalf("participantCreateView() = %#v", createView)
	}

	editView := participantEditView(page, "camp-1", "part-1", editor)
	if editView.ParticipantID != "part-1" || editView.ParticipantEditor.Name != "Mira" {
		t.Fatalf("participantEditView() = %#v", editView)
	}

	if got := participantsBreadcrumbs(page); len(got) != 1 || got[0].Label != "Participants" {
		t.Fatalf("participantsBreadcrumbs() = %#v", got)
	}
	if got := participantCreateBreadcrumbs(page, "camp-1"); len(got) != 2 || got[0].URL != routepath.AppCampaignParticipants("camp-1") {
		t.Fatalf("participantCreateBreadcrumbs() = %#v", got)
	}
	if got := participantEditBreadcrumbs(page, "camp-1"); len(got) != 2 || got[1].Label != "Edit participant" {
		t.Fatalf("participantEditBreadcrumbs() = %#v", got)
	}
}

func TestParseCreateParticipantInputAndRoutes(t *testing.T) {
	t.Parallel()

	input := parseCreateParticipantInput(url.Values{
		"name":            {"  Mira  "},
		"role":            {"  gm  "},
		"campaign_access": {"  owner  "},
	})
	if input.Name != "Mira" || input.Role != "gm" || input.CampaignAccess != "owner" {
		t.Fatalf("parseCreateParticipantInput() = %#v", input)
	}

	mux := http.NewServeMux()
	RegisterStableRoutes(mux, Handler{})
	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: routepath.AppCampaignParticipants("camp-1")},
		{method: http.MethodGet, path: routepath.AppCampaignParticipantCreate("camp-1")},
		{method: http.MethodPost, path: routepath.AppCampaignParticipantCreate("camp-1")},
		{method: http.MethodPost, path: routepath.AppCampaignParticipantEdit("camp-1", "part-1")},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if _, pattern := mux.Handler(req); pattern == "" {
			t.Fatalf("route %s %s was not registered", tc.method, tc.path)
		}
	}
}
