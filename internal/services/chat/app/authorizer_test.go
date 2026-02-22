package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeParticipantClient struct {
	pages map[string]*statev1.ListParticipantsResponse
	err   error
	calls []*statev1.ListParticipantsRequest
	users []string
}

type fakeSessionClient struct {
	pages map[string]*statev1.ListSessionsResponse
	err   error
	calls []*statev1.ListSessionsRequest
	users []string
}

func (f *fakeSessionClient) ListSessions(ctx context.Context, req *statev1.ListSessionsRequest, _ ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	f.users = append(f.users, userIDFromOutgoingContext(ctx))
	if f.err != nil {
		return nil, f.err
	}
	clone := *req
	f.calls = append(f.calls, &clone)
	if resp, ok := f.pages[req.GetPageToken()]; ok {
		return resp, nil
	}
	return &statev1.ListSessionsResponse{}, nil
}

func (*fakeSessionClient) StartSession(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) GetSession(context.Context, *statev1.GetSessionRequest, ...grpc.CallOption) (*statev1.GetSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) EndSession(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) OpenSessionGate(context.Context, *statev1.OpenSessionGateRequest, ...grpc.CallOption) (*statev1.OpenSessionGateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) ResolveSessionGate(context.Context, *statev1.ResolveSessionGateRequest, ...grpc.CallOption) (*statev1.ResolveSessionGateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) AbandonSessionGate(context.Context, *statev1.AbandonSessionGateRequest, ...grpc.CallOption) (*statev1.AbandonSessionGateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) GetSessionSpotlight(context.Context, *statev1.GetSessionSpotlightRequest, ...grpc.CallOption) (*statev1.GetSessionSpotlightResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) SetSessionSpotlight(context.Context, *statev1.SetSessionSpotlightRequest, ...grpc.CallOption) (*statev1.SetSessionSpotlightResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) ClearSessionSpotlight(context.Context, *statev1.ClearSessionSpotlightRequest, ...grpc.CallOption) (*statev1.ClearSessionSpotlightResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

type fakeCampaignClient struct {
	response *statev1.GetCampaignResponse
	err      error
	calls    []*statev1.GetCampaignRequest
}

func (f *fakeCampaignClient) GetCampaign(_ context.Context, req *statev1.GetCampaignRequest, _ ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	clone := *req
	f.calls = append(f.calls, &clone)
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.GetCampaignResponse{}, nil
}

func (*fakeCampaignClient) CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeCampaignClient) EndCampaign(context.Context, *statev1.EndCampaignRequest, ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeCampaignClient) ArchiveCampaign(context.Context, *statev1.ArchiveCampaignRequest, ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeCampaignClient) RestoreCampaign(context.Context, *statev1.RestoreCampaignRequest, ...grpc.CallOption) (*statev1.RestoreCampaignResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeCampaignClient) SetCampaignCover(context.Context, *statev1.SetCampaignCoverRequest, ...grpc.CallOption) (*statev1.SetCampaignCoverResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeParticipantClient) ListParticipants(ctx context.Context, req *statev1.ListParticipantsRequest, _ ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	f.users = append(f.users, userIDFromOutgoingContext(ctx))
	if f.err != nil {
		return nil, f.err
	}
	clone := *req
	f.calls = append(f.calls, &clone)
	if resp, ok := f.pages[req.GetPageToken()]; ok {
		return resp, nil
	}
	return &statev1.ListParticipantsResponse{}, nil
}

func (*fakeParticipantClient) CreateParticipant(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeParticipantClient) UpdateParticipant(context.Context, *statev1.UpdateParticipantRequest, ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeParticipantClient) DeleteParticipant(context.Context, *statev1.DeleteParticipantRequest, ...grpc.CallOption) (*statev1.DeleteParticipantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeParticipantClient) GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func userIDFromOutgoingContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ""
	}
	userIDs := md.Get(grpcmeta.UserIDHeader)
	if len(userIDs) == 0 {
		return ""
	}
	return strings.TrimSpace(userIDs[0])
}

func TestCampaignAuthorizerAuthenticateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer token-1")
		}
		if got := r.Header.Get("X-Resource-Secret"); got != "secret-1" {
			t.Fatalf("X-Resource-Secret = %q, want %q", got, "secret-1")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(authIntrospectResponse{Active: true, UserID: "user-1"})
	}))
	t.Cleanup(srv.Close)

	a := &campaignAuthorizer{
		authBaseURL:         srv.URL,
		oauthResourceSecret: "secret-1",
		httpClient:          srv.Client(),
	}

	userID, err := a.Authenticate(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("userID = %q, want %q", userID, "user-1")
	}
}

