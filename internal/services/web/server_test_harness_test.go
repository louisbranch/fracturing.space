package web

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"google.golang.org/grpc"
)

func assertPrimaryNavLinks(t *testing.T, body string) {
	t.Helper()
	for _, href := range []string{"/app/dashboard", "/app/campaigns", "/app/notifications", "/app/settings"} {
		if !strings.Contains(body, "href=\""+href+"\"") {
			t.Fatalf("body missing nav href %q", href)
		}
	}
	if !strings.Contains(body, `action="/logout"`) {
		t.Fatalf("body missing logout form action %q", "/logout")
	}
}

func attachSessionCookie(t *testing.T, req *http.Request, auth *fakeWebAuthClient, userID string) {
	t.Helper()
	if req == nil {
		t.Fatalf("request is required")
	}
	if auth == nil {
		t.Fatalf("auth client is required")
	}
	if strings.TrimSpace(userID) == "" {
		t.Fatalf("user id is required")
	}
	resp, err := auth.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: userID})
	if err != nil {
		t.Fatalf("CreateWebSession() error = %v", err)
	}
	sessionID := strings.TrimSpace(resp.GetSession().GetId())
	if sessionID == "" {
		t.Fatalf("expected non-empty session id")
	}
	req.AddCookie(&http.Cookie{Name: "web_session", Value: sessionID})
}

func defaultProtectedConfig(auth *fakeWebAuthClient) Config {
	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{
		Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US},
	}}
	social := defaultSocialClient()
	return Config{
		EnableExperimentalModules: true,
		Dependencies: newDependencyBundle(
			PrincipalDependencies{
				SessionClient: auth,
				AccountClient: account,
				SocialClient:  social,
			},
			modules.Dependencies{
				AuthClient:               auth,
				CampaignClient:           defaultCampaignClient(),
				ParticipantClient:        defaultParticipantClient(),
				CharacterClient:          defaultCharacterClient(),
				DaggerheartContentClient: defaultDaggerheartContentClient(),
				SessionClient:            defaultSessionClient(),
				InviteClient:             defaultInviteClient(),
				AuthorizationClient:      defaultAuthorizationClient(),
				AccountClient:            account,
				ProfileSocialClient:      social,
				SettingsSocialClient:     social,
				CredentialClient:         fakeCredentialClient{},
			},
		),
	}
}

func newDependencyBundle(principal PrincipalDependencies, moduleDeps modules.Dependencies) *DependencyBundle {
	return &DependencyBundle{
		Principal: principal,
		Modules:   moduleDeps,
	}
}

func newDefaultDependencyBundle(moduleDeps modules.Dependencies) *DependencyBundle {
	return newDependencyBundle(PrincipalDependencies{}, moduleDeps)
}

func defaultStableProtectedConfig(auth *fakeWebAuthClient) Config {
	account := &fakeAccountClient{getProfileResp: &authv1.GetProfileResponse{
		Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_EN_US},
	}}
	social := defaultSocialClient()
	return Config{
		EnableExperimentalModules: false,
		Dependencies: newDependencyBundle(
			PrincipalDependencies{
				SessionClient: auth,
				AccountClient: account,
				SocialClient:  social,
			},
			modules.Dependencies{
				AuthClient:               auth,
				CampaignClient:           defaultCampaignClient(),
				ParticipantClient:        defaultParticipantClient(),
				CharacterClient:          defaultCharacterClient(),
				DaggerheartContentClient: defaultDaggerheartContentClient(),
				SessionClient:            defaultSessionClient(),
				InviteClient:             defaultInviteClient(),
				AuthorizationClient:      defaultAuthorizationClient(),
				AccountClient:            account,
				ProfileSocialClient:      social,
				SettingsSocialClient:     social,
				CredentialClient:         fakeCredentialClient{},
			},
		),
	}
}

func defaultSocialClient() *fakeSocialClient {
	return &fakeSocialClient{getUserProfileResp: &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Username: "adventurer", Name: "Adventurer"}}}
}

