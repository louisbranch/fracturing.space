package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAppInvitesPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/app/invites", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

type fakeWebInviteClient struct {
	response        *statev1.ListPendingInvitesForUserResponse
	listPendingErr  error
	lastReq         *statev1.ListPendingInvitesForUserRequest
	listMetadata    metadata.MD
	listInvitesResp *statev1.ListInvitesResponse
	listInvitesErr  error
	listInvitesReq  *statev1.ListInvitesRequest
	listInvitesMD   metadata.MD
	createResp      *statev1.CreateInviteResponse
	createErr       error
	createReq       *statev1.CreateInviteRequest
	createMD        metadata.MD
	claimResp       *statev1.ClaimInviteResponse
	claimErr        error
	claimReq        *statev1.ClaimInviteRequest
	claimMD         metadata.MD
	revokeResp      *statev1.RevokeInviteResponse
	revokeErr       error
	revokeReq       *statev1.RevokeInviteRequest
	revokeMD        metadata.MD
}

func (f *fakeWebInviteClient) CreateInvite(ctx context.Context, req *statev1.CreateInviteRequest, _ ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.createMD = md
	f.createReq = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebInviteClient) ClaimInvite(ctx context.Context, req *statev1.ClaimInviteRequest, _ ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.claimMD = md
	f.claimReq = req
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	if f.claimResp != nil {
		return f.claimResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebInviteClient) GetInvite(context.Context, *statev1.GetInviteRequest, ...grpc.CallOption) (*statev1.GetInviteResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebInviteClient) ListInvites(ctx context.Context, req *statev1.ListInvitesRequest, _ ...grpc.CallOption) (*statev1.ListInvitesResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.listInvitesMD = md
	f.listInvitesReq = req
	if f.listInvitesErr != nil {
		return nil, f.listInvitesErr
	}
	if f.listInvitesResp != nil {
		return f.listInvitesResp, nil
	}
	return &statev1.ListInvitesResponse{}, nil
}

func (f *fakeWebInviteClient) ListPendingInvites(context.Context, *statev1.ListPendingInvitesRequest, ...grpc.CallOption) (*statev1.ListPendingInvitesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebInviteClient) ListPendingInvitesForUser(ctx context.Context, req *statev1.ListPendingInvitesForUserRequest, _ ...grpc.CallOption) (*statev1.ListPendingInvitesForUserResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.listMetadata = md
	f.lastReq = req
	if f.listPendingErr != nil {
		return nil, f.listPendingErr
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListPendingInvitesForUserResponse{}, nil
}

func (f *fakeWebInviteClient) RevokeInvite(ctx context.Context, req *statev1.RevokeInviteRequest, _ ...grpc.CallOption) (*statev1.RevokeInviteResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.revokeMD = md
	f.revokeReq = req
	if f.revokeErr != nil {
		return nil, f.revokeErr
	}
	if f.revokeResp != nil {
		return f.revokeResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestAppInvitesPageRendersPendingInvitesForUser(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	inviteClient := &fakeWebInviteClient{
		response: &statev1.ListPendingInvitesForUserResponse{
			Invites: []*statev1.PendingUserInvite{
				{
					Invite:      &statev1.Invite{Id: "inv-1", CampaignId: "camp-1"},
					Campaign:    &statev1.Campaign{Id: "camp-1", Name: "Winds of Iron"},
					Participant: &statev1.Participant{Id: "part-1", DisplayName: "Mira"},
				},
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
		inviteClient: inviteClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/invites", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppInvites(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if inviteClient.lastReq == nil {
		t.Fatalf("expected ListPendingInvitesForUser request to be captured")
	}
	if inviteClient.lastReq.GetPageSize() != 10 {
		t.Fatalf("page_size = %d, want %d", inviteClient.lastReq.GetPageSize(), 10)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Winds of Iron") {
		t.Fatalf("expected campaign name in response")
	}
	if !strings.Contains(body, "Mira") {
		t.Fatalf("expected participant display name in response")
	}
	userIDs := inviteClient.listMetadata.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-123" {
		t.Fatalf("metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, userIDs)
	}
}

func TestAppInviteClaimRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/app/invites/claim", strings.NewReader("campaign_id=camp-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

type fakeJoinGrantAuthClient struct {
	*fakeAuthClient
	issueJoinGrantResp *authv1.IssueJoinGrantResponse
	issueJoinGrantReq  *authv1.IssueJoinGrantRequest
}

func (f *fakeJoinGrantAuthClient) IssueJoinGrant(_ context.Context, req *authv1.IssueJoinGrantRequest, _ ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	f.issueJoinGrantReq = req
	if f.issueJoinGrantResp != nil {
		return f.issueJoinGrantResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestAppInviteClaimRedirectsToCampaignAfterClaim(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	authClient := &fakeJoinGrantAuthClient{
		fakeAuthClient: &fakeAuthClient{},
		issueJoinGrantResp: &authv1.IssueJoinGrantResponse{
			JoinGrant: "join-grant-1",
		},
	}
	inviteClient := &fakeWebInviteClient{
		claimResp: &statev1.ClaimInviteResponse{
			Invite: &statev1.Invite{Id: "inv-1", CampaignId: "camp-1"},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		authClient:   authClient,
		sessions:     newSessionStore(),
		pendingFlows: newPendingFlowStore(),
		inviteClient: inviteClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"campaign_id":    {"camp-1"},
		"invite_id":      {"inv-1"},
		"participant_id": {"part-1"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/invites/claim", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppInviteClaim(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/app/campaigns/camp-1" {
		t.Fatalf("location = %q, want %q", location, "/app/campaigns/camp-1")
	}
	if authClient.issueJoinGrantReq == nil {
		t.Fatalf("expected IssueJoinGrant request to be captured")
	}
	if authClient.issueJoinGrantReq.GetUserId() != "user-123" {
		t.Fatalf("user_id = %q, want %q", authClient.issueJoinGrantReq.GetUserId(), "user-123")
	}
	if authClient.issueJoinGrantReq.GetCampaignId() != "camp-1" {
		t.Fatalf("campaign_id = %q, want %q", authClient.issueJoinGrantReq.GetCampaignId(), "camp-1")
	}
	if authClient.issueJoinGrantReq.GetInviteId() != "inv-1" {
		t.Fatalf("invite_id = %q, want %q", authClient.issueJoinGrantReq.GetInviteId(), "inv-1")
	}
	if authClient.issueJoinGrantReq.GetParticipantId() != "part-1" {
		t.Fatalf("participant_id = %q, want %q", authClient.issueJoinGrantReq.GetParticipantId(), "part-1")
	}
	if inviteClient.claimReq == nil {
		t.Fatalf("expected ClaimInvite request to be captured")
	}
	if inviteClient.claimReq.GetCampaignId() != "camp-1" {
		t.Fatalf("campaign_id = %q, want %q", inviteClient.claimReq.GetCampaignId(), "camp-1")
	}
	if inviteClient.claimReq.GetInviteId() != "inv-1" {
		t.Fatalf("invite_id = %q, want %q", inviteClient.claimReq.GetInviteId(), "inv-1")
	}
	if inviteClient.claimReq.GetJoinGrant() != "join-grant-1" {
		t.Fatalf("join_grant = %q, want %q", inviteClient.claimReq.GetJoinGrant(), "join-grant-1")
	}
	userIDs := inviteClient.claimMD.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-123" {
		t.Fatalf("metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, userIDs)
	}
}
