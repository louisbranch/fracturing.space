package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	web2 "github.com/louisbranch/fracturing.space/internal/services/web2"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/sessioncookie"
	"google.golang.org/grpc"
)

func TestCutoverSmokeMatrixAuthenticatedJourneyParity(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"active":true,"user_id":"user-123"}`))
	}))
	t.Cleanup(authServer.Close)

	legacyParticipantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{"": {
			Participants: []*statev1.Participant{{
				Id:             "part-owner",
				CampaignId:     "camp-123",
				UserId:         "user-123",
				Name:           "Owner",
				CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
			}},
		}},
	}
	legacyHandler := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:           newSessionStore(),
		pendingFlows:       newPendingFlowStore(),
		campaignClient:     &fakeWebCampaignClient{response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "camp-123", Name: "Skyfall"}}}, getResponse: &statev1.GetCampaignResponse{Campaign: &statev1.Campaign{Id: "camp-123", Name: "Skyfall"}}},
		sessionClient:      &fakeWebSessionClient{response: &statev1.ListSessionsResponse{Sessions: []*statev1.Session{{Id: "sess-1", CampaignId: "camp-123", Name: "Session One"}}}},
		participantClient:  legacyParticipantClient,
		characterClient:    &fakeWebCharacterClient{response: &statev1.ListCharactersResponse{Characters: []*statev1.Character{{Id: "char-1", CampaignId: "camp-123", Name: "Mira"}}}},
		inviteClient:       &fakeWebInviteClient{response: &statev1.ListPendingInvitesForUserResponse{}},
		notificationClient: &fakeWebNotificationClient{listResp: &notificationsv1.ListNotificationsResponse{}},
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   legacyParticipantClient,
		},
	}
	legacySessionID := legacyHandler.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	modernAuthClient := newModernSmokeAuthClient()
	modernHandler, err := web2.NewHandler(web2.Config{
		EnableExperimentalModules: false,
		AuthClient:                modernAuthClient,
		CampaignClient:            modernSmokeCampaignClient{},
		ParticipantClient:         modernSmokeParticipantClient{},
		AccountClient:             modernSmokeAccountClient{},
		ConnectionsClient: &fakeConnectionsClient{getUserProfileResp: &connectionsv1.GetUserProfileResponse{UserProfile: &connectionsv1.UserProfile{
			Username: "adventurer",
			Name:     "Adventurer",
		}}},
		CredentialClient: modernSmokeCredentialClient{},
	})
	if err != nil {
		t.Fatalf("web2.NewHandler() error = %v", err)
	}
	modernSessionResp, err := modernAuthClient.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("CreateWebSession() error = %v", err)
	}
	modernSessionID := modernSessionResp.GetSession().GetId()

	testCases := []struct {
		name         string
		method       string
		legacyPath   string
		modernPath   string
		legacyStatus int
		modernStatus int
		legacyServe  func(http.ResponseWriter, *http.Request)
	}{
		{
			name:         "campaign list is available when authenticated",
			method:       http.MethodGet,
			legacyPath:   "/app/campaigns",
			modernPath:   "/app/campaigns/",
			legacyStatus: http.StatusOK,
			modernStatus: http.StatusOK,
			legacyServe:  legacyHandler.handleAppCampaigns,
		},
		{
			name:         "campaign detail is available when authenticated",
			method:       http.MethodGet,
			legacyPath:   "/app/campaigns/camp-123",
			modernPath:   "/app/campaigns/c1",
			legacyStatus: http.StatusOK,
			modernStatus: http.StatusOK,
			legacyServe:  legacyHandler.handleAppCampaignDetail,
		},
		{
			name:         "invites parity gap remains explicit",
			method:       http.MethodGet,
			legacyPath:   "/app/invites",
			modernPath:   "/app/invites",
			legacyStatus: http.StatusOK,
			modernStatus: http.StatusNotFound,
			legacyServe:  legacyHandler.handleAppInvites,
		},
		{
			name:         "notifications parity gap remains explicit",
			method:       http.MethodGet,
			legacyPath:   "/app/notifications",
			modernPath:   "/app/notifications",
			legacyStatus: http.StatusOK,
			modernStatus: http.StatusNotFound,
			legacyServe:  legacyHandler.handleAppNotifications,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			legacyReq := httptest.NewRequest(tc.method, tc.legacyPath, nil)
			legacyReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: legacySessionID})
			legacyRR := httptest.NewRecorder()
			tc.legacyServe(legacyRR, legacyReq)
			if legacyRR.Code != tc.legacyStatus {
				t.Fatalf("legacy status for %q = %d, want %d", tc.legacyPath, legacyRR.Code, tc.legacyStatus)
			}

			modernReq := httptest.NewRequest(tc.method, tc.modernPath, nil)
			modernReq.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: modernSessionID})
			modernRR := httptest.NewRecorder()
			modernHandler.ServeHTTP(modernRR, modernReq)
			if modernRR.Code != tc.modernStatus {
				t.Fatalf("modern status for %q = %d, want %d", tc.modernPath, modernRR.Code, tc.modernStatus)
			}
		})
	}
}

type modernSmokeAuthClient struct {
	mu       sync.Mutex
	sessions map[string]string
}

func newModernSmokeAuthClient() *modernSmokeAuthClient {
	return &modernSmokeAuthClient{sessions: map[string]string{}}
}

func (m *modernSmokeAuthClient) CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (m *modernSmokeAuthClient) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{}, nil
}

func (m *modernSmokeAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return &authv1.FinishPasskeyRegistrationResponse{}, nil
}

func (m *modernSmokeAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return &authv1.BeginPasskeyLoginResponse{}, nil
}

func (m *modernSmokeAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return &authv1.FinishPasskeyLoginResponse{}, nil
}

func (m *modernSmokeAuthClient) CreateWebSession(_ context.Context, req *authv1.CreateWebSessionRequest, _ ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := "smoke-session"
	m.sessions[id] = req.GetUserId()
	return &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: id, UserId: req.GetUserId()}, User: &authv1.User{Id: req.GetUserId()}}, nil
}

func (m *modernSmokeAuthClient) GetWebSession(_ context.Context, req *authv1.GetWebSessionRequest, _ ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	userID := m.sessions[req.GetSessionId()]
	return &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: req.GetSessionId(), UserId: userID}, User: &authv1.User{Id: userID}}, nil
}

func (m *modernSmokeAuthClient) RevokeWebSession(_ context.Context, req *authv1.RevokeWebSessionRequest, _ ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, req.GetSessionId())
	return &authv1.RevokeWebSessionResponse{}, nil
}

type modernSmokeCampaignClient struct{}

func (modernSmokeCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	return &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "c1", Name: "Campaign"}}}, nil
}

func (modernSmokeCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	return &statev1.GetCampaignResponse{Campaign: &statev1.Campaign{Id: "c1", Name: "Campaign"}}, nil
}

func (modernSmokeCampaignClient) CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "c1", Name: "Campaign"}}, nil
}

type modernSmokeParticipantClient struct{}

func (modernSmokeParticipantClient) ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	return &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{{
		Id:             "p1",
		CampaignId:     "c1",
		UserId:         "user-1",
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
	}}}, nil
}

type modernSmokeAccountClient struct{}

func (modernSmokeAccountClient) GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	return &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}, nil
}

func (modernSmokeAccountClient) UpdateProfile(context.Context, *authv1.UpdateProfileRequest, ...grpc.CallOption) (*authv1.UpdateProfileResponse, error) {
	return &authv1.UpdateProfileResponse{Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US}}, nil
}

type modernSmokeCredentialClient struct{}

func (modernSmokeCredentialClient) ListCredentials(context.Context, *aiv1.ListCredentialsRequest, ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error) {
	return &aiv1.ListCredentialsResponse{}, nil
}

func (modernSmokeCredentialClient) CreateCredential(context.Context, *aiv1.CreateCredentialRequest, ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error) {
	return &aiv1.CreateCredentialResponse{}, nil
}

func (modernSmokeCredentialClient) RevokeCredential(context.Context, *aiv1.RevokeCredentialRequest, ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error) {
	return &aiv1.RevokeCredentialResponse{}, nil
}
