package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type fakeParticipantClient struct {
	pages map[string]*statev1.ListParticipantsResponse
	err   error
	calls []*statev1.ListParticipantsRequest
}

func (f *fakeParticipantClient) ListParticipants(_ context.Context, req *statev1.ListParticipantsRequest, _ ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if req != nil {
		f.calls = append(f.calls, proto.Clone(req).(*statev1.ListParticipantsRequest))
	}
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

type fakeSessionClient struct {
	response *statev1.GetSessionResponse
	err      error
	calls    []*statev1.GetSessionRequest
}

func (f *fakeSessionClient) GetSession(_ context.Context, req *statev1.GetSessionRequest, _ ...grpc.CallOption) (*statev1.GetSessionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if req != nil {
		f.calls = append(f.calls, proto.Clone(req).(*statev1.GetSessionRequest))
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.GetSessionResponse{}, nil
}

func (*fakeSessionClient) StartSession(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeSessionClient) ListActiveSessionsForUser(context.Context, *statev1.ListActiveSessionsForUserRequest, ...grpc.CallOption) (*statev1.ListActiveSessionsForUserResponse, error) {
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
	if req != nil {
		f.calls = append(f.calls, proto.Clone(req).(*statev1.GetCampaignRequest))
	}
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

func (*fakeCampaignClient) UpdateCampaign(context.Context, *statev1.UpdateCampaignRequest, ...grpc.CallOption) (*statev1.UpdateCampaignResponse, error) {
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

func (*fakeCampaignClient) SetCampaignAIBinding(context.Context, *statev1.SetCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.SetCampaignAIBindingResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeCampaignClient) ClearCampaignAIBinding(context.Context, *statev1.ClearCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.ClearCampaignAIBindingResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeCampaignClient) GetCampaignAIBindingUsage(context.Context, *statev1.GetCampaignAIBindingUsageRequest, ...grpc.CallOption) (*statev1.GetCampaignAIBindingUsageResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeCampaignClient) GetCampaignSessionReadiness(context.Context, *statev1.GetCampaignSessionReadinessRequest, ...grpc.CallOption) (*statev1.GetCampaignSessionReadinessResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

type fakeWebSessionAuthClient struct {
	response *authv1.GetWebSessionResponse
	err      error
}

func (f *fakeWebSessionAuthClient) GetWebSession(_ context.Context, req *authv1.GetWebSessionRequest, _ ...grpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: req.GetSessionId(), UserId: "user-1"}}, nil
}

func TestCampaignAuthorizerResolveJoinWelcome(t *testing.T) {
	t.Parallel()

	authorizer := newCampaignAuthorizer(Config{},
		&fakeParticipantClient{pages: map[string]*statev1.ListParticipantsResponse{
			"": {Participants: []*statev1.Participant{{Id: "part-1", UserId: "user-1", Name: "Ari"}}},
		}},
		&fakeSessionClient{response: &statev1.GetSessionResponse{Session: &statev1.Session{Id: "sess-1", Name: "Session One"}}},
		&fakeCampaignClient{response: &statev1.GetCampaignResponse{Campaign: &statev1.Campaign{Id: "camp-1", Name: "Guildhouse"}}},
		nil,
	)

	welcome, err := authorizer.ResolveJoinWelcome(context.Background(), "camp-1", "sess-1", "user-1")
	if err != nil {
		t.Fatalf("ResolveJoinWelcome() error = %v", err)
	}
	if welcome.ParticipantID != "part-1" || welcome.ParticipantName != "Ari" {
		t.Fatalf("unexpected participant welcome: %+v", welcome)
	}
	if welcome.CampaignName != "Guildhouse" || welcome.SessionID != "sess-1" || welcome.SessionName != "Session One" {
		t.Fatalf("unexpected room welcome: %+v", welcome)
	}
}

func TestCampaignAuthorizerResolveJoinWelcomeRequiresParticipant(t *testing.T) {
	t.Parallel()

	authorizer := newCampaignAuthorizer(Config{},
		&fakeParticipantClient{pages: map[string]*statev1.ListParticipantsResponse{"": {}}},
		&fakeSessionClient{response: &statev1.GetSessionResponse{Session: &statev1.Session{Id: "sess-1"}}},
		nil,
		nil,
	)

	_, err := authorizer.ResolveJoinWelcome(context.Background(), "camp-1", "sess-1", "user-1")
	if !errors.Is(err, errCampaignParticipantRequired) {
		t.Fatalf("ResolveJoinWelcome() error = %v, want errCampaignParticipantRequired", err)
	}
}

func TestCampaignAuthorizerResolveJoinWelcomeFallsBackToIDs(t *testing.T) {
	t.Parallel()

	authorizer := newCampaignAuthorizer(Config{},
		&fakeParticipantClient{pages: map[string]*statev1.ListParticipantsResponse{
			"": {Participants: []*statev1.Participant{{Id: "part-1", UserId: "user-1"}}},
		}},
		&fakeSessionClient{response: &statev1.GetSessionResponse{Session: &statev1.Session{Id: "sess-1"}}},
		nil,
		nil,
	)

	welcome, err := authorizer.ResolveJoinWelcome(context.Background(), "camp-1", "sess-1", "user-1")
	if err != nil {
		t.Fatalf("ResolveJoinWelcome() error = %v", err)
	}
	if welcome.ParticipantName != "user-1" || welcome.CampaignName != "camp-1" || welcome.SessionName != "sess-1" {
		t.Fatalf("unexpected fallback welcome: %+v", welcome)
	}
}

func TestCampaignAuthorizerAuthenticateWebSession(t *testing.T) {
	t.Parallel()

	authorizer := newCampaignAuthorizer(Config{}, nil, nil, nil, &fakeWebSessionAuthClient{
		response: &authv1.GetWebSessionResponse{Session: &authv1.WebSession{Id: "sess-web", UserId: "user-2"}},
	})

	userID, err := authorizer.Authenticate(context.Background(), webSessionTokenPrefix+"sess-web")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if userID != "user-2" {
		t.Fatalf("Authenticate() userID = %q, want user-2", userID)
	}
}

func TestCampaignAuthorizerAuthenticateIntrospection(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("X-Resource-Secret"); got != "secret-1" {
			t.Fatalf("X-Resource-Secret = %q", got)
		}
		_, _ = w.Write([]byte(`{"active":true,"user_id":"user-9"}`))
	}))
	defer srv.Close()

	authorizer := newCampaignAuthorizer(Config{
		AuthBaseURL:         srv.URL,
		OAuthResourceSecret: "secret-1",
	}, nil, nil, nil, nil)

	userID, err := authorizer.Authenticate(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if strings.TrimSpace(userID) != "user-9" {
		t.Fatalf("Authenticate() userID = %q, want user-9", userID)
	}
}
