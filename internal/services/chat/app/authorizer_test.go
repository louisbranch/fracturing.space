package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	a := &campaignAuthorizer{participantClient: client}
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
	a := &campaignAuthorizer{participantClient: &fakeParticipantClient{err: errors.New("boom")}}
	_, err := a.IsCampaignParticipant(context.Background(), "camp-1", "user-a")
	if err == nil {
		t.Fatal("expected list participants error")
	}
}
