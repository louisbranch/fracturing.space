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

	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"golang.org/x/text/language"
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
	userIDs := inviteClient.listInvitesMD.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-123" {
		t.Fatalf("metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, userIDs)
	}
	if len(participantClient.calls) != 0 {
		t.Fatalf("list participants calls = %d, want %d", len(participantClient.calls), 0)
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

func TestAppCampaignInviteCreateParticipantResolvesRecipientUsername(t *testing.T) {
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
	connectionsClient := &fakeConnectionsClient{
		lookupUsernameResp: &connectionsv1.LookupUsernameResponse{
			UsernameRecord: &connectionsv1.UsernameRecord{
				UserId:   "user-456",
				Username: "alice",
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
		connectionsClient: connectionsClient,
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
		"recipient_user_id": {"@alice"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if inviteClient.createReq == nil {
		t.Fatalf("expected CreateInvite request to be captured")
	}
	if got := inviteClient.createReq.GetRecipientUserId(); got != "user-456" {
		t.Fatalf("recipient_user_id = %q, want user-456", got)
	}
	if connectionsClient.lookupUsernameReq == nil {
		t.Fatal("expected LookupUsername request")
	}
	if got := connectionsClient.lookupUsernameReq.GetUsername(); got != "alice" {
		t.Fatalf("lookup username = %q, want alice", got)
	}
	lookupUserIDs := connectionsClient.lookupUsernameMD.Get(grpcmeta.UserIDHeader)
	if len(lookupUserIDs) != 1 || lookupUserIDs[0] != "user-123" {
		t.Fatalf("lookup metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, lookupUserIDs)
	}
}

func TestAppCampaignInviteCreateParticipantVerifyUsernameRendersVerificationContext(t *testing.T) {
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
		listInvitesResp: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{}},
		createResp: &statev1.CreateInviteResponse{
			Invite: &statev1.Invite{Id: "inv-new", CampaignId: "camp-123", ParticipantId: "seat-1"},
		},
	}
	connectionsClient := &fakeConnectionsClient{
		lookupUsernameResp: &connectionsv1.LookupUsernameResponse{
			UsernameRecord: &connectionsv1.UsernameRecord{
				UserId:   "user-456",
				Username: "alice",
			},
		},
		lookupPublicProfileResp: &connectionsv1.LookupPublicProfileResponse{
			UsernameRecord: &connectionsv1.UsernameRecord{
				UserId:   "user-456",
				Username: "alice",
			},
			PublicProfileRecord: &connectionsv1.PublicProfileRecord{
				UserId:        "user-456",
				Name:          "Alice Verified",
				AvatarSetId:   "avatar_set_v1",
				AvatarAssetId: "001",
				Bio:           "GM",
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
		connectionsClient: connectionsClient,
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
		"recipient_user_id": {"@alice"},
		"action":            {"verify"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if inviteClient.createReq != nil {
		t.Fatal("expected CreateInvite to be skipped during verification")
	}
	if connectionsClient.lookupPublicProfileReq == nil {
		t.Fatal("expected LookupPublicProfile request")
	}
	if got := connectionsClient.lookupPublicProfileReq.GetUsername(); got != "alice" {
		t.Fatalf("lookup public profile username = %q, want alice", got)
	}
	lookupUserIDs := connectionsClient.lookupPublicProfileMD.Get(grpcmeta.UserIDHeader)
	if len(lookupUserIDs) != 1 || lookupUserIDs[0] != "user-123" {
		t.Fatalf("lookup public profile metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, lookupUserIDs)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Alice Verified") {
		t.Fatalf("response body missing verification display name: %q", body)
	}
	if !strings.Contains(body, "user-456") {
		t.Fatalf("response body missing verification user id: %q", body)
	}
}

func TestAppCampaignInviteCreateParticipantVerifyUsernameRequiresAtPrefix(t *testing.T) {
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
		listInvitesResp: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{}},
		createResp: &statev1.CreateInviteResponse{
			Invite: &statev1.Invite{Id: "inv-new", CampaignId: "camp-123", ParticipantId: "seat-1"},
		},
	}
	connectionsClient := &fakeConnectionsClient{
		lookupPublicProfileResp: &connectionsv1.LookupPublicProfileResponse{
			UsernameRecord: &connectionsv1.UsernameRecord{
				UserId:   "user-456",
				Username: "alice",
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
		connectionsClient: connectionsClient,
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
		"recipient_user_id": {"alice"},
		"action":            {"verify"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if inviteClient.createReq != nil {
		t.Fatal("expected CreateInvite to be skipped during verification failure")
	}
	if connectionsClient.lookupPublicProfileReq != nil {
		t.Fatal("expected LookupPublicProfile to be skipped when @ prefix is missing")
	}
	if !strings.Contains(w.Body.String(), "recipient username must start with @") {
		t.Fatalf("response body missing @ prefix message: %q", w.Body.String())
	}
}

func TestAppCampaignInviteCreateParticipantVerifyUsernameRendersWhenProfileMissing(t *testing.T) {
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
		listInvitesResp: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{}},
		createResp: &statev1.CreateInviteResponse{
			Invite: &statev1.Invite{Id: "inv-new", CampaignId: "camp-123", ParticipantId: "seat-1"},
		},
	}
	connectionsClient := &fakeConnectionsClient{
		lookupPublicProfileResp: &connectionsv1.LookupPublicProfileResponse{
			UsernameRecord: &connectionsv1.UsernameRecord{
				UserId:   "user-456",
				Username: "alice",
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
		connectionsClient: connectionsClient,
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
		"recipient_user_id": {"@alice"},
		"action":            {"verify"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Username: @alice") {
		t.Fatalf("response body missing verification username: %q", body)
	}
	if !strings.Contains(body, "User ID: user-456") {
		t.Fatalf("response body missing verification user id: %q", body)
	}
}

func TestAppCampaignInviteCreateParticipantVerifyUsernameLocalizesVerificationCopy(t *testing.T) {
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
		listInvitesResp: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{}},
		createResp: &statev1.CreateInviteResponse{
			Invite: &statev1.Invite{Id: "inv-new", CampaignId: "camp-123", ParticipantId: "seat-1"},
		},
	}
	connectionsClient := &fakeConnectionsClient{
		lookupPublicProfileResp: &connectionsv1.LookupPublicProfileResponse{
			UsernameRecord: &connectionsv1.UsernameRecord{
				UserId:   "user-456",
				Username: "alice",
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
		connectionsClient: connectionsClient,
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
		"recipient_user_id": {"@alice"},
		"action":            {"verify"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create?lang=pt-BR", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Destinatário verificado") {
		t.Fatalf("expected localized verification heading in response, got %q", body)
	}
}

func TestAppCampaignInviteCreateParticipantRecipientUsernameLookupNotFound(t *testing.T) {
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
	connectionsClient := &fakeConnectionsClient{
		lookupUsernameErr: status.Error(codes.NotFound, "username not found"),
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
		connectionsClient: connectionsClient,
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
		"recipient_user_id": {"@missing"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if inviteClient.createReq != nil {
		t.Fatal("expected CreateInvite to be skipped")
	}
}

func TestAppCampaignInviteCreateParticipantRecipientUsernameInvalid(t *testing.T) {
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
	connectionsClient := &fakeConnectionsClient{
		lookupUsernameErr: status.Error(codes.InvalidArgument, "invalid username"),
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
		connectionsClient: connectionsClient,
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
		"recipient_user_id": {"@bad username"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if inviteClient.createReq != nil {
		t.Fatal("expected CreateInvite to be skipped")
	}
	if !strings.Contains(w.Body.String(), "recipient username is invalid") {
		t.Fatalf("response body missing invalid username message: %q", w.Body.String())
	}
}

func TestAppCampaignInviteCreateParticipantRecipientUsernameRequired(t *testing.T) {
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
	connectionsClient := &fakeConnectionsClient{
		lookupUsernameResp: &connectionsv1.LookupUsernameResponse{
			UsernameRecord: &connectionsv1.UsernameRecord{
				UserId:   "user-456",
				Username: "alice",
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
		connectionsClient: connectionsClient,
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
		"recipient_user_id": {"@"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if inviteClient.createReq != nil {
		t.Fatal("expected CreateInvite to be skipped")
	}
	if connectionsClient.lookupUsernameReq != nil {
		t.Fatal("expected LookupUsername to be skipped for empty username")
	}
}

func TestAppCampaignInviteCreateParticipantRecipientUsernameRequiresConnectionsClient(t *testing.T) {
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
		"recipient_user_id": {"@alice"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
	if inviteClient.createReq != nil {
		t.Fatal("expected CreateInvite to be skipped")
	}
}

func TestAppCampaignInviteCreateParticipantRecipientUsernameLookupEmptyRecord(t *testing.T) {
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
	connectionsClient := &fakeConnectionsClient{
		lookupUsernameResp: &connectionsv1.LookupUsernameResponse{},
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
		connectionsClient: connectionsClient,
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
		"recipient_user_id": {"@alice"},
	}
	req := httptest.NewRequest(http.MethodPost, "/app/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if inviteClient.createReq != nil {
		t.Fatal("expected CreateInvite to be skipped")
	}
}

func TestAppCampaignInviteCreateParticipantUsesRecipientUserIDFallback(t *testing.T) {
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
	connectionsClient := &fakeConnectionsClient{
		lookupUsernameResp: &connectionsv1.LookupUsernameResponse{
			UsernameRecord: &connectionsv1.UsernameRecord{
				UserId:   "user-lookup",
				Username: "user-456",
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
		connectionsClient: connectionsClient,
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
	if inviteClient.createReq == nil {
		t.Fatal("expected CreateInvite request")
	}
	if got := inviteClient.createReq.GetRecipientUserId(); got != "user-456" {
		t.Fatalf("recipient_user_id = %q, want user-456", got)
	}
	if connectionsClient.lookupUsernameReq != nil {
		t.Fatal("expected LookupUsername to be skipped for explicit user id")
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

func TestAppCampaignInvitesPageRendersContactOptions(t *testing.T) {
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
		listInvitesResp: &statev1.ListInvitesResponse{
			Invites: []*statev1.Invite{},
		},
	}
	connectionsClient := &fakeConnectionsClient{
		listContactsResp: &connectionsv1.ListContactsResponse{
			Contacts: []*connectionsv1.Contact{
				{OwnerUserId: "user-123", ContactUserId: "user-777"},
			},
		},
	}

	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		connectionsClient: connectionsClient,
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
	if connectionsClient.listContactsReq == nil {
		t.Fatal("expected ListContacts request")
	}
	if connectionsClient.listContactsReq.GetOwnerUserId() != "user-123" {
		t.Fatalf("owner_user_id = %q, want user-123", connectionsClient.listContactsReq.GetOwnerUserId())
	}
	if !strings.Contains(w.Body.String(), "value=\"user-777\"") {
		t.Fatalf("expected contact option in response body")
	}
}

func TestAppCampaignInvitesPageUsesCachedParticipantsForContactOptions(t *testing.T) {
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
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
					},
				},
			},
		},
	}
	inviteClient := &fakeWebInviteClient{
		listInvitesResp: &statev1.ListInvitesResponse{Invites: []*statev1.Invite{}},
	}
	connectionsClient := &fakeConnectionsClient{
		listContactsResp: &connectionsv1.ListContactsResponse{
			Contacts: []*connectionsv1.Contact{
				{OwnerUserId: "user-123", ContactUserId: "user-777"},
			},
		},
	}

	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		connectionsClient: connectionsClient,
		cacheStore:        cacheStore,
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
	h.setCampaignParticipantsCache(context.Background(), "camp-123", []*statev1.Participant{
		{Id: "part-owner", CampaignId: "camp-123", UserId: "user-123", CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER},
	})
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(participantClient.calls) != 0 {
		t.Fatalf("list participants calls = %d, want 0", len(participantClient.calls))
	}
}

func TestRenderAppCampaignInvitesPageHidesWriteActionsWithoutManageAccess(t *testing.T) {
	w := httptest.NewRecorder()

	renderAppCampaignInvitesPage(w, httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil), webtemplates.PageContext{}, "camp-123", []*statev1.Invite{
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

	renderAppCampaignInvitesPage(w, httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil), webtemplates.PageContext{}, "camp-123", []*statev1.Invite{
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

func TestRenderAppCampaignInvitesPageUsesRecipientUsernameCopy(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/camp-123/invites", nil)
	page := webtemplates.PageContext{
		Lang: "en",
		Loc:  webi18n.Printer(language.English),
	}

	renderAppCampaignInvitesPage(w, req, page, "camp-123", []*statev1.Invite{}, true)

	body := w.Body.String()
	if !strings.Contains(body, "Recipient Username or User ID") {
		t.Fatalf("expected recipient label to mention username and user id, got %q", body)
	}
	if !strings.Contains(body, "@username or user id") {
		t.Fatalf("expected placeholder to mention @username fallback, got %q", body)
	}
}

func TestBuildInviteContactOptionsFiltersParticipantsAndPendingInvites(t *testing.T) {
	options := buildInviteContactOptions(
		[]*connectionsv1.Contact{
			{OwnerUserId: "user-1", ContactUserId: "user-200"},
			{OwnerUserId: "user-1", ContactUserId: "user-300"},
			{OwnerUserId: "user-1", ContactUserId: "user-400"},
		},
		[]*statev1.Participant{
			{UserId: "user-200"},
		},
		[]*statev1.Invite{
			{RecipientUserId: "user-300", Status: statev1.InviteStatus_PENDING},
			{RecipientUserId: "user-500", Status: statev1.InviteStatus_CLAIMED},
		},
	)

	if len(options) != 1 {
		t.Fatalf("options len = %d, want 1", len(options))
	}
	if options[0].UserID != "user-400" {
		t.Fatalf("user_id = %q, want user-400", options[0].UserID)
	}
}

func TestListAllContactsRejectsRepeatedPageToken(t *testing.T) {
	h := &handler{
		connectionsClient: &fakeConnectionsClient{
			listContactsPages: map[string]*connectionsv1.ListContactsResponse{
				"": {
					Contacts:      []*connectionsv1.Contact{{OwnerUserId: "user-1", ContactUserId: "user-2"}},
					NextPageToken: "loop",
				},
				"loop": {
					Contacts:      []*connectionsv1.Contact{{OwnerUserId: "user-1", ContactUserId: "user-3"}},
					NextPageToken: "loop",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := h.listAllContacts(ctx, "user-1")
	if err == nil {
		t.Fatal("expected error for repeated page token")
	}
	if !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("error = %v, want repeated page token", err)
	}
}

func TestListAllContactsPaginates(t *testing.T) {
	h := &handler{
		connectionsClient: &fakeConnectionsClient{
			listContactsPages: map[string]*connectionsv1.ListContactsResponse{
				"": {
					Contacts:      []*connectionsv1.Contact{{OwnerUserId: "user-1", ContactUserId: "user-2"}},
					NextPageToken: "next",
				},
				"next": {
					Contacts: []*connectionsv1.Contact{{OwnerUserId: "user-1", ContactUserId: "user-3"}},
				},
			},
		},
	}

	contacts, err := h.listAllContacts(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list all contacts: %v", err)
	}
	if len(contacts) != 2 {
		t.Fatalf("contacts len = %d, want 2", len(contacts))
	}
	if got := contacts[0].GetContactUserId(); got != "user-2" {
		t.Fatalf("contact[0] = %q, want user-2", got)
	}
	if got := contacts[1].GetContactUserId(); got != "user-3" {
		t.Fatalf("contact[1] = %q, want user-3", got)
	}
}

func TestListAllCampaignParticipantsRejectsRepeatedPageToken(t *testing.T) {
	h := &handler{
		participantClient: &fakeWebParticipantClient{
			pages: map[string]*statev1.ListParticipantsResponse{
				"": {
					Participants:  []*statev1.Participant{{Id: "part-1", UserId: "user-1"}},
					NextPageToken: "loop",
				},
				"loop": {
					Participants:  []*statev1.Participant{{Id: "part-2", UserId: "user-2"}},
					NextPageToken: "loop",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := h.listAllCampaignParticipants(ctx, "camp-1")
	if err == nil {
		t.Fatal("expected error for repeated page token")
	}
	if !strings.Contains(err.Error(), "repeated page token") {
		t.Fatalf("error = %v, want repeated page token", err)
	}
}

func TestListAllCampaignParticipantsPaginatesAndCaches(t *testing.T) {
	cacheStore := newFakeWebCacheStore()
	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants:  []*statev1.Participant{{Id: "part-1", UserId: "user-1"}},
				NextPageToken: "next",
			},
			"next": {
				Participants: []*statev1.Participant{{Id: "part-2", UserId: "user-2"}},
			},
		},
	}
	h := &handler{
		cacheStore:        cacheStore,
		participantClient: participantClient,
	}

	first, err := h.listAllCampaignParticipants(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("first list participants: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("participants len = %d, want 2", len(first))
	}
	if len(participantClient.calls) != 2 {
		t.Fatalf("participant calls = %d, want 2", len(participantClient.calls))
	}

	second, err := h.listAllCampaignParticipants(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("second list participants: %v", err)
	}
	if len(second) != 2 {
		t.Fatalf("cached participants len = %d, want 2", len(second))
	}
	if len(participantClient.calls) != 2 {
		t.Fatalf("participant calls after cache = %d, want 2", len(participantClient.calls))
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
	if inviteClient.listInvitesReq == nil {
		t.Fatalf("expected ListInvites to be called for member")
	}
}

func TestAppCampaignInvitesPageDoesNotRequireParticipantLookup(t *testing.T) {
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
	if len(participantClient.calls) != 0 {
		t.Fatalf("list participants calls = %d, want %d", len(participantClient.calls), 0)
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