func defaultCampaignClient() fakeCampaignClient {
	return fakeCampaignClient{response: &statev1.ListCampaignsResponse{Campaigns: []*statev1.Campaign{{Id: "c1", Name: "Campaign"}}}}
}

func defaultParticipantClient() fakeWebParticipantClient {
	return fakeWebParticipantClient{response: &statev1.ListParticipantsResponse{Participants: []*statev1.Participant{{
		Id:             "p1",
		CampaignId:     "c1",
		UserId:         "user-1",
		Name:           "Owner",
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
	}}}}
}

func defaultCharacterClient() fakeWebCharacterClient {
	return fakeWebCharacterClient{response: &statev1.ListCharactersResponse{Characters: []*statev1.Character{{
		Id:   "char-1",
		Name: "Aria",
		Kind: statev1.CharacterKind_PC,
	}}}}
}

func defaultSessionClient() fakeWebSessionClient {
	return fakeWebSessionClient{response: &statev1.ListSessionsResponse{Sessions: []*statev1.Session{{
		Id:     "sess-1",
		Name:   "Session One",
		Status: statev1.SessionStatus_SESSION_ACTIVE,
	}}}}
}

func defaultInviteClient() fakeWebInviteClient {
	return fakeWebInviteClient{response: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{{
		Id:              "inv-1",
		CampaignId:      "c1",
		ParticipantId:   "p1",
		RecipientUserId: "user-2",
		Status:          statev1.InviteStatus_PENDING,
	}}}}
}

func defaultDaggerheartContentClient() fakeWebDaggerheartContentClient {
	return fakeWebDaggerheartContentClient{response: &daggerheartv1.GetDaggerheartContentCatalogResponse{Catalog: &daggerheartv1.DaggerheartContentCatalog{}}}
}

func defaultAuthorizationClient() fakeWebAuthorizationClient {
	return fakeWebAuthorizationClient{}
}

type fakeCampaignClient struct {
	response   *statev1.ListCampaignsResponse
	err        error
	getResp    *statev1.GetCampaignResponse
	getErr     error
	createResp *statev1.CreateCampaignResponse
	createErr  error
}

type fakeWebParticipantClient struct {
	response *statev1.ListParticipantsResponse
	err      error
}

func (f fakeWebParticipantClient) ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListParticipantsResponse{}, nil
}

type fakeWebCharacterClient struct {
	response *statev1.ListCharactersResponse
	err      error
}

func (f fakeWebCharacterClient) ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListCharactersResponse{}, nil
}

func (f fakeWebCharacterClient) CreateCharacter(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.CreateCharacterResponse{Character: &statev1.Character{Id: "char-created"}}, nil
}

func (f fakeWebCharacterClient) GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.GetCharacterSheetResponse{}, nil
}

func (f fakeWebCharacterClient) GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.GetCharacterCreationProgressResponse{}, nil
}

func (f fakeWebCharacterClient) ApplyCharacterCreationStep(context.Context, *statev1.ApplyCharacterCreationStepRequest, ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ApplyCharacterCreationStepResponse{}, nil
}

func (f fakeWebCharacterClient) ResetCharacterCreationWorkflow(context.Context, *statev1.ResetCharacterCreationWorkflowRequest, ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ResetCharacterCreationWorkflowResponse{}, nil
}

type fakeWebSessionClient struct {
	response *statev1.ListSessionsResponse
	err      error
}

func (f fakeWebSessionClient) ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListSessionsResponse{}, nil
}

func (f fakeWebSessionClient) StartSession(_ context.Context, req *statev1.StartSessionRequest, _ ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.StartSessionResponse{Session: &statev1.Session{
		Id:         "sess-started",
		CampaignId: strings.TrimSpace(req.GetCampaignId()),
		Name:       strings.TrimSpace(req.GetName()),
		Status:     statev1.SessionStatus_SESSION_ACTIVE,
	}}, nil
}

