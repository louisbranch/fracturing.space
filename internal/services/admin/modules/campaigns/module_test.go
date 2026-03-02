package campaigns

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

type fakeService struct {
	lastCall      string
	lastCampaign  string
	lastCharacter string
	lastSession   string
}

func (f *fakeService) HandleCampaignsPage(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "campaigns_page"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleCampaignsTable(w http.ResponseWriter, _ *http.Request) {
	f.lastCall = "campaigns_table"
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleCampaignDetail(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "campaign_detail"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleCharactersList(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "characters_list"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleCharactersTable(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "characters_table"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleCharacterSheet(w http.ResponseWriter, _ *http.Request, campaignID string, characterID string) {
	f.lastCall = "character_sheet"
	f.lastCampaign = campaignID
	f.lastCharacter = characterID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleCharacterActivity(w http.ResponseWriter, _ *http.Request, campaignID string, characterID string) {
	f.lastCall = "character_activity"
	f.lastCampaign = campaignID
	f.lastCharacter = characterID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleParticipantsList(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "participants_list"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleParticipantsTable(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "participants_table"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleInvitesList(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "invites_list"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleInvitesTable(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "invites_table"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleSessionsList(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "sessions_list"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleSessionsTable(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "sessions_table"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleSessionDetail(w http.ResponseWriter, _ *http.Request, campaignID string, sessionID string) {
	f.lastCall = "session_detail"
	f.lastCampaign = campaignID
	f.lastSession = sessionID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleSessionEvents(w http.ResponseWriter, _ *http.Request, campaignID string, sessionID string) {
	f.lastCall = "session_events"
	f.lastCampaign = campaignID
	f.lastSession = sessionID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleEventLog(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "event_log"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleEventLogTable(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "event_log_table"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func TestMount(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	m, err := New(svc).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if m.Prefix != routepath.CampaignsPrefix {
		t.Fatalf("prefix = %q, want %q", m.Prefix, routepath.CampaignsPrefix)
	}

	tests := []struct {
		path         string
		wantCode     int
		wantCall     string
		wantCampaign string
	}{
		{path: "/campaigns", wantCode: http.StatusNoContent, wantCall: "campaigns_page"},
		{path: "/campaigns/_rows", wantCode: http.StatusNoContent, wantCall: "campaigns_table"},
		{path: "/campaigns/camp-1/characters/_rows", wantCode: http.StatusNoContent, wantCall: "characters_table", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/events/_rows", wantCode: http.StatusNoContent, wantCall: "event_log_table", wantCampaign: "camp-1"},
		{path: "/campaigns/camp-1/characters/table", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastCampaign = ""
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			m.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
			if svc.lastCampaign != tc.wantCampaign {
				t.Fatalf("lastCampaign = %q, want %q", svc.lastCampaign, tc.wantCampaign)
			}
		})
	}
}

func TestMountNilService(t *testing.T) {
	t.Parallel()

	m, err := New(nil).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-1/events/_rows", nil)
	rec := httptest.NewRecorder()
	m.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
