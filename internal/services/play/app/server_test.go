package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
)

func TestHandleCampaignShellRedirectsToWebWhenSessionAndGrantAreMissing(t *testing.T) {
	t.Parallel()

	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/campaigns/c1", nil)
	req.SetPathValue("campaignID", "c1")
	rr := httptest.NewRecorder()

	server.handleCampaignShell(rr, req, testPlayLaunchGrantConfig(t))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "http://example.com/app/campaigns/c1" {
		t.Fatalf("Location = %q, want %q", got, "http://example.com/app/campaigns/c1")
	}
}

func TestHandleCampaignShellExchangesLaunchGrantForPlaySession(t *testing.T) {
	t.Parallel()

	auth := &fakePlayAuthClient{createdSessionID: "ps-1"}
	server := &Server{deps: Dependencies{Auth: auth}}
	grantCfg := testPlayLaunchGrantConfig(t)
	grant, _, err := playlaunchgrant.Issue(grantCfg, playlaunchgrant.IssueInput{
		GrantID:    "grant-1",
		CampaignID: "c1",
		UserID:     "user-1",
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/campaigns/c1?launch="+grant, nil)
	req.SetPathValue("campaignID", "c1")
	rr := httptest.NewRecorder()

	server.handleCampaignShell(rr, req, grantCfg)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "http://play.example.com/campaigns/c1" {
		t.Fatalf("Location = %q, want %q", got, "http://play.example.com/campaigns/c1")
	}
	if auth.createdUserID != "user-1" {
		t.Fatalf("CreateWebSession user = %q, want %q", auth.createdUserID, "user-1")
	}
	cookie := rr.Result().Cookies()
	if len(cookie) == 0 {
		t.Fatal("expected play_session cookie to be set")
	}
	if cookie[0].Name != playSessionCookieName || cookie[0].Value != "ps-1" {
		t.Fatalf("cookie = %s=%s, want %s=%s", cookie[0].Name, cookie[0].Value, playSessionCookieName, "ps-1")
	}
}

func TestHandleCampaignShellLaunchGrantOverridesExistingPlaySession(t *testing.T) {
	t.Parallel()

	auth := &fakePlayAuthClient{
		sessions:         map[string]string{"stale-session": "stale-user"},
		createdSessionID: "ps-2",
	}
	server := &Server{deps: Dependencies{Auth: auth}}
	grantCfg := testPlayLaunchGrantConfig(t)
	grant, _, err := playlaunchgrant.Issue(grantCfg, playlaunchgrant.IssueInput{
		GrantID:    "grant-2",
		CampaignID: "c1",
		UserID:     "user-1",
	})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/campaigns/c1?launch="+grant, nil)
	req.SetPathValue("campaignID", "c1")
	req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "stale-session"})
	rr := httptest.NewRecorder()

	server.handleCampaignShell(rr, req, grantCfg)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusSeeOther)
	}
	if auth.createdUserID != "user-1" {
		t.Fatalf("CreateWebSession user = %q, want %q", auth.createdUserID, "user-1")
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Value != "ps-2" {
		t.Fatalf("cookies = %#v, want play_session=ps-2", cookies)
	}
}

func TestHandleCampaignShellRendersSPAShellForExistingPlaySession(t *testing.T) {
	t.Parallel()

	server := &Server{
		deps: Dependencies{
			Auth: &fakePlayAuthClient{sessions: map[string]string{"ps-1": "user-1"}},
			Interaction: stubInteractionClient{
				response: &gamev1.GetInteractionStateResponse{State: &gamev1.InteractionState{
					CampaignId:   "c1",
					CampaignName: "The Guildhouse",
					Viewer:       &gamev1.InteractionViewer{ParticipantId: "p1", Name: "Avery", Role: gamev1.ParticipantRole_PLAYER},
					ActiveSession: &gamev1.InteractionSession{
						SessionId: "s1",
						Name:      "Session One",
					},
				}},
			},
			Campaign:     fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{}},
			System:       fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{}},
			Participants: fakePlayParticipantClient{response: &gamev1.ListParticipantsResponse{}},
			Characters:   fakePlayCharacterClient{listResponse: &gamev1.ListCharactersResponse{}},
			Transcripts:  &stubTranscriptStore{},
		},
		shellAssets: shellAssets{devServerURL: "http://localhost:5173"},
	}

	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/campaigns/c1", nil)
	req.SetPathValue("campaignID", "c1")
	req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
	rr := httptest.NewRecorder()

	server.handleCampaignShell(rr, req, testPlayLaunchGrantConfig(t))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<div id=\"root\"></div>") {
		t.Fatalf("body missing SPA root: %q", body)
	}
	if !strings.Contains(body, "/src/main.tsx") {
		t.Fatalf("body missing dev entrypoint: %q", body)
	}
	if !strings.Contains(body, "http://example.com/app/campaigns/c1") {
		t.Fatalf("body missing web overview back url: %q", body)
	}
}

