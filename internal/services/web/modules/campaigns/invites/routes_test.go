package invites

import (
	"bytes"
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

func TestInviteViewsInputsAndRoutes(t *testing.T) {
	t.Parallel()

	page := &campaigndetail.PageContext{
		Workspace:        campaignapp.CampaignWorkspace{ID: "camp-1", Name: "Starfall"},
		CanManageInvites: true,
		Loc: testLocalizer{
			"game.campaign_invites.title":         "Invites",
			"game.campaign_invites.submit_create": "Create Invite",
		},
	}
	req := httptest.NewRequest(http.MethodGet, "https://example.com"+routepath.AppCampaignInvites("camp-1"), nil)
	invites := []campaignapp.CampaignInvite{{ID: "inv-1", ParticipantID: "part-1", ParticipantName: "Mira", Status: "pending"}}
	participants := []campaignapp.CampaignParticipant{
		{ID: "part-1", Name: "Mira", Controller: "human"},
		{ID: "part-2", Name: "Taro", Controller: "human"},
	}

	view := invitesView(page, "camp-1", invites, req)
	if len(view.Invites) != 1 || view.Invites[0].PublicURL == "" {
		t.Fatalf("invitesView() = %#v", view)
	}
	if got := invitesBreadcrumbs(page); len(got) != 1 || got[0].Label != "Invites" {
		t.Fatalf("invitesBreadcrumbs() = %#v", got)
	}
	createView := inviteCreateView(page, "camp-1", participants, invites)
	if len(createView.InviteSeatOptions) != 1 || createView.InviteSeatOptions[0].ParticipantID != "part-2" {
		t.Fatalf("inviteCreateView() = %#v", createView)
	}
	if got := inviteCreateBreadcrumbs(page, "camp-1"); len(got) != 2 || got[1].Label != "Create Invite" {
		t.Fatalf("inviteCreateBreadcrumbs() = %#v", got)
	}
	if got := mapInvitesView(invites, req); len(got) != 1 || got[0].PublicURL != "https://example.com"+routepath.PublicInvite("inv-1") {
		t.Fatalf("mapInvitesView() = %#v", got)
	}
	if got := absolutePublicInviteURL(req, "inv-1"); got != "https://example.com"+routepath.PublicInvite("inv-1") {
		t.Fatalf("absolutePublicInviteURL() = %q", got)
	}
	if got := absolutePublicInviteURL(nil, "inv-1"); got != routepath.PublicInvite("inv-1") {
		t.Fatalf("absolutePublicInviteURL(nil) = %q", got)
	}
	if !campaignInviteIsPending(" pending ") || campaignInviteSeatController("controller_ai") != "ai" {
		t.Fatalf("invite helper normalization failed")
	}

	create := parseCreateInviteInput(url.Values{"participant_id": {"  part-1  "}, "username": {"  mira  "}})
	if create.ParticipantID != "part-1" || create.RecipientUsername != "mira" {
		t.Fatalf("parseCreateInviteInput() = %#v", create)
	}
	revoke := parseRevokeInviteInput(url.Values{"invite_id": {"  inv-1  "}})
	if revoke.InviteID != "inv-1" {
		t.Fatalf("parseRevokeInviteInput() = %#v", revoke)
	}

	searchReq := httptest.NewRequest(http.MethodPost, routepath.AppCampaignInviteSearch("camp-1"), bytes.NewBufferString(`{"query":"  mira  ","limit":5}`))
	searchReq.Header.Set("Content-Type", "application/json")
	input, err := parseInviteSearchInput(searchReq)
	if err != nil || input.Query != "mira" || input.Limit != 5 {
		t.Fatalf("parseInviteSearchInput() = (%#v, %v)", input, err)
	}
	response := newInviteSearchResponse([]campaignapp.InviteUserSearchResult{{UserID: "user-1", Username: "mira", Name: "Mira", IsContact: true}})
	if len(response.Users) != 1 || !response.Users[0].IsContact {
		t.Fatalf("newInviteSearchResponse() = %#v", response)
	}

	mux := http.NewServeMux()
	RegisterStableRoutes(mux, Handler{})
	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: routepath.AppCampaignInvites("camp-1")},
		{method: http.MethodGet, path: routepath.AppCampaignInviteCreate("camp-1")},
		{method: http.MethodPost, path: routepath.AppCampaignInviteSearch("camp-1")},
		{method: http.MethodPost, path: routepath.AppCampaignInviteCreate("camp-1")},
		{method: http.MethodPost, path: routepath.AppCampaignInviteRevoke("camp-1")},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		if _, pattern := mux.Handler(req); pattern == "" {
			t.Fatalf("route %s %s was not registered", tc.method, tc.path)
		}
	}
}