func TestCampaignAuthorizerAuthenticateInactiveToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(authIntrospectResponse{Active: false})
	}))
	t.Cleanup(srv.Close)

	a := &campaignAuthorizer{
		authBaseURL:         srv.URL,
		oauthResourceSecret: "secret-1",
		httpClient:          srv.Client(),
	}

	_, err := a.Authenticate(context.Background(), "token-1")
	if err == nil {
		t.Fatal("expected error for inactive token")
	}
}

func TestCampaignAuthorizerIsCampaignParticipantPaginates(t *testing.T) {
	client := &fakeParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants:  []*statev1.Participant{{Id: "p-1", UserId: "user-a"}},
				NextPageToken: "next-1",
			},
			"next-1": {
				Participants: []*statev1.Participant{{Id: "p-2", UserId: "user-b"}},
			},
		},
	}
	sessionClient := &fakeSessionClient{
		pages: map[string]*statev1.ListSessionsResponse{
			"": {
				Sessions: []*statev1.Session{
					{Id: "sess-1", CampaignId: "camp-1", Status: statev1.SessionStatus_SESSION_ACTIVE},
				},
			},
		},
	}

	a := &campaignAuthorizer{participantClient: client, sessionClient: sessionClient}
	allowed, err := a.IsCampaignParticipant(context.Background(), "camp-1", "user-b")
	if err != nil {
		t.Fatalf("is campaign participant: %v", err)
	}
	if !allowed {
		t.Fatal("expected participant access")
	}
	if len(client.calls) != 2 {
		t.Fatalf("list participants calls = %d, want 2", len(client.calls))
	}
	if got := client.calls[0].GetPageSize(); got != 100 {
		t.Fatalf("first page size = %d, want 100", got)
	}
}

func TestCampaignAuthorizerIsCampaignParticipantReturnsError(t *testing.T) {
	sessionClient := &fakeSessionClient{
		pages: map[string]*statev1.ListSessionsResponse{
			"": {
				Sessions: []*statev1.Session{
					{Id: "sess-1", CampaignId: "camp-1", Status: statev1.SessionStatus_SESSION_ACTIVE},
				},
			},
		},
	}
	a := &campaignAuthorizer{participantClient: &fakeParticipantClient{err: errors.New("boom")}, sessionClient: sessionClient}
	_, err := a.IsCampaignParticipant(context.Background(), "camp-1", "user-a")
	if err == nil {
		t.Fatal("expected list participants error")
	}
}

func TestCampaignAuthorizerIsCampaignParticipantAllowsWithoutActiveSession(t *testing.T) {
	a := &campaignAuthorizer{
		participantClient: &fakeParticipantClient{
			pages: map[string]*statev1.ListParticipantsResponse{
				"": {Participants: []*statev1.Participant{{Id: "p-1", UserId: "user-a"}}},
			},
		},
		sessionClient: &fakeSessionClient{pages: map[string]*statev1.ListSessionsResponse{"": {Sessions: []*statev1.Session{}}}},
	}
	allowed, err := a.IsCampaignParticipant(context.Background(), "camp-1", "user-a")
	if err != nil {
		t.Fatalf("is campaign participant: %v", err)
	}
	if !allowed {
		t.Fatal("expected participant access without active session")
	}
}

func TestCampaignAuthorizerResolveJoinWelcomeWithoutActiveSession(t *testing.T) {
	a := &campaignAuthorizer{
		participantClient: &fakeParticipantClient{
			pages: map[string]*statev1.ListParticipantsResponse{
				"": {Participants: []*statev1.Participant{{Id: "p-1", UserId: "user-a", Name: "Ari"}}},
			},
		},
		sessionClient: &fakeSessionClient{
			pages: map[string]*statev1.ListSessionsResponse{
				"": {Sessions: []*statev1.Session{}},
			},
		},
		campaignClient: &fakeCampaignClient{
			response: &statev1.GetCampaignResponse{
				Campaign: &statev1.Campaign{Id: "camp-1", Name: "Campanha Um", Locale: commonv1.Locale_LOCALE_PT_BR},
			},
		},
	}

	welcome, err := a.ResolveJoinWelcome(context.Background(), "camp-1", "user-a")
	if err != nil {
		t.Fatalf("resolve join welcome: %v", err)
	}
	if welcome.ParticipantName != "Ari" {
		t.Fatalf("participant = %q, want %q", welcome.ParticipantName, "Ari")
	}
	if welcome.CampaignName != "Campanha Um" {
		t.Fatalf("campaign = %q, want %q", welcome.CampaignName, "Campanha Um")
	}
	if welcome.SessionID != "" {
		t.Fatalf("session id = %q, want empty", welcome.SessionID)
	}
	if welcome.SessionName != "" {
		t.Fatalf("session name = %q, want empty", welcome.SessionName)
	}
}

