package invites

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

type inviteWorkspaceService struct {
	workspace campaignapp.CampaignWorkspace
}

func (s inviteWorkspaceService) CampaignName(context.Context, string) string { return s.workspace.Name }
func (s inviteWorkspaceService) CampaignWorkspace(context.Context, string) (campaignapp.CampaignWorkspace, error) {
	return s.workspace, nil
}

type inviteSessionReads struct{}

func (inviteSessionReads) CampaignSessions(context.Context, string) ([]campaignapp.CampaignSession, error) {
	return nil, nil
}
func (inviteSessionReads) CampaignSessionReadiness(context.Context, string, language.Tag) (campaignapp.CampaignSessionReadiness, error) {
	return campaignapp.CampaignSessionReadiness{}, nil
}

type inviteAuth struct{}

func (inviteAuth) RequireManageCampaign(context.Context, string) error     { return nil }
func (inviteAuth) RequireManageSession(context.Context, string) error      { return nil }
func (inviteAuth) RequireManageParticipants(context.Context, string) error { return nil }
func (inviteAuth) RequireManageInvites(context.Context, string) error      { return nil }
func (inviteAuth) RequireMutateCharacters(context.Context, string) error   { return nil }

type inviteReads struct {
	items   []campaignapp.CampaignInvite
	results []campaignapp.InviteUserSearchResult
}

func (r inviteReads) CampaignInvites(context.Context, string) ([]campaignapp.CampaignInvite, error) {
	return r.items, nil
}
func (r inviteReads) SearchInviteUsers(_ context.Context, _ string, input campaignapp.SearchInviteUsersInput) ([]campaignapp.InviteUserSearchResult, error) {
	return append([]campaignapp.InviteUserSearchResult(nil), r.results...), nil
}

type inviteParticipantReads struct {
	items []campaignapp.CampaignParticipant
}

func (r inviteParticipantReads) CampaignParticipants(context.Context, string) ([]campaignapp.CampaignParticipant, error) {
	return r.items, nil
}
func (inviteParticipantReads) CampaignParticipantCreator(context.Context, string) (campaignapp.CampaignParticipantCreator, error) {
	return campaignapp.CampaignParticipantCreator{}, nil
}
func (inviteParticipantReads) CampaignParticipantEditor(context.Context, string, string) (campaignapp.CampaignParticipantEditor, error) {
	return campaignapp.CampaignParticipantEditor{}, nil
}

type inviteMutation struct {
	lastCreate campaignapp.CreateInviteInput
	lastRevoke campaignapp.RevokeInviteInput
}

func (m *inviteMutation) CreateInvite(_ context.Context, _ string, input campaignapp.CreateInviteInput) error {
	m.lastCreate = input
	return nil
}
func (m *inviteMutation) RevokeInvite(_ context.Context, _ string, input campaignapp.RevokeInviteInput) error {
	m.lastRevoke = input
	return nil
}

type inviteSync struct {
	lastUserIDs  []string
	lastCampaign string
}

func (s *inviteSync) ProfileSaved(context.Context, string)            {}
func (s *inviteSync) CampaignCreated(context.Context, string, string) {}
func (s *inviteSync) SessionStarted(context.Context, string, string)  {}
func (s *inviteSync) SessionEnded(context.Context, string, string)    {}
func (s *inviteSync) InviteChanged(_ context.Context, userIDs []string, campaignID string) {
	s.lastUserIDs = append([]string(nil), userIDs...)
	s.lastCampaign = campaignID
}

func newInviteHandler(t *testing.T) (Handler, *inviteMutation, *inviteSync) {
	t.Helper()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		func(*http.Request) module.Viewer { return module.Viewer{} },
	)
	sync := &inviteSync{}
	detailHandler := campaigndetail.NewHandler(
		campaigndetail.NewSupport(base, requestmeta.SchemePolicy{}, sync),
		campaigndetail.PageServices{
			Workspace:     inviteWorkspaceService{workspace: campaignapp.CampaignWorkspace{ID: "camp-1", Name: "The Guildhouse"}},
			SessionReads:  inviteSessionReads{},
			Authorization: inviteAuth{},
		},
	)
	mutation := &inviteMutation{}
	return NewHandler(detailHandler, HandlerServices{
		reads: inviteReads{
			items:   []campaignapp.CampaignInvite{{ID: "inv-1", ParticipantID: "p-1", ParticipantName: "Owner", Status: "Pending"}},
			results: []campaignapp.InviteUserSearchResult{{UserID: "user-2", Username: "alice", Name: "Alice", IsContact: true}},
		},
		mutation: mutation,
		participantReads: inviteParticipantReads{items: []campaignapp.CampaignParticipant{
			{ID: "p-2", Name: "Aria", Controller: "human"},
		}},
	}), mutation, sync
}

func TestHandleInvitesRendersOwnedInvitesPage(t *testing.T) {
	t.Parallel()

	h, _, _ := newInviteHandler(t)
	req := httptest.NewRequest(http.MethodGet, "https://example.com"+routepath.AppCampaignInvites("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleInvites(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-invites-header="true"`,
		`data-campaign-invite-card-id="inv-1"`,
		`data-campaign-invite-create-option-id="p-2"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing invite marker %q: %q", marker, body)
		}
	}
}

func TestHandleInviteSearchReturnsJSONResults(t *testing.T) {
	t.Parallel()

	h, _, _ := newInviteHandler(t)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignInviteSearch("camp-1"), strings.NewReader(`{"query":"alice","limit":3}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.HandleInviteSearch(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `"username":"alice"`) {
		t.Fatalf("body = %q, want username result", rr.Body.String())
	}
}

func TestHandleInviteCreateRedirectsAndNotifiesSync(t *testing.T) {
	t.Parallel()

	h, mutation, sync := newInviteHandler(t)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignInviteCreate("camp-1"), strings.NewReader("participant_id=p-2&username=alice"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleInviteCreate(rr, req, "camp-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignInvites("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignInvites("camp-1"))
	}
	if mutation.lastCreate.ParticipantID != "p-2" || mutation.lastCreate.RecipientUsername != "alice" {
		t.Fatalf("create input = %#v", mutation.lastCreate)
	}
	if sync.lastCampaign != "camp-1" || len(sync.lastUserIDs) != 1 || sync.lastUserIDs[0] != "user-1" {
		t.Fatalf("sync = %#v", sync)
	}
}

func TestHandleInviteRevokeRedirectsAndNotifiesSync(t *testing.T) {
	t.Parallel()

	h, mutation, sync := newInviteHandler(t)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignInviteRevoke("camp-1"), strings.NewReader("invite_id=inv-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleInviteRevoke(rr, req, "camp-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignInvites("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignInvites("camp-1"))
	}
	if mutation.lastRevoke.InviteID != "inv-1" {
		t.Fatalf("revoke input = %#v", mutation.lastRevoke)
	}
	if sync.lastCampaign != "camp-1" {
		t.Fatalf("sync = %#v", sync)
	}
}