func TestHandleRootShellRendersSPAShellWithoutSession(t *testing.T) {
	t.Parallel()

	server := &Server{
		shellAssets: shellAssets{devServerURL: "http://localhost:5173"},
	}

	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/", nil)
	rr := httptest.NewRecorder()

	server.handleRootShell(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`<div id="root"></div>`,
		`/src/main.tsx`,
		`type="application/json"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %q", want, body)
		}
	}
	if strings.Contains(body, `/api/campaigns/`) {
		t.Fatalf("root shell should not include campaign bootstrap wiring: %q", body)
	}
}

func TestHandleBootstrapReturnsPlayContract(t *testing.T) {
	t.Parallel()

	server := &Server{
		deps: Dependencies{
			Auth: &fakePlayAuthClient{sessions: map[string]string{"ps-1": "user-1"}},
			Interaction: stubInteractionClient{
				response: &gamev1.GetInteractionStateResponse{State: &gamev1.InteractionState{
					CampaignId:   "c1",
					CampaignName: "The Guildhouse",
					Viewer:       &gamev1.InteractionViewer{ParticipantId: "p1", Name: "Avery"},
					ActiveSession: &gamev1.InteractionSession{
						SessionId: "s1",
						Name:      "Session One",
					},
				}},
			},
			Campaign:     fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{}},
			System:       fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{}},
			Participants: fakePlayParticipantClient{response: &gamev1.ListParticipantsResponse{}},
			Characters:   fakePlayCharacterClient{listResponse: &gamev1.ListCharactersResponse{}},
			Transcripts:  &stubTranscriptStore{},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/bootstrap", nil)
	req.SetPathValue("campaignID", "c1")
	req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
	rr := httptest.NewRecorder()

	server.handleBootstrap(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var payload playprotocol.Bootstrap
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.CampaignID != "c1" {
		t.Fatalf("campaign_id = %q, want %q", payload.CampaignID, "c1")
	}
	if payload.Viewer == nil || payload.Viewer.ParticipantID != "p1" || payload.Viewer.Name != "Avery" {
		t.Fatalf("viewer = %#v", payload.Viewer)
	}
	if payload.InteractionState.CampaignID != "c1" || payload.InteractionState.CampaignName != "The Guildhouse" {
		t.Fatalf("interaction_state = %#v", payload.InteractionState)
	}
	if payload.InteractionState.ActiveSession == nil || payload.InteractionState.ActiveSession.SessionID != "s1" {
		t.Fatalf("active_session = %#v", payload.InteractionState.ActiveSession)
	}
	if payload.Chat.HistoryURL != "/api/campaigns/c1/chat/history" {
		t.Fatalf("history_url = %q", payload.Chat.HistoryURL)
	}
	if payload.Realtime.URL != "/realtime" {
		t.Fatalf("realtime.url = %q", payload.Realtime.URL)
	}
}

func TestHandleBootstrapUsesCookieScopedRealtimeURL(t *testing.T) {
	t.Parallel()

	server := &Server{
		deps: Dependencies{
			Auth: &fakePlayAuthClient{sessions: map[string]string{"ps-1": "user-1"}},
			Interaction: stubInteractionClient{
				response: &gamev1.GetInteractionStateResponse{State: &gamev1.InteractionState{
					CampaignId: "c1",
					Viewer:     &gamev1.InteractionViewer{ParticipantId: "p1", Name: "Avery"},
				}},
			},
			Campaign:     fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{}},
			System:       fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{}},
			Participants: fakePlayParticipantClient{response: &gamev1.ListParticipantsResponse{}},
			Characters:   fakePlayCharacterClient{listResponse: &gamev1.ListCharactersResponse{}},
			Transcripts:  &stubTranscriptStore{},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/bootstrap", nil)
	req.SetPathValue("campaignID", "c1")
	req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
	rr := httptest.NewRecorder()

	server.handleBootstrap(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var payload playprotocol.Bootstrap
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.InteractionState.Viewer == nil || payload.InteractionState.Viewer.ParticipantID != "p1" {
		t.Fatalf("interaction viewer = %#v", payload.InteractionState.Viewer)
	}
	if payload.Realtime.URL != "/realtime" {
		t.Fatalf("realtime.url = %q, want %q", payload.Realtime.URL, "/realtime")
	}
}

func TestHandleBootstrapRejectsPlaySessionQueryParamWithoutCookie(t *testing.T) {
	t.Parallel()

	server := &Server{
		deps: Dependencies{
			Auth: &fakePlayAuthClient{sessions: map[string]string{"ps-1": "user-1"}},
			Interaction: stubInteractionClient{
				response: &gamev1.GetInteractionStateResponse{State: &gamev1.InteractionState{
					CampaignId: "c1",
					Viewer:     &gamev1.InteractionViewer{ParticipantId: "p1", Name: "Avery"},
				}},
			},
			Campaign:     fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{}},
			System:       fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{}},
			Participants: fakePlayParticipantClient{response: &gamev1.ListParticipantsResponse{}},
			Characters:   fakePlayCharacterClient{listResponse: &gamev1.ListCharactersResponse{}},
			Transcripts:  &stubTranscriptStore{},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/bootstrap?play_session=ps-1", nil)
	req.SetPathValue("campaignID", "c1")
	rr := httptest.NewRecorder()

	server.handleBootstrap(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestListenAndServeGuards(t *testing.T) {
	t.Parallel()

	if err := (*Server)(nil).ListenAndServe(context.Background()); err == nil {
		t.Fatal("ListenAndServe(nil) error = nil, want non-nil")
	}
	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	if err := server.ListenAndServe(nil); err == nil {
		t.Fatal("ListenAndServe(nil context) error = nil, want non-nil")
	}
}
