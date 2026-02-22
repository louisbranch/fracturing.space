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
	webtemplates "github.com/louisbranch/fracturing.space/internal/services/web/templates"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAppCampaignInvitesPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/invites/create", strings.NewReader("participant_id=part-1"))
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
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/invites/revoke", strings.NewReader("invite_id=inv-1"))
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
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/campaigns/camp-123/invites" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-123/invites")
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
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/invites/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/campaigns/camp-123/invites" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-123/invites")
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
	authClient := &fakeAuthClient{
		listContactsResp: &authv1.ListContactsResponse{
			Contacts: []*authv1.Contact{
				{OwnerUserId: "user-123", ContactUserId: "user-777"},
			},
		},
	}

	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		authClient:        authClient,
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if authClient.listContactsReq == nil {
		t.Fatal("expected ListContacts request")
	}
	if authClient.listContactsReq.GetOwnerUserId() != "user-123" {
		t.Fatalf("owner_user_id = %q, want user-123", authClient.listContactsReq.GetOwnerUserId())
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
	authClient := &fakeAuthClient{
		listContactsResp: &authv1.ListContactsResponse{
			Contacts: []*authv1.Contact{
				{OwnerUserId: "user-123", ContactUserId: "user-777"},
			},
		},
	}

	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		authClient:        authClient,
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil)
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

	renderAppCampaignInvitesPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil), webtemplates.PageContext{}, "camp-123", []*statev1.Invite{
		{Id: "inv-1", CampaignId: "camp-123", RecipientUserId: "user-456"},
	}, false)

	body := w.Body.String()
	if strings.Contains(body, "/campaigns/camp-123/invites/create") {
		t.Fatalf("expected create invite form to be hidden")
	}
	if strings.Contains(body, "/campaigns/camp-123/invites/revoke") {
		t.Fatalf("expected revoke invite form to be hidden")
	}
}

func TestRenderAppCampaignInvitesPageSkipsRevokeForMissingInviteID(t *testing.T) {
	w := httptest.NewRecorder()

	renderAppCampaignInvitesPage(w, httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil), webtemplates.PageContext{}, "camp-123", []*statev1.Invite{
		{CampaignId: "camp-123", RecipientUserId: "user-456"},
	}, true)

	body := w.Body.String()
	if !strings.Contains(body, "unknown-invite - user-456") {
		t.Fatalf("expected fallback invite id in response body")
	}
	if strings.Contains(body, "/campaigns/camp-123/invites/revoke") {
		t.Fatalf("expected revoke invite form to be hidden for missing invite id")
	}
}

func TestBuildInviteContactOptionsFiltersParticipantsAndPendingInvites(t *testing.T) {
	options := buildInviteContactOptions(
		[]*authv1.Contact{
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
		authClient: &fakeAuthClient{
			listContactsPages: map[string]*authv1.ListContactsResponse{
				"": {
					Contacts:      []*authv1.Contact{{OwnerUserId: "user-1", ContactUserId: "user-2"}},
					NextPageToken: "loop",
				},
				"loop": {
					Contacts:      []*authv1.Contact{{OwnerUserId: "user-1", ContactUserId: "user-3"}},
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
		authClient: &fakeAuthClient{
			listContactsPages: map[string]*authv1.ListContactsResponse{
				"": {
					Contacts:      []*authv1.Contact{{OwnerUserId: "user-1", ContactUserId: "user-2"}},
					NextPageToken: "next",
				},
				"next": {
					Contacts: []*authv1.Contact{{OwnerUserId: "user-1", ContactUserId: "user-3"}},
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
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
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/invites/revoke", strings.NewReader(form.Encode()))
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/invites", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
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
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/invites/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
