package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeWebParticipantClient struct {
	pages        map[string]*statev1.ListParticipantsResponse
	err          error
	calls        []*statev1.ListParticipantsRequest
	listMD       metadata.MD
	listMDByCall []metadata.MD
	updateReq    *statev1.UpdateParticipantRequest
	updateMD     metadata.MD
	updateResp   *statev1.UpdateParticipantResponse
	updateErr    error
}

func (f *fakeWebParticipantClient) ListParticipants(ctx context.Context, req *statev1.ListParticipantsRequest, _ ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.err != nil {
		return nil, f.err
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	f.listMD = md
	if len(f.listMDByCall) < len(f.calls)+1 {
		f.listMDByCall = append(f.listMDByCall, md)
	} else {
		f.listMDByCall[len(f.calls)] = md
	}
	cloned := *req
	f.calls = append(f.calls, &cloned)
	if resp, ok := f.pages[req.GetPageToken()]; ok {
		return resp, nil
	}
	return &statev1.ListParticipantsResponse{}, nil
}

func (*fakeWebParticipantClient) CreateParticipant(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebParticipantClient) UpdateParticipant(ctx context.Context, req *statev1.UpdateParticipantRequest, _ ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.updateMD = md
	f.updateReq = req
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.updateResp != nil {
		return f.updateResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeWebParticipantClient) DeleteParticipant(context.Context, *statev1.DeleteParticipantRequest, ...grpc.CallOption) (*statev1.DeleteParticipantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (*fakeWebParticipantClient) GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestNewCampaignAccessCheckerRequiresConfigAndClient(t *testing.T) {
	cfg := Config{AuthBaseURL: "http://auth.test", OAuthResourceSecret: "secret-1"}
	if got := newCampaignAccessChecker(cfg, nil); got != nil {
		t.Fatal("expected nil checker without participant client")
	}

	if got := newCampaignAccessChecker(Config{OAuthResourceSecret: "secret-1"}, &fakeWebParticipantClient{}); got != nil {
		t.Fatal("expected nil checker without auth base url")
	}

	if got := newCampaignAccessChecker(Config{AuthBaseURL: "http://auth.test"}, &fakeWebParticipantClient{}); got != nil {
		t.Fatal("expected nil checker without oauth resource secret")
	}

	if got := newCampaignAccessChecker(cfg, &fakeWebParticipantClient{}); got == nil {
		t.Fatal("expected checker when config and participant client are present")
	}
}

func TestCampaignAccessServiceResolveUserIDSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %s, want /introspect", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer token-1")
		}
		if got := r.Header.Get("X-Resource-Secret"); got != "secret-1" {
			t.Fatalf("X-Resource-Secret = %q, want %q", got, "secret-1")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: " user-1 "})
	}))
	t.Cleanup(srv.Close)

	svc := &campaignAccessService{
		authBaseURL:         srv.URL,
		oauthResourceSecret: "secret-1",
		httpClient:          srv.Client(),
	}

	userID, err := svc.ResolveUserID(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("introspect user id: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("userID = %q, want %q", userID, "user-1")
	}
}

func TestCampaignAccessServiceResolveUserIDInactive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: false, UserID: "user-1"})
	}))
	t.Cleanup(srv.Close)

	svc := &campaignAccessService{
		authBaseURL:         srv.URL,
		oauthResourceSecret: "secret-1",
		httpClient:          srv.Client(),
	}

	userID, err := svc.ResolveUserID(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("introspect user id: %v", err)
	}
	if userID != "" {
		t.Fatalf("userID = %q, want empty", userID)
	}
}

func TestCampaignAccessServiceIsCampaignParticipantPaginatesUntilMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "user-b"})
	}))
	t.Cleanup(srv.Close)

	client := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants:  []*statev1.Participant{{Id: "p-1", UserId: "user-a"}},
				NextPageToken: "page-2",
			},
			"page-2": {
				Participants: []*statev1.Participant{{Id: "p-2", UserId: "user-b"}},
			},
		},
	}

	svc := &campaignAccessService{
		authBaseURL:         srv.URL,
		oauthResourceSecret: "secret-1",
		httpClient:          srv.Client(),
		participantClient:   client,
	}

	allowed, err := svc.IsCampaignParticipant(context.Background(), "camp-1", "token-1")
	if err != nil {
		t.Fatalf("is campaign participant: %v", err)
	}
	if !allowed {
		t.Fatal("expected allowed participant")
	}
	if len(client.calls) != 2 {
		t.Fatalf("list participants calls = %d, want 2", len(client.calls))
	}
	if got := client.calls[0].GetPageSize(); got != 10 {
		t.Fatalf("page size = %d, want 10", got)
	}
}

func TestCampaignAccessServiceIsCampaignParticipantFalseAndErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "user-z"})
	}))
	t.Cleanup(srv.Close)

	noMatch := &campaignAccessService{
		authBaseURL:         srv.URL,
		oauthResourceSecret: "secret-1",
		httpClient:          srv.Client(),
		participantClient: &fakeWebParticipantClient{pages: map[string]*statev1.ListParticipantsResponse{
			"": {Participants: []*statev1.Participant{{Id: "p-1", UserId: "user-a"}}},
		}},
	}
	allowed, err := noMatch.IsCampaignParticipant(context.Background(), "camp-1", "token-1")
	if err != nil {
		t.Fatalf("is campaign participant: %v", err)
	}
	if allowed {
		t.Fatal("expected not allowed when participant is absent")
	}

	failing := &campaignAccessService{
		authBaseURL:         srv.URL,
		oauthResourceSecret: "secret-1",
		httpClient:          srv.Client(),
		participantClient:   &fakeWebParticipantClient{err: errors.New("boom")},
	}
	_, err = failing.IsCampaignParticipant(context.Background(), "camp-1", "token-1")
	if err == nil || !strings.Contains(err.Error(), "list campaign participants") {
		t.Fatalf("error = %v, expected list campaign participants error", err)
	}
}