func (f fakeWebSessionClient) EndSession(_ context.Context, req *statev1.EndSessionRequest, _ ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.EndSessionResponse{Session: &statev1.Session{
		Id:         strings.TrimSpace(req.GetSessionId()),
		CampaignId: strings.TrimSpace(req.GetCampaignId()),
		Status:     statev1.SessionStatus_SESSION_ENDED,
	}}, nil
}

type fakeWebInviteClient struct {
	response *statev1.ListInvitesResponse
	err      error
}

func (f fakeWebInviteClient) ListInvites(context.Context, *statev1.ListInvitesRequest, ...grpc.CallOption) (*statev1.ListInvitesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListInvitesResponse{}, nil
}

func (f fakeWebInviteClient) CreateInvite(_ context.Context, req *statev1.CreateInviteRequest, _ ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.CreateInviteResponse{Invite: &statev1.Invite{
		Id:              "inv-created",
		CampaignId:      strings.TrimSpace(req.GetCampaignId()),
		ParticipantId:   strings.TrimSpace(req.GetParticipantId()),
		RecipientUserId: strings.TrimSpace(req.GetRecipientUserId()),
		Status:          statev1.InviteStatus_PENDING,
	}}, nil
}

func (f fakeWebInviteClient) RevokeInvite(_ context.Context, req *statev1.RevokeInviteRequest, _ ...grpc.CallOption) (*statev1.RevokeInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.RevokeInviteResponse{Invite: &statev1.Invite{
		Id:     strings.TrimSpace(req.GetInviteId()),
		Status: statev1.InviteStatus_REVOKED,
	}}, nil
}

type fakeWebDaggerheartContentClient struct {
	response *daggerheartv1.GetDaggerheartContentCatalogResponse
	err      error
}

func (f fakeWebDaggerheartContentClient) GetContentCatalog(context.Context, *daggerheartv1.GetDaggerheartContentCatalogRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartContentCatalogResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &daggerheartv1.GetDaggerheartContentCatalogResponse{Catalog: &daggerheartv1.DaggerheartContentCatalog{}}, nil
}

type fakeWebAuthorizationClient struct{}

func (fakeWebAuthorizationClient) Can(context.Context, *statev1.CanRequest, ...grpc.CallOption) (*statev1.CanResponse, error) {
	return &statev1.CanResponse{Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"}, nil
}

func (fakeWebAuthorizationClient) BatchCan(context.Context, *statev1.BatchCanRequest, ...grpc.CallOption) (*statev1.BatchCanResponse, error) {
	return &statev1.BatchCanResponse{}, nil
}

type fakeWebNotificationClient struct {
	listResp   *notificationsv1.ListNotificationsResponse
	listErr    error
	getResp    *notificationsv1.GetNotificationResponse
	getErr     error
	markResp   *notificationsv1.MarkNotificationReadResponse
	markErr    error
	unreadResp *notificationsv1.GetUnreadNotificationStatusResponse
	unreadErr  error
}

func (f fakeWebNotificationClient) ListNotifications(context.Context, *notificationsv1.ListNotificationsRequest, ...grpc.CallOption) (*notificationsv1.ListNotificationsResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &notificationsv1.ListNotificationsResponse{}, nil
}

func (f fakeWebNotificationClient) GetNotification(_ context.Context, req *notificationsv1.GetNotificationRequest, _ ...grpc.CallOption) (*notificationsv1.GetNotificationResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &notificationsv1.GetNotificationResponse{Notification: &notificationsv1.Notification{Id: req.GetNotificationId()}}, nil
}

func (f fakeWebNotificationClient) GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
	if f.unreadErr != nil {
		return nil, f.unreadErr
	}
	if f.unreadResp != nil {
		return f.unreadResp, nil
	}
	return &notificationsv1.GetUnreadNotificationStatusResponse{}, nil
}

