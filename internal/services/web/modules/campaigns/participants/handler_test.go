package participants

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

type participantWorkspaceService struct {
	workspace campaignapp.CampaignWorkspace
}

func (s participantWorkspaceService) CampaignName(context.Context, string) string {
	return s.workspace.Name
}
func (s participantWorkspaceService) CampaignWorkspace(context.Context, string) (campaignapp.CampaignWorkspace, error) {
	return s.workspace, nil
}

type participantSessionReads struct{}

func (participantSessionReads) CampaignSessions(context.Context, string) ([]campaignapp.CampaignSession, error) {
	return nil, nil
}
func (participantSessionReads) CampaignSessionReadiness(context.Context, string, language.Tag) (campaignapp.CampaignSessionReadiness, error) {
	return campaignapp.CampaignSessionReadiness{}, nil
}

type participantAuth struct{}

func (participantAuth) RequireManageCampaign(context.Context, string) error     { return nil }
func (participantAuth) RequireManageSession(context.Context, string) error      { return nil }
func (participantAuth) RequireManageParticipants(context.Context, string) error { return nil }
func (participantAuth) RequireManageInvites(context.Context, string) error      { return nil }
func (participantAuth) RequireMutateCharacters(context.Context, string) error   { return nil }

type participantReads struct {
	items   []campaignapp.CampaignParticipant
	creator campaignapp.CampaignParticipantCreator
	editor  campaignapp.CampaignParticipantEditor
}

func (r participantReads) CampaignParticipants(context.Context, string) ([]campaignapp.CampaignParticipant, error) {
	return r.items, nil
}
func (r participantReads) CampaignParticipantCreator(context.Context, string) (campaignapp.CampaignParticipantCreator, error) {
	return r.creator, nil
}
func (r participantReads) CampaignParticipantEditor(context.Context, string, string) (campaignapp.CampaignParticipantEditor, error) {
	return r.editor, nil
}

type participantMutation struct {
	lastCreate campaignapp.CreateParticipantInput
	lastUpdate campaignapp.UpdateParticipantInput
}

func (m *participantMutation) CreateParticipant(_ context.Context, _ string, input campaignapp.CreateParticipantInput) (campaignapp.CreateParticipantResult, error) {
	m.lastCreate = input
	return campaignapp.CreateParticipantResult{}, nil
}
func (m *participantMutation) UpdateParticipant(_ context.Context, _ string, input campaignapp.UpdateParticipantInput) error {
	m.lastUpdate = input
	return nil
}

func newParticipantHandler(t *testing.T) (Handler, *participantMutation) {
	t.Helper()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		func(*http.Request) module.Viewer { return module.Viewer{} },
	)
	detailHandler := campaigndetail.NewHandler(
		campaigndetail.NewSupport(base, requestmeta.SchemePolicy{}, nil),
		campaigndetail.PageServices{
			Workspace:     participantWorkspaceService{workspace: campaignapp.CampaignWorkspace{ID: "camp-1", Name: "The Guildhouse"}},
			SessionReads:  participantSessionReads{},
			Authorization: participantAuth{},
		},
	)
	mutation := &participantMutation{}
	return NewHandler(detailHandler, HandlerServices{
		reads: participantReads{
			items: []campaignapp.CampaignParticipant{{ID: "p-1", Name: "Owner", UserID: "user-1", CanEdit: true}},
			creator: campaignapp.CampaignParticipantCreator{
				Name: "Pending Seat",
			},
			editor: campaignapp.CampaignParticipantEditor{
				Participant: campaignapp.CampaignParticipant{ID: "p-1", Name: "Owner", Role: "gm"},
			},
		},
		mutation: mutation,
	}), mutation
}

func TestHandleParticipantsRendersOwnedParticipantsPage(t *testing.T) {
	t.Parallel()

	h, _ := newParticipantHandler(t)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipants("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleParticipants(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-participants-header="true"`,
		`data-campaign-participant-card-id="p-1"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing participant marker %q: %q", marker, body)
		}
	}
}

func TestHandleParticipantCreatePageRendersOwnedCreatePage(t *testing.T) {
	t.Parallel()

	h, _ := newParticipantHandler(t)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipantCreate("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleParticipantCreatePage(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-participant-create-page="true"`,
		`value="Pending Seat"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing create-page marker %q: %q", marker, body)
		}
	}
}

func TestHandleParticipantCreateRedirectsAndForwardsInput(t *testing.T) {
	t.Parallel()

	h, mutation := newParticipantHandler(t)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignParticipantCreate("camp-1"), strings.NewReader("name=Pending+Seat&role=player&campaign_access=member"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleParticipantCreate(rr, req, "camp-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignInvites("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignInvites("camp-1"))
	}
	if mutation.lastCreate.Name != "Pending Seat" {
		t.Fatalf("create input = %#v", mutation.lastCreate)
	}
}

func TestHandleParticipantEditRendersOwnedEditPage(t *testing.T) {
	t.Parallel()

	h, _ := newParticipantHandler(t)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipantEdit("camp-1", "p-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleParticipantEdit(rr, req, "camp-1", "p-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-participant-edit-page="true"`,
		`value="Owner"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing edit-page marker %q: %q", marker, body)
		}
	}
}

func TestHandleParticipantUpdateRedirectsAndForwardsInput(t *testing.T) {
	t.Parallel()

	h, mutation := newParticipantHandler(t)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignParticipantEdit("camp-1", "p-1"), strings.NewReader("name=Owner&role=gm&pronouns=they%2Fthem&campaign_access=owner"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleParticipantUpdate(rr, req, "camp-1", "p-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignParticipants("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignParticipants("camp-1"))
	}
	if mutation.lastUpdate.ParticipantID != "p-1" || mutation.lastUpdate.Pronouns != "they/them" {
		t.Fatalf("update input = %#v", mutation.lastUpdate)
	}
}
