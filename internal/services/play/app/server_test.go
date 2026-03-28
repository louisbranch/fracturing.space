package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"google.golang.org/grpc"
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
			Interaction: fakePlayInteractionClient{
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
			Transcripts:  &fakeTranscriptStore{},
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
			Interaction: fakePlayInteractionClient{
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
			Transcripts:  &fakeTranscriptStore{},
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
			Interaction: fakePlayInteractionClient{
				response: &gamev1.GetInteractionStateResponse{State: &gamev1.InteractionState{
					CampaignId: "c1",
					Viewer:     &gamev1.InteractionViewer{ParticipantId: "p1", Name: "Avery"},
				}},
			},
			Campaign:     fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{}},
			System:       fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{}},
			Participants: fakePlayParticipantClient{response: &gamev1.ListParticipantsResponse{}},
			Characters:   fakePlayCharacterClient{listResponse: &gamev1.ListCharactersResponse{}},
			Transcripts:  &fakeTranscriptStore{},
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
			Interaction: fakePlayInteractionClient{
				response: &gamev1.GetInteractionStateResponse{State: &gamev1.InteractionState{
					CampaignId: "c1",
					Viewer:     &gamev1.InteractionViewer{ParticipantId: "p1", Name: "Avery"},
				}},
			},
			Campaign:     fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{}},
			System:       fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{}},
			Participants: fakePlayParticipantClient{response: &gamev1.ListParticipantsResponse{}},
			Characters:   fakePlayCharacterClient{listResponse: &gamev1.ListCharactersResponse{}},
			Transcripts:  &fakeTranscriptStore{},
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

func testPlayLaunchGrantConfig(t *testing.T) playlaunchgrant.Config {
	t.Helper()
	now := time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC)
	return playlaunchgrant.Config{
		Issuer:   "fracturing-space-web",
		Audience: "fracturing-space-play",
		HMACKey:  []byte("0123456789abcdef0123456789abcdef"),
		TTL:      2 * time.Minute,
		Now: func() time.Time {
			return now
		},
	}
}

type fakePlayAuthClient struct {
	sessions         map[string]string
	createdSessionID string
	createdUserID    string
}

func (f *fakePlayAuthClient) CreateWebSession(_ context.Context, req *authv1.CreateWebSessionRequest, _ ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	if f.sessions == nil {
		f.sessions = map[string]string{}
	}
	sessionID := strings.TrimSpace(f.createdSessionID)
	if sessionID == "" {
		sessionID = "ps-1"
	}
	f.createdUserID = strings.TrimSpace(req.GetUserId())
	f.sessions[sessionID] = f.createdUserID
	return &authv1.CreateWebSessionResponse{
		Session: &authv1.WebSession{Id: sessionID, UserId: f.createdUserID},
	}, nil
}

func (f *fakePlayAuthClient) GetWebSession(_ context.Context, req *authv1.GetWebSessionRequest, _ ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	sessionID := strings.TrimSpace(req.GetSessionId())
	userID := strings.TrimSpace(f.sessions[sessionID])
	if userID == "" {
		return nil, context.Canceled
	}
	return &authv1.GetWebSessionResponse{
		Session: &authv1.WebSession{Id: sessionID, UserId: userID},
	}, nil
}

type fakePlayInteractionClient struct {
	response *gamev1.GetInteractionStateResponse
	err      error
}

func (f fakePlayInteractionClient) GetInteractionState(context.Context, *gamev1.GetInteractionStateRequest, ...grpc.CallOption) (*gamev1.GetInteractionStateResponse, error) {
	return f.response, f.err
}

