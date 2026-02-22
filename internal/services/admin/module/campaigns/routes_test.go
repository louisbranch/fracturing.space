package campaigns

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	lastCall      string
	lastCampaign  string
	lastCharacter string
	lastSession   string
}

func (f *fakeService) HandleCampaignsPage(http.ResponseWriter, *http.Request) {
	f.lastCall = "campaigns_page"
}

func (f *fakeService) HandleCampaignsTable(http.ResponseWriter, *http.Request) {
	f.lastCall = "campaigns_table"
}

func (f *fakeService) HandleCampaignDetail(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "campaign_detail"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleCharactersList(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "characters_list"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleCharactersTable(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "characters_table"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleCharacterSheet(_ http.ResponseWriter, _ *http.Request, campaignID string, characterID string) {
	f.lastCall = "character_sheet"
	f.lastCampaign = campaignID
	f.lastCharacter = characterID
}

func (f *fakeService) HandleCharacterActivity(_ http.ResponseWriter, _ *http.Request, campaignID string, characterID string) {
	f.lastCall = "character_activity"
	f.lastCampaign = campaignID
	f.lastCharacter = characterID
}

func (f *fakeService) HandleParticipantsList(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "participants_list"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleParticipantsTable(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "participants_table"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleInvitesList(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "invites_list"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleInvitesTable(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "invites_table"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleSessionsList(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "sessions_list"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleSessionsTable(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "sessions_table"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleSessionDetail(_ http.ResponseWriter, _ *http.Request, campaignID string, sessionID string) {
	f.lastCall = "session_detail"
	f.lastCampaign = campaignID
	f.lastSession = sessionID
}

func (f *fakeService) HandleSessionEvents(_ http.ResponseWriter, _ *http.Request, campaignID string, sessionID string) {
	f.lastCall = "session_events"
	f.lastCampaign = campaignID
	f.lastSession = sessionID
}

func (f *fakeService) HandleEventLog(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "event_log"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleEventLogTable(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "event_log_table"
	f.lastCampaign = campaignID
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path          string
		wantCode      int
		wantCall      string
		wantCampaign  string
		wantSession   string
		wantCharacter string
	}{
		{path: "/campaigns", wantCode: http.StatusOK, wantCall: "campaigns_page"},
		{path: "/campaigns/table", wantCode: http.StatusOK, wantCall: "campaigns_table"},
		{path: "/campaigns/camp-1", wantCode: http.StatusOK, wantCall: "campaign_detail", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/characters", wantCode: http.StatusOK, wantCall: "characters_list", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/characters/table", wantCode: http.StatusOK, wantCall: "characters_table", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/characters/ch-1", wantCode: http.StatusOK, wantCall: "character_sheet", wantCampaign: "camp-1", wantCharacter: "ch-1"},
		{path: "/campaigns/camp-1/characters/ch-1/activity", wantCode: http.StatusOK, wantCall: "character_activity", wantCampaign: "camp-1", wantCharacter: "ch-1"},
		{path: "/campaigns/camp-1/participants", wantCode: http.StatusOK, wantCall: "participants_list", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/participants/table", wantCode: http.StatusOK, wantCall: "participants_table", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/invites", wantCode: http.StatusOK, wantCall: "invites_list", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/invites/table", wantCode: http.StatusOK, wantCall: "invites_table", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/sessions", wantCode: http.StatusOK, wantCall: "sessions_list", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/sessions/table", wantCode: http.StatusOK, wantCall: "sessions_table", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/sessions/s-1", wantCode: http.StatusOK, wantCall: "session_detail", wantCampaign: "camp-1", wantSession: "s-1"},
		{path: "/campaigns/camp-1/sessions/s-1/events", wantCode: http.StatusOK, wantCall: "session_events", wantCampaign: "camp-1", wantSession: "s-1"},
		{path: "/campaigns/camp-1/events", wantCode: http.StatusOK, wantCall: "event_log", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/events/table", wantCode: http.StatusOK, wantCall: "event_log_table", wantCampaign: "camp-1"},
		{path: "/campaigns/create", wantCode: http.StatusNotFound},
		{path: "/campaigns/camp-1/unknown", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastCampaign = ""
			svc.lastCharacter = ""
			svc.lastSession = ""

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
			if svc.lastCampaign != tc.wantCampaign {
				t.Fatalf("lastCampaign = %q, want %q", svc.lastCampaign, tc.wantCampaign)
			}
			if svc.lastSession != tc.wantSession {
				t.Fatalf("lastSession = %q, want %q", svc.lastSession, tc.wantSession)
			}
			if svc.lastCharacter != tc.wantCharacter {
				t.Fatalf("lastCharacter = %q, want %q", svc.lastCharacter, tc.wantCharacter)
			}
		})
	}
}

func TestHandleCampaignPathRedirectsTrailingSlash(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/", nil)
	rec := httptest.NewRecorder()

	HandleCampaignPath(rec, req, svc)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if location := rec.Header().Get("Location"); location != "/campaigns/camp-1" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-1")
	}
}