func (f fakeWebNotificationClient) MarkNotificationRead(_ context.Context, req *notificationsv1.MarkNotificationReadRequest, _ ...grpc.CallOption) (*notificationsv1.MarkNotificationReadResponse, error) {
	if f.markErr != nil {
		return nil, f.markErr
	}
	if f.markResp != nil {
		return f.markResp, nil
	}
	return &notificationsv1.MarkNotificationReadResponse{Notification: &notificationsv1.Notification{Id: req.GetNotificationId()}}, nil
}

type fakeWebAuthClient struct {
	mu       sync.Mutex
	sessions map[string]string
}

type countingWebAuthClient struct {
	*fakeWebAuthClient
	countMu            sync.Mutex
	getWebSessionCalls int
}

func newCountingWebAuthClient() *countingWebAuthClient {
	return &countingWebAuthClient{fakeWebAuthClient: newFakeWebAuthClient()}
}

func (f *countingWebAuthClient) GetWebSession(ctx context.Context, req *authv1.GetWebSessionRequest, opts ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	f.countMu.Lock()
	f.getWebSessionCalls++
	f.countMu.Unlock()
	return f.fakeWebAuthClient.GetWebSession(ctx, req, opts...)
}

func (f *countingWebAuthClient) GetWebSessionCalls() int {
	f.countMu.Lock()
	defer f.countMu.Unlock()
	return f.getWebSessionCalls
}

func newFakeWebAuthClient() *fakeWebAuthClient {
	return &fakeWebAuthClient{sessions: map[string]string{}}
}

func (f *fakeWebAuthClient) CreateUser(context.Context, *authv1.CreateUserRequest, ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return &authv1.CreateUserResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *fakeWebAuthClient) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "register-session", CredentialCreationOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","rp":{"name":"web"},"user":{"id":"dXNlcg","name":"new@example.com","displayName":"new@example.com"},"pubKeyCredParams":[{"type":"public-key","alg":-7}]}}`)}, nil
}

func (f *fakeWebAuthClient) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return &authv1.FinishPasskeyRegistrationResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *fakeWebAuthClient) BeginPasskeyLogin(context.Context, *authv1.BeginPasskeyLoginRequest, ...grpc.CallOption) (*authv1.BeginPasskeyLoginResponse, error) {
	return &authv1.BeginPasskeyLoginResponse{SessionId: "login-session", CredentialRequestOptionsJson: []byte(`{"publicKey":{"challenge":"ZmFrZQ","timeout":60000,"userVerification":"preferred"}}`)}, nil
}

func (f *fakeWebAuthClient) FinishPasskeyLogin(context.Context, *authv1.FinishPasskeyLoginRequest, ...grpc.CallOption) (*authv1.FinishPasskeyLoginResponse, error) {
	return &authv1.FinishPasskeyLoginResponse{User: &authv1.User{Id: "user-1"}}, nil
}

func (f *fakeWebAuthClient) CreateWebSession(_ context.Context, req *authv1.CreateWebSessionRequest, _ ...grpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := "ws-1"
	f.sessions[id] = req.GetUserId()
	return &authv1.CreateWebSessionResponse{Session: &authv1.WebSession{Id: id, UserId: req.GetUserId()}, User: &authv1.User{Id: req.GetUserId()}}, nil
}

func (f *fakeWebAuthClient) GetWebSession(_ context.Context, req *authv1.GetWebSessionRequest, _ ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, ok := f.sessions[req.GetSessionId()]
	if !ok {
		return nil, context.Canceled
	}
	return &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: req.GetSessionId(), UserId: userID}, User: &authv1.User{Id: userID}}, nil
}

func (f *fakeWebAuthClient) RevokeWebSession(_ context.Context, req *authv1.RevokeWebSessionRequest, _ ...grpc.CallOption) (*authv1.RevokeWebSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, req.GetSessionId())
	return &authv1.RevokeWebSessionResponse{}, nil
}

func (f fakeCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

func (f fakeCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &statev1.GetCampaignResponse{Campaign: &statev1.Campaign{Id: "c1", Name: "Campaign"}}, nil
}

func (f fakeCampaignClient) CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "created"}}, nil
}