func (f fakePlayInteractionClient) ActivateScene(context.Context, *gamev1.ActivateSceneRequest, ...grpc.CallOption) (*gamev1.ActivateSceneResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) OpenScenePlayerPhase(context.Context, *gamev1.OpenScenePlayerPhaseRequest, ...grpc.CallOption) (*gamev1.OpenScenePlayerPhaseResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) SubmitScenePlayerAction(context.Context, *gamev1.SubmitScenePlayerActionRequest, ...grpc.CallOption) (*gamev1.SubmitScenePlayerActionResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) YieldScenePlayerPhase(context.Context, *gamev1.YieldScenePlayerPhaseRequest, ...grpc.CallOption) (*gamev1.YieldScenePlayerPhaseResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) WithdrawScenePlayerYield(context.Context, *gamev1.WithdrawScenePlayerYieldRequest, ...grpc.CallOption) (*gamev1.WithdrawScenePlayerYieldResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) InterruptScenePlayerPhase(context.Context, *gamev1.InterruptScenePlayerPhaseRequest, ...grpc.CallOption) (*gamev1.InterruptScenePlayerPhaseResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) RecordSceneGMInteraction(context.Context, *gamev1.RecordSceneGMInteractionRequest, ...grpc.CallOption) (*gamev1.RecordSceneGMInteractionResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) ResolveScenePlayerReview(context.Context, *gamev1.ResolveScenePlayerReviewRequest, ...grpc.CallOption) (*gamev1.ResolveScenePlayerReviewResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) OpenSessionOOC(context.Context, *gamev1.OpenSessionOOCRequest, ...grpc.CallOption) (*gamev1.OpenSessionOOCResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) PostSessionOOC(context.Context, *gamev1.PostSessionOOCRequest, ...grpc.CallOption) (*gamev1.PostSessionOOCResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) MarkOOCReadyToResume(context.Context, *gamev1.MarkOOCReadyToResumeRequest, ...grpc.CallOption) (*gamev1.MarkOOCReadyToResumeResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) ClearOOCReadyToResume(context.Context, *gamev1.ClearOOCReadyToResumeRequest, ...grpc.CallOption) (*gamev1.ClearOOCReadyToResumeResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) ResolveSessionOOC(context.Context, *gamev1.ResolveSessionOOCRequest, ...grpc.CallOption) (*gamev1.ResolveSessionOOCResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) SetSessionGMAuthority(context.Context, *gamev1.SetSessionGMAuthorityRequest, ...grpc.CallOption) (*gamev1.SetSessionGMAuthorityResponse, error) {
	return nil, nil
}
func (f fakePlayInteractionClient) RetryAIGMTurn(context.Context, *gamev1.RetryAIGMTurnRequest, ...grpc.CallOption) (*gamev1.RetryAIGMTurnResponse, error) {
	return nil, nil
}

type fakePlayCampaignClient struct {
	response *gamev1.GetCampaignResponse
	err      error
}

func (f fakePlayCampaignClient) GetCampaign(context.Context, *gamev1.GetCampaignRequest, ...grpc.CallOption) (*gamev1.GetCampaignResponse, error) {
	return f.response, f.err
}

type fakePlaySystemClient struct {
	response *gamev1.GetGameSystemResponse
	err      error
}

func (f fakePlaySystemClient) GetGameSystem(context.Context, *gamev1.GetGameSystemRequest, ...grpc.CallOption) (*gamev1.GetGameSystemResponse, error) {
	return f.response, f.err
}

type fakePlayParticipantClient struct {
	response *gamev1.ListParticipantsResponse
	err      error
}

func (f fakePlayParticipantClient) ListParticipants(context.Context, *gamev1.ListParticipantsRequest, ...grpc.CallOption) (*gamev1.ListParticipantsResponse, error) {
	return f.response, f.err
}

type fakePlayCharacterClient struct {
	listResponse  *gamev1.ListCharactersResponse
	listErr       error
	sheetResponse *gamev1.GetCharacterSheetResponse
	sheetErr      error
}

func (f fakePlayCharacterClient) ListCharacters(context.Context, *gamev1.ListCharactersRequest, ...grpc.CallOption) (*gamev1.ListCharactersResponse, error) {
	return f.listResponse, f.listErr
}

func (f fakePlayCharacterClient) GetCharacterSheet(context.Context, *gamev1.GetCharacterSheetRequest, ...grpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
	return f.sheetResponse, f.sheetErr
}

type fakeTranscriptStore struct{}

func (f *fakeTranscriptStore) LatestSequence(context.Context, transcript.Scope) (int64, error) {
	return 0, nil
}

func (f *fakeTranscriptStore) AppendMessage(context.Context, transcript.AppendRequest) (transcript.AppendResult, error) {
	return transcript.AppendResult{}, nil
}

func (f *fakeTranscriptStore) HistoryAfter(context.Context, transcript.HistoryAfterQuery) ([]transcript.Message, error) {
	return nil, nil
}

func (f *fakeTranscriptStore) HistoryBefore(context.Context, transcript.HistoryBeforeQuery) ([]transcript.Message, error) {
	return nil, nil
}

func (f *fakeTranscriptStore) Close() error {
	return nil
}
