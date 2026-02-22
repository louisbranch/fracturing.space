package campaigns

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	lastCall        string
	lastCampaignID  string
	lastSessionID   string
	lastCharacterID string
}

func (f *fakeService) HandleCampaigns(http.ResponseWriter, *http.Request) {
	f.lastCall = "campaigns"
}

func (f *fakeService) HandleCampaignCreate(http.ResponseWriter, *http.Request) {
	f.lastCall = "campaign_create"
}

func (f *fakeService) HandleCampaignOverview(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "overview"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignSessions(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "sessions"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignSessionStart(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "sessions_start"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignSessionEnd(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "sessions_end"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignSessionDetail(_ http.ResponseWriter, _ *http.Request, campaignID string, sessionID string) {
	f.lastCall = "sessions_detail"
	f.lastCampaignID = campaignID
	f.lastSessionID = sessionID
}

func (f *fakeService) HandleCampaignParticipants(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "participants"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignParticipantUpdate(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "participants_update"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignCharacters(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "characters"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignCharacterCreate(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "characters_create"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignCharacterUpdate(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "characters_update"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignCharacterControl(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "characters_control"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignCharacterDetail(_ http.ResponseWriter, _ *http.Request, campaignID string, characterID string) {
	f.lastCall = "characters_detail"
	f.lastCampaignID = campaignID
	f.lastCharacterID = characterID
}

func (f *fakeService) HandleCampaignInvites(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "invites"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignInviteCreate(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "invites_create"
	f.lastCampaignID = campaignID
}

func (f *fakeService) HandleCampaignInviteRevoke(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "invites_revoke"
	f.lastCampaignID = campaignID
}

func TestHandleCampaignDetailPathDispatchesRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		path            string
		wantCall        string
		wantCode        int
		wantCampaignID  string
		wantSessionID   string
		wantCharacterID string
	}{
		{name: "overview", path: "/app/campaigns/camp-1", wantCall: "overview", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "sessions", path: "/app/campaigns/camp-1/sessions", wantCall: "sessions", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "session start", path: "/app/campaigns/camp-1/sessions/start", wantCall: "sessions_start", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "session end", path: "/app/campaigns/camp-1/sessions/end", wantCall: "sessions_end", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "session detail", path: "/app/campaigns/camp-1/sessions/s1", wantCall: "sessions_detail", wantCode: http.StatusOK, wantCampaignID: "camp-1", wantSessionID: "s1"},
		{name: "participants", path: "/app/campaigns/camp-1/participants", wantCall: "participants", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "participants update", path: "/app/campaigns/camp-1/participants/update", wantCall: "participants_update", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "characters", path: "/app/campaigns/camp-1/characters", wantCall: "characters", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "characters create", path: "/app/campaigns/camp-1/characters/create", wantCall: "characters_create", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "characters update", path: "/app/campaigns/camp-1/characters/update", wantCall: "characters_update", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "characters control", path: "/app/campaigns/camp-1/characters/control", wantCall: "characters_control", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "character detail", path: "/app/campaigns/camp-1/characters/char-1", wantCall: "characters_detail", wantCode: http.StatusOK, wantCampaignID: "camp-1", wantCharacterID: "char-1"},
		{name: "invites", path: "/app/campaigns/camp-1/invites", wantCall: "invites", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "invites create", path: "/app/campaigns/camp-1/invites/create", wantCall: "invites_create", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "invites revoke", path: "/app/campaigns/camp-1/invites/revoke", wantCall: "invites_revoke", wantCode: http.StatusOK, wantCampaignID: "camp-1"},
		{name: "missing campaign id", path: "/app/campaigns//sessions", wantCall: "", wantCode: http.StatusNotFound},
		{name: "not found", path: "/app/campaigns/camp-1/unknown", wantCall: "", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := &fakeService{}
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			HandleCampaignDetailPath(rec, req, svc)

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
			if svc.lastCampaignID != tc.wantCampaignID {
				t.Fatalf("lastCampaignID = %q, want %q", svc.lastCampaignID, tc.wantCampaignID)
			}
			if svc.lastSessionID != tc.wantSessionID {
				t.Fatalf("lastSessionID = %q, want %q", svc.lastSessionID, tc.wantSessionID)
			}
			if svc.lastCharacterID != tc.wantCharacterID {
				t.Fatalf("lastCharacterID = %q, want %q", svc.lastCharacterID, tc.wantCharacterID)
			}
		})
	}
}

func TestHandleCampaignDetailPathRedirectsTrailingSlash(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-1/", nil)
	rec := httptest.NewRecorder()

	HandleCampaignDetailPath(rec, req, svc)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if location := rec.Header().Get("Location"); location != "/app/campaigns/camp-1" {
		t.Fatalf("location = %q, want %q", location, "/app/campaigns/camp-1")
	}
}

func TestRegisterRoutesWiresCampaignEndpoints(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path     string
		wantCall string
	}{
		{path: "/app/campaigns", wantCall: "campaigns"},
		{path: "/app/campaigns/create", wantCall: "campaign_create"},
		{path: "/app/campaigns/camp-1", wantCall: "overview"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			svc.lastCall = ""

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
		})
	}
}