func TestCampaignAuthorizerResolveJoinWelcomeRequiresParticipant(t *testing.T) {
	a := &campaignAuthorizer{
		participantClient: &fakeParticipantClient{
			pages: map[string]*statev1.ListParticipantsResponse{
				"": {Participants: []*statev1.Participant{}},
			},
		},
		sessionClient: &fakeSessionClient{
			pages: map[string]*statev1.ListSessionsResponse{
				"": {Sessions: []*statev1.Session{}},
			},
		},
	}

	_, err := a.ResolveJoinWelcome(context.Background(), "camp-1", "user-a")
	if !errors.Is(err, errCampaignParticipantRequired) {
		t.Fatalf("error = %v, want errCampaignParticipantRequired", err)
	}
}

func TestCampaignAuthorizerResolveJoinWelcomeUsesCampaignLocale(t *testing.T) {
	a := &campaignAuthorizer{
		participantClient: &fakeParticipantClient{
			pages: map[string]*statev1.ListParticipantsResponse{
				"": {Participants: []*statev1.Participant{{Id: "p-1", UserId: "user-a", Name: "Ari"}}},
			},
		},
		sessionClient: &fakeSessionClient{
			pages: map[string]*statev1.ListSessionsResponse{
				"": {Sessions: []*statev1.Session{{Id: "sess-1", Name: "Sessao Um", CampaignId: "camp-1", Status: statev1.SessionStatus_SESSION_ACTIVE}}},
			},
		},
		campaignClient: &fakeCampaignClient{
			response: &statev1.GetCampaignResponse{
				Campaign: &statev1.Campaign{Id: "camp-1", Name: "Campanha Um", Locale: commonv1.Locale_LOCALE_PT_BR},
			},
		},
	}
	welcome, err := a.ResolveJoinWelcome(context.Background(), "camp-1", "user-a")
	if err != nil {
		t.Fatalf("resolve join welcome: %v", err)
	}
	if welcome.Locale != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("locale = %v, want %v", welcome.Locale, commonv1.Locale_LOCALE_PT_BR)
	}
	if welcome.ParticipantName != "Ari" {
		t.Fatalf("participant = %q, want %q", welcome.ParticipantName, "Ari")
	}
	if welcome.CampaignName != "Campanha Um" {
		t.Fatalf("campaign = %q, want %q", welcome.CampaignName, "Campanha Um")
	}
	if welcome.SessionID != "sess-1" {
		t.Fatalf("session id = %q, want %q", welcome.SessionID, "sess-1")
	}
	if welcome.SessionName != "Sessao Um" {
		t.Fatalf("session name = %q, want %q", welcome.SessionName, "Sessao Um")
	}
}

func TestCampaignAuthorizerResolveJoinWelcomePropagatesUserIdentityToGameRPC(t *testing.T) {
	sessionClient := &fakeSessionClient{
		pages: map[string]*statev1.ListSessionsResponse{
			"": {
				Sessions: []*statev1.Session{{Id: "sess-1", Name: "Sessao Um", CampaignId: "camp-1", Status: statev1.SessionStatus_SESSION_ACTIVE}},
			},
		},
	}
	participantClient := &fakeParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{{Id: "p-1", UserId: "user-a", Name: "Ari"}},
			},
		},
	}
	a := &campaignAuthorizer{
		sessionClient:     sessionClient,
		participantClient: participantClient,
		campaignClient: &fakeCampaignClient{
			response: &statev1.GetCampaignResponse{
				Campaign: &statev1.Campaign{Id: "camp-1", Name: "Campanha Um", Locale: commonv1.Locale_LOCALE_PT_BR},
			},
		},
	}

	_, err := a.ResolveJoinWelcome(context.Background(), "camp-1", "user-a")
	if err != nil {
		t.Fatalf("resolve join welcome: %v", err)
	}
	if len(sessionClient.users) != 1 {
		t.Fatalf("session calls = %d, want 1", len(sessionClient.users))
	}
	if got := sessionClient.users[0]; got != "user-a" {
		t.Fatalf("session call user = %q, want %q", got, "user-a")
	}
	if len(participantClient.users) != 1 {
		t.Fatalf("participant calls = %d, want 1", len(participantClient.users))
	}
	if got := participantClient.users[0]; got != "user-a" {
		t.Fatalf("participant call user = %q, want %q", got, "user-a")
	}
}
