package sessions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/language"

	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaigndetail "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/detail"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

type sessionWorkspaceService struct {
	workspace campaignapp.CampaignWorkspace
}

func (s sessionWorkspaceService) CampaignName(context.Context, string) string {
	return s.workspace.Name
}
func (s sessionWorkspaceService) CampaignWorkspace(context.Context, string) (campaignapp.CampaignWorkspace, error) {
	return s.workspace, nil
}

type sessionReads struct {
	sessions  []campaignapp.CampaignSession
	readiness campaignapp.CampaignSessionReadiness
}

func (r sessionReads) CampaignSessions(context.Context, string) ([]campaignapp.CampaignSession, error) {
	return append([]campaignapp.CampaignSession(nil), r.sessions...), nil
}
func (r sessionReads) CampaignSessionReadiness(context.Context, string, language.Tag) (campaignapp.CampaignSessionReadiness, error) {
	return r.readiness, nil
}

type sessionAuth struct{}

func (sessionAuth) RequireManageCampaign(context.Context, string) error     { return nil }
func (sessionAuth) RequireManageSession(context.Context, string) error      { return nil }
func (sessionAuth) RequireManageParticipants(context.Context, string) error { return nil }
func (sessionAuth) RequireManageInvites(context.Context, string) error      { return nil }
func (sessionAuth) RequireMutateCharacters(context.Context, string) error   { return nil }

type sessionMutation struct {
	lastStart campaignapp.StartSessionInput
	lastEnd   campaignapp.EndSessionInput
}

func (m *sessionMutation) StartSession(_ context.Context, _ string, input campaignapp.StartSessionInput) error {
	m.lastStart = input
	return nil
}
func (m *sessionMutation) EndSession(_ context.Context, _ string, input campaignapp.EndSessionInput) error {
	m.lastEnd = input
	return nil
}

type sessionSync struct {
	lastStartedCampaign string
	lastEndedCampaign   string
	lastUserID          string
}

func (s *sessionSync) ProfileSaved(context.Context, string)            {}
func (s *sessionSync) CampaignCreated(context.Context, string, string) {}
func (s *sessionSync) SessionStarted(_ context.Context, userID, campaignID string) {
	s.lastUserID = userID
	s.lastStartedCampaign = campaignID
}
func (s *sessionSync) SessionEnded(_ context.Context, userID, campaignID string) {
	s.lastUserID = userID
	s.lastEndedCampaign = campaignID
}
func (s *sessionSync) InviteChanged(context.Context, []string, string) {}

func testPlayLaunchGrantConfig() playlaunchgrant.Config {
	return playlaunchgrant.Config{
		Issuer:   "issuer-test",
		Audience: "audience-test",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      time.Minute,
		Now:      func() time.Time { return time.Date(2026, 3, 13, 16, 0, 0, 0, time.UTC) },
	}
}

func newSessionsHandler(t *testing.T) (Handler, *sessionMutation, *sessionSync) {
	t.Helper()

	base := modulehandler.NewBase(
		func(*http.Request) string { return "user-1" },
		func(*http.Request) string { return "en-US" },
		func(*http.Request) module.Viewer { return module.Viewer{} },
	)
	sync := &sessionSync{}
	detailHandler := campaigndetail.NewHandler(
		campaigndetail.NewSupport(base, requestmeta.SchemePolicy{}, sync),
		campaigndetail.PageServices{
			Workspace: sessionWorkspaceService{workspace: campaignapp.CampaignWorkspace{
				ID:     "camp-1",
				Name:   "The Guildhouse",
				System: "Daggerheart",
			}},
			SessionReads: sessionReads{
				sessions: []campaignapp.CampaignSession{{ID: "sess-1", Name: "Session One", Status: "Active"}},
				readiness: campaignapp.CampaignSessionReadiness{
					Ready: false,
					Blockers: []campaignapp.CampaignSessionReadinessBlocker{{
						Code:    "SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED",
						Message: "Campaign readiness requires at least one AI-controlled GM participant for AI GM mode",
					}},
				},
			},
			Authorization: sessionAuth{},
		},
	)
	mutation := &sessionMutation{}
	return NewHandler(detailHandler, HandlerServices{mutation: mutation}, "8094", testPlayLaunchGrantConfig()), mutation, sync
}

func TestHandleSessionsRendersOwnedSessionsPage(t *testing.T) {
	t.Parallel()

	h, _, _ := newSessionsHandler(t)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignSessions("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleSessions(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-sessions-header="true"`,
		`data-campaign-session-card-id="sess-1"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing sessions marker %q: %q", marker, body)
		}
	}
}

func TestHandleSessionCreatePageRendersReadinessBlockers(t *testing.T) {
	t.Parallel()

	h, _, _ := newSessionsHandler(t)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignSessionCreate("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleSessionCreatePage(rr, req, "camp-1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, marker := range []string{
		`data-campaign-session-readiness-blocked="true"`,
		`data-campaign-session-readiness-blocker-code="SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing readiness marker %q: %q", marker, body)
		}
	}
}

func TestHandleSessionCreateRedirectsAndNotifiesSync(t *testing.T) {
	t.Parallel()

	h, mutation, sync := newSessionsHandler(t)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignSessionCreate("camp-1"), strings.NewReader("name=Session+Two"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleSessionCreate(rr, req, "camp-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignSessions("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignSessions("camp-1"))
	}
	if mutation.lastStart.Name != "Session Two" {
		t.Fatalf("start input = %#v", mutation.lastStart)
	}
	if sync.lastStartedCampaign != "camp-1" || sync.lastUserID != "user-1" {
		t.Fatalf("sync = %#v", sync)
	}
}

func TestHandleSessionEndRedirectsAndNotifiesSync(t *testing.T) {
	t.Parallel()

	h, mutation, sync := newSessionsHandler(t)
	req := httptest.NewRequest(http.MethodPost, routepath.AppCampaignSessionEnd("camp-1"), strings.NewReader("session_id=sess-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	h.HandleSessionEnd(rr, req, "camp-1")

	if rr.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != routepath.AppCampaignSessions("camp-1") {
		t.Fatalf("Location = %q, want %q", got, routepath.AppCampaignSessions("camp-1"))
	}
	if mutation.lastEnd.SessionID != "sess-1" {
		t.Fatalf("end input = %#v", mutation.lastEnd)
	}
	if sync.lastEndedCampaign != "camp-1" || sync.lastUserID != "user-1" {
		t.Fatalf("sync = %#v", sync)
	}
}

func TestHandleSessionDetailReturnsNotFoundWhenSessionMissing(t *testing.T) {
	t.Parallel()

	h, _, _ := newSessionsHandler(t)
	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignSession("camp-1", "missing"), nil)
	rr := httptest.NewRecorder()

	h.HandleSessionDetail(rr, req, "camp-1", "missing")

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleGameRedirectsIntoPlaySurface(t *testing.T) {
	t.Parallel()

	h, _, _ := newSessionsHandler(t)
	req := httptest.NewRequest(http.MethodGet, "http://example.com"+routepath.AppCampaignGame("camp-1"), nil)
	rr := httptest.NewRecorder()

	h.HandleGame(rr, req, "camp-1")

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	location := rr.Header().Get("Location")
	if !strings.Contains(location, "/campaigns/camp-1?launch=") {
		t.Fatalf("Location = %q, want play launch redirect", location)
	}
}
