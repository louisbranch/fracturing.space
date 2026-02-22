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
	response         *statev1.ListPendingInvitesForUserResponse
	listPendingErr   error
	lastReq          *statev1.ListPendingInvitesForUserRequest
	listMetadata     metadata.MD
	listInvitesResp  *statev1.ListInvitesResponse
	listInvitesErr   error
	listInvitesReq   *statev1.ListInvitesRequest
	listInvitesMD    metadata.MD
	listInvitesCalls int
	createResp       *statev1.CreateInviteResponse
	createErr        error
	createReq        *statev1.CreateInviteRequest
	createMD         metadata.MD
	claimResp        *statev1.ClaimInviteResponse
	claimErr         error
	claimReq         *statev1.ClaimInviteRequest
	claimMD          metadata.MD
	revokeResp       *statev1.RevokeInviteResponse
	revokeErr        error
	revokeReq        *statev1.RevokeInviteRequest
	revokeMD         metadata.MD
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
	f.listInvitesCalls++
	if f.listInvitesErr != nil {
		return nil, f.listInvitesErr
	}
	if f.listInvitesResp != nil {
		return f.listInvitesResp, nil
	}
	return &statev1.ListInvitesResponse{}, nil
}

func TestAppCampaignInvitesPageCachesCampaignInvites(t *testing.T) {
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

	cacheStore := newFakeWebCacheStore()
	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-owner",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		listInvitesResp: &statev1.ListInvitesResponse{
			Invites: []*statev1.Invite{
				{Id: "inv-1", CampaignId: "camp-123", RecipientUserId: "user-456"},
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		participantClient: participantClient,
		inviteClient:      inviteClient,
		cacheStore:        cacheStore,
		campaignClient: &fakeWebCampaignClient{
			getResponse: &statev1.GetCampaignResponse{
				Campaign: &statev1.Campaign{
					Id:   "camp-123",
					Name: "Campaign One",
				},
			},
		},
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	req1 := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	req1.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w1 := httptest.NewRecorder()
	h.handleAppCampaignDetail(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", w1.Code, http.StatusOK)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	req2.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w2 := httptest.NewRecorder()
	h.handleAppCampaignDetail(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", w2.Code, http.StatusOK)
	}
	if !strings.Contains(w2.Body.String(), "Campaign One") {
		t.Fatalf("expected campaign name in cached response body")
	}

	if inviteClient.listInvitesCalls != 1 {
		t.Fatalf("list invites calls = %d, want %d", inviteClient.listInvitesCalls, 1)
	}
	if cacheStore.putCalls == 0 {
		t.Fatalf("expected cache store put calls")
	}
}

func TestCampaignInvitesCacheIsolatesByUser(t *testing.T) {
	cacheStore := newFakeWebCacheStore()
	h := &handler{cacheStore: cacheStore}

	ctx := context.Background()
	invites := []*statev1.Invite{
		{Id: "inv-1", CampaignId: "camp-1", RecipientUserId: "recipient-1"},
	}

	// Cache invites for user-A.
	h.setCampaignInvitesCache(ctx, "camp-1", "user-A", invites)

	// User-A should get a cache hit.
	got, ok := h.cachedCampaignInvites(ctx, "camp-1", "user-A")
	if !ok {
		t.Fatal("expected cache hit for user-A")
	}
	if len(got) != 1 || got[0].GetId() != "inv-1" {
		t.Fatalf("user-A cached invites = %v, want [inv-1]", got)
	}

	// User-B should get a cache miss — invites are policy-scoped.
	_, ok = h.cachedCampaignInvites(ctx, "camp-1", "user-B")
	if ok {
		t.Fatal("expected cache miss for user-B, got hit")
	}
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
					Participant: &statev1.Participant{Id: "part-1", Name: "Mira"},
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
