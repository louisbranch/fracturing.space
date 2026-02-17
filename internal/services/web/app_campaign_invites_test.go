package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAppCampaignInvitesPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignInvitesPageParticipantRendersInvites(t *testing.T) {
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if inviteClient.listInvitesReq == nil {
		t.Fatalf("expected ListInvites request to be captured")
	}
	if inviteClient.listInvitesReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", inviteClient.listInvitesReq.GetCampaignId(), "camp-123")
	}
	participantIDs := inviteClient.listInvitesMD.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 1 || participantIDs[0] != "part-owner" {
		t.Fatalf("metadata %s = %v, want [part-owner]", grpcmeta.ParticipantIDHeader, participantIDs)
	}
	if len(participantClient.calls) != 1 {
		t.Fatalf("list participants calls = %d, want %d", len(participantClient.calls), 1)
	}
	body := w.Body.String()
	if !strings.Contains(body, "inv-1") {
		t.Fatalf("expected invite id in response body")
	}
}

func TestAppCampaignInviteCreateRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader("participant_id=part-1"))
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

func TestAppCampaignInviteRevokeRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/revoke", strings.NewReader("invite_id=inv-1"))
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

func TestAppCampaignInviteCreateParticipantCallsCreateInvite(t *testing.T) {
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
		createResp: &statev1.CreateInviteResponse{
			Invite: &statev1.Invite{Id: "inv-new", CampaignId: "camp-123", ParticipantId: "seat-1"},
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"participant_id":    {"seat-1"},
		"recipient_user_id": {"user-456"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/app/campaigns/camp-123/invites" {
		t.Fatalf("location = %q, want %q", location, "/app/campaigns/camp-123/invites")
	}
	if inviteClient.createReq == nil {
		t.Fatalf("expected CreateInvite request to be captured")
	}
	if inviteClient.createReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", inviteClient.createReq.GetCampaignId(), "camp-123")
	}
	if inviteClient.createReq.GetParticipantId() != "seat-1" {
		t.Fatalf("participant_id = %q, want %q", inviteClient.createReq.GetParticipantId(), "seat-1")
	}
	if inviteClient.createReq.GetRecipientUserId() != "user-456" {
		t.Fatalf("recipient_user_id = %q, want %q", inviteClient.createReq.GetRecipientUserId(), "user-456")
	}
	participantIDs := inviteClient.createMD.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 1 || participantIDs[0] != "part-owner" {
		t.Fatalf("metadata %s = %v, want [part-owner]", grpcmeta.ParticipantIDHeader, participantIDs)
	}
}

func TestAppCampaignInviteRevokeParticipantCallsRevokeInvite(t *testing.T) {
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
		revokeResp: &statev1.RevokeInviteResponse{
			Invite: &statev1.Invite{Id: "inv-1", CampaignId: "camp-123"},
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{"invite_id": {"inv-1"}}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/app/campaigns/camp-123/invites" {
		t.Fatalf("location = %q, want %q", location, "/app/campaigns/camp-123/invites")
	}
	if inviteClient.revokeReq == nil {
		t.Fatalf("expected RevokeInvite request to be captured")
	}
	if inviteClient.revokeReq.GetInviteId() != "inv-1" {
		t.Fatalf("invite_id = %q, want %q", inviteClient.revokeReq.GetInviteId(), "inv-1")
	}
	participantIDs := inviteClient.revokeMD.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 1 || participantIDs[0] != "part-owner" {
		t.Fatalf("metadata %s = %v, want [part-owner]", grpcmeta.ParticipantIDHeader, participantIDs)
	}
}

func TestRenderAppCampaignInvitesPageHidesWriteActionsWithoutManageAccess(t *testing.T) {
	w := httptest.NewRecorder()

	renderAppCampaignInvitesPage(w, "camp-123", []*statev1.Invite{
		{Id: "inv-1", CampaignId: "camp-123", RecipientUserId: "user-456"},
	}, false)

	body := w.Body.String()
	if strings.Contains(body, "/app/campaigns/camp-123/invites/create") {
		t.Fatalf("expected create invite form to be hidden")
	}
	if strings.Contains(body, "/app/campaigns/camp-123/invites/revoke") {
		t.Fatalf("expected revoke invite form to be hidden")
	}
}

func TestRenderAppCampaignInvitesPageSkipsRevokeForMissingInviteID(t *testing.T) {
	w := httptest.NewRecorder()

	renderAppCampaignInvitesPage(w, "camp-123", []*statev1.Invite{
		{CampaignId: "camp-123", RecipientUserId: "user-456"},
	}, true)

	body := w.Body.String()
	if !strings.Contains(body, "unknown-invite - user-456") {
		t.Fatalf("expected fallback invite id in response body")
	}
	if strings.Contains(body, "/app/campaigns/camp-123/invites/revoke") {
		t.Fatalf("expected revoke invite form to be hidden for missing invite id")
	}
}

func TestAppCampaignInvitesPageMemberCannotManageInvites(t *testing.T) {
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

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		listInvitesResp: &statev1.ListInvitesResponse{
			Invites: []*statev1.Invite{
				{Id: "inv-1", CampaignId: "camp-123", RecipientUserId: "user-999"},
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if inviteClient.listInvitesReq != nil {
		t.Fatalf("expected ListInvites not to be called for member")
	}
}

func TestAppCampaignInvitesPageIgnoresNilParticipantsInLookup(t *testing.T) {
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

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					nil,
					{
						Id:             "part-owner",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		listInvitesResp: &statev1.ListInvitesResponse{
			Invites: []*statev1.Invite{
				{Id: "inv-1", CampaignId: "camp-123", RecipientUserId: "user-999"},
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if inviteClient.listInvitesReq == nil {
		t.Fatalf("expected ListInvites request to be captured")
	}
}

func TestAppCampaignInviteCreateMemberCannotManageInvites(t *testing.T) {
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

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		createResp: &statev1.CreateInviteResponse{
			Invite: &statev1.Invite{Id: "inv-new", CampaignId: "camp-123", ParticipantId: "seat-1"},
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"participant_id":    {"seat-1"},
		"recipient_user_id": {"user-456"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if inviteClient.createReq != nil {
		t.Fatalf("expected CreateInvite not to be called for member")
	}
}

func TestAppCampaignInviteRevokeMemberCannotManageInvites(t *testing.T) {
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

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		revokeResp: &statev1.RevokeInviteResponse{
			Invite: &statev1.Invite{Id: "inv-1", CampaignId: "camp-123"},
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{"invite_id": {"inv-1"}}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if inviteClient.revokeReq != nil {
		t.Fatalf("expected RevokeInvite not to be called for member")
	}
}

func TestAppCampaignInvitesPageMapsPermissionDeniedToForbidden(t *testing.T) {
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

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-owner",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		listInvitesErr: status.Error(codes.PermissionDenied, "not allowed"),
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAppCampaignInvitesPageMapsInvalidArgumentToBadRequest(t *testing.T) {
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

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-owner",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		listInvitesErr: status.Error(codes.InvalidArgument, "bad request"),
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAppCampaignInviteCreateMapsPermissionDeniedToForbidden(t *testing.T) {
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

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-owner",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		createErr: status.Error(codes.PermissionDenied, "not allowed"),
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"participant_id":    {"seat-1"},
		"recipient_user_id": {"user-456"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestAppCampaignInviteCreateMapsUnavailableToServiceUnavailable(t *testing.T) {
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

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-owner",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		createErr: status.Error(codes.Unavailable, "temporarily unavailable"),
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
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"participant_id":    {"seat-1"},
		"recipient_user_id": {"user-456"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
