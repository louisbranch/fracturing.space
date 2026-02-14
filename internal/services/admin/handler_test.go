package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TestWebPageRendering verifies layout rendering based on HTMX requests.
func TestWebPageRendering(t *testing.T) {
	handler := NewHandler(nil)

	tests := []struct {
		name        string
		path        string
		htmx        bool
		contains    []string
		notContains []string
	}{
		{
			name: "home full page",
			path: "/",
			contains: []string{
				"<!doctype html>",
				branding.AppName,
			},
			notContains: []string{
				"<h2>Campaigns</h2>",
			},
		},
		{
			name: "campaigns full page",
			path: "/campaigns",
			contains: []string{
				"<!doctype html>",
				branding.AppName,
				"<h2>Campaigns</h2>",
			},
		},
		{
			name: "campaigns htmx",
			path: "/campaigns",
			htmx: true,
			contains: []string{
				"<h2>Campaigns</h2>",
			},
			notContains: []string{
				"<!doctype html>",
				branding.AppName,
				"<html",
			},
		},
		{
			name: "systems full page",
			path: "/systems",
			contains: []string{
				"<!doctype html>",
				branding.AppName,
				"<h2>Systems</h2>",
			},
		},
		{
			name: "systems htmx",
			path: "/systems",
			htmx: true,
			contains: []string{
				"<h2>Systems</h2>",
			},
			notContains: []string{
				"<!doctype html>",
				branding.AppName,
				"<html",
			},
		},
		{
			name: "campaign detail full page",
			path: "/campaigns/camp-123",
			contains: []string{
				"<!doctype html>",
				branding.AppName,
				"Campaign service unavailable.",
				"<h2>Campaign</h2>",
			},
		},
		{
			name: "campaign detail htmx",
			path: "/campaigns/camp-123",
			htmx: true,
			contains: []string{
				"Campaign service unavailable.",
				"<h2>Campaign</h2>",
			},
			notContains: []string{
				"<!doctype html>",
				branding.AppName,
				"<html",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com"+tc.path, nil)
			if tc.htmx {
				req.Header.Set("HX-Request", "true")
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
			}

			body := recorder.Body.String()
			for _, expected := range tc.contains {
				assertContains(t, body, expected)
			}
			for _, unexpected := range tc.notContains {
				assertNotContains(t, body, unexpected)
			}
		})
	}
}

func TestSystemsTableRendersDefaultBadge(t *testing.T) {
	systemClient := &testSystemClient{
		listResponse: &statev1.ListGameSystemsResponse{
			Systems: []*statev1.GameSystemInfo{
				{
					Id:                  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
					Name:                "Daggerheart",
					Version:             "1.0.0",
					ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL,
					OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
					AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA,
					IsDefault:           true,
				},
			},
		},
	}
	handler := NewHandler(testClientProvider{system: systemClient})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/table", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()
	assertContains(t, body, "Daggerheart")
	assertContains(t, body, "1.0.0")
	assertContains(t, body, "Default")
}

// TestCampaignSessionsRoute verifies session routes render pages correctly.
func TestCampaignSessionsRoute(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("sessions htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/sessions", nil)
		req.Header.Set("HX-Request", "true")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}

		body := recorder.Body.String()
		assertContains(t, body, "<h3>Sessions</h3>")
		assertNotContains(t, body, "<!doctype html>")
	})

	t.Run("sessions full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/sessions", nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}

		body := recorder.Body.String()
		assertContains(t, body, "<!doctype html>")
		assertContains(t, body, "<h3>Sessions</h3>")
	})

	t.Run("sessions table htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/sessions/table", nil)
		req.Header.Set("HX-Request", "true")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}

		body := recorder.Body.String()
		assertContains(t, body, "Session service unavailable.")
	})
}

type testClientProvider struct {
	auth        authv1.AuthServiceClient
	campaign    statev1.CampaignServiceClient
	invite      statev1.InviteServiceClient
	participant statev1.ParticipantServiceClient
	system      statev1.SystemServiceClient
}

func (p testClientProvider) CampaignClient() statev1.CampaignServiceClient {
	return p.campaign
}

func (p testClientProvider) SessionClient() statev1.SessionServiceClient {
	return nil
}

func (p testClientProvider) CharacterClient() statev1.CharacterServiceClient {
	return nil
}

func (p testClientProvider) ParticipantClient() statev1.ParticipantServiceClient {
	return p.participant
}

func (p testClientProvider) InviteClient() statev1.InviteServiceClient {
	return p.invite
}

func (p testClientProvider) EventClient() statev1.EventServiceClient {
	return nil
}

func (p testClientProvider) SnapshotClient() statev1.SnapshotServiceClient {
	return nil
}

func (p testClientProvider) StatisticsClient() statev1.StatisticsServiceClient {
	return nil
}

func (p testClientProvider) SystemClient() statev1.SystemServiceClient {
	return p.system
}

func (p testClientProvider) AuthClient() authv1.AuthServiceClient {
	return p.auth
}

type testAuthClient struct {
	user          *authv1.User
	lastMetadata  metadata.MD
	lastUserIDReq string
}

type testSystemClient struct {
	listResponse *statev1.ListGameSystemsResponse
	getResponse  *statev1.GetGameSystemResponse
}

func (c *testSystemClient) ListGameSystems(ctx context.Context, in *statev1.ListGameSystemsRequest, opts ...grpc.CallOption) (*statev1.ListGameSystemsResponse, error) {
	return c.listResponse, nil
}

func (c *testSystemClient) GetGameSystem(ctx context.Context, in *statev1.GetGameSystemRequest, opts ...grpc.CallOption) (*statev1.GetGameSystemResponse, error) {
	return c.getResponse, nil
}

func (c *testAuthClient) CreateUser(ctx context.Context, in *authv1.CreateUserRequest, opts ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	return &authv1.CreateUserResponse{User: c.user}, nil
}

func (c *testAuthClient) IssueJoinGrant(ctx context.Context, in *authv1.IssueJoinGrantRequest, opts ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return &authv1.IssueJoinGrantResponse{}, nil
}

func (c *testAuthClient) GetUser(ctx context.Context, in *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	c.lastUserIDReq = in.GetUserId()
	md, _ := metadata.FromOutgoingContext(ctx)
	c.lastMetadata = md
	return &authv1.GetUserResponse{User: c.user}, nil
}

func (c *testAuthClient) ListUsers(ctx context.Context, in *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	return &authv1.ListUsersResponse{}, nil
}

type testCampaignClient struct {
	lastMetadata metadata.MD
	lastRequest  *statev1.CreateCampaignRequest
	response     *statev1.CreateCampaignResponse
}

func (c *testCampaignClient) CreateCampaign(ctx context.Context, in *statev1.CreateCampaignRequest, opts ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	c.lastRequest = in
	md, _ := metadata.FromOutgoingContext(ctx)
	c.lastMetadata = md
	if c.response != nil {
		return c.response, nil
	}
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "camp-123"}}, nil
}

func (c *testCampaignClient) ListCampaigns(ctx context.Context, in *statev1.ListCampaignsRequest, opts ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	return &statev1.ListCampaignsResponse{}, nil
}

func (c *testCampaignClient) GetCampaign(ctx context.Context, in *statev1.GetCampaignRequest, opts ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	return &statev1.GetCampaignResponse{}, nil
}

func (c *testCampaignClient) EndCampaign(ctx context.Context, in *statev1.EndCampaignRequest, opts ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
	return &statev1.EndCampaignResponse{}, nil
}

func (c *testCampaignClient) ArchiveCampaign(ctx context.Context, in *statev1.ArchiveCampaignRequest, opts ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
	return &statev1.ArchiveCampaignResponse{}, nil
}

func (c *testCampaignClient) RestoreCampaign(ctx context.Context, in *statev1.RestoreCampaignRequest, opts ...grpc.CallOption) (*statev1.RestoreCampaignResponse, error) {
	return &statev1.RestoreCampaignResponse{}, nil
}

type testParticipantClient struct {
	participants []*statev1.Participant
}

func (c *testParticipantClient) CreateParticipant(ctx context.Context, in *statev1.CreateParticipantRequest, opts ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	return &statev1.CreateParticipantResponse{}, nil
}

func (c *testParticipantClient) UpdateParticipant(ctx context.Context, in *statev1.UpdateParticipantRequest, opts ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error) {
	return &statev1.UpdateParticipantResponse{}, nil
}

func (c *testParticipantClient) DeleteParticipant(ctx context.Context, in *statev1.DeleteParticipantRequest, opts ...grpc.CallOption) (*statev1.DeleteParticipantResponse, error) {
	return &statev1.DeleteParticipantResponse{}, nil
}

func (c *testParticipantClient) GetParticipant(ctx context.Context, in *statev1.GetParticipantRequest, opts ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	return &statev1.GetParticipantResponse{}, nil
}

func (c *testParticipantClient) ListParticipants(ctx context.Context, in *statev1.ListParticipantsRequest, opts ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	return &statev1.ListParticipantsResponse{Participants: c.participants}, nil
}

type testInviteClient struct {
	lastMetadata        metadata.MD
	lastListMetadata    metadata.MD
	lastPendingUserReq  *statev1.ListPendingInvitesForUserRequest
	pendingUserResponse *statev1.ListPendingInvitesForUserResponse
}

func (c *testInviteClient) CreateInvite(ctx context.Context, in *statev1.CreateInviteRequest, opts ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
	return &statev1.CreateInviteResponse{}, nil
}

func (c *testInviteClient) ClaimInvite(ctx context.Context, in *statev1.ClaimInviteRequest, opts ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
	return &statev1.ClaimInviteResponse{}, nil
}

func (c *testInviteClient) GetInvite(ctx context.Context, in *statev1.GetInviteRequest, opts ...grpc.CallOption) (*statev1.GetInviteResponse, error) {
	return &statev1.GetInviteResponse{}, nil
}

func (c *testInviteClient) ListInvites(ctx context.Context, in *statev1.ListInvitesRequest, opts ...grpc.CallOption) (*statev1.ListInvitesResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	c.lastListMetadata = md
	return &statev1.ListInvitesResponse{}, nil
}

func (c *testInviteClient) ListPendingInvites(ctx context.Context, in *statev1.ListPendingInvitesRequest, opts ...grpc.CallOption) (*statev1.ListPendingInvitesResponse, error) {
	return &statev1.ListPendingInvitesResponse{}, nil
}

func (c *testInviteClient) ListPendingInvitesForUser(ctx context.Context, in *statev1.ListPendingInvitesForUserRequest, opts ...grpc.CallOption) (*statev1.ListPendingInvitesForUserResponse, error) {
	c.lastPendingUserReq = in
	md, _ := metadata.FromOutgoingContext(ctx)
	c.lastMetadata = md
	if c.pendingUserResponse != nil {
		return c.pendingUserResponse, nil
	}
	return &statev1.ListPendingInvitesForUserResponse{}, nil
}

func (c *testInviteClient) RevokeInvite(ctx context.Context, in *statev1.RevokeInviteRequest, opts ...grpc.CallOption) (*statev1.RevokeInviteResponse, error) {
	return &statev1.RevokeInviteResponse{}, nil
}

func TestCampaignCreateFlow(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	form := url.Values{}
	form.Set("user_id", "user-123")
	form.Set("name", "New Campaign")
	form.Set("system", "daggerheart")
	form.Set("gm_mode", "human")
	form.Set("theme_prompt", "Misty marshes")
	form.Set("creator_display_name", "Owner")

	req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, recorder.Code)
	}
	location := recorder.Header().Get("Location")
	if location != "/campaigns/camp-123" {
		t.Fatalf("expected redirect to /campaigns/camp-123, got %q", location)
	}
	values := campaignClient.lastMetadata.Get(grpcmeta.UserIDHeader)
	if len(values) != 1 || values[0] != "user-123" {
		t.Fatalf("expected metadata %s to be set, got %v", grpcmeta.UserIDHeader, values)
	}
	if campaignClient.lastRequest == nil {
		t.Fatalf("expected CreateCampaign request to be captured")
	}
	if campaignClient.lastRequest.GetName() != "New Campaign" {
		t.Fatalf("expected campaign name to be set")
	}
	if campaignClient.lastRequest.GetSystem() != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("expected system to be daggerheart")
	}
	if campaignClient.lastRequest.GetGmMode() != statev1.GmMode_HUMAN {
		t.Fatalf("expected gm mode to be human")
	}
}

func TestCampaignCreateHTMXRedirect(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	form := url.Values{}
	form.Set("user_id", "user-htmx")
	form.Set("name", "HTMX Campaign")
	form.Set("system", "daggerheart")
	form.Set("gm_mode", "human")

	req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("HX-Request", "true")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, recorder.Code)
	}
	location := recorder.Header().Get("Location")
	if location != "/campaigns/camp-123" {
		t.Fatalf("expected Location to /campaigns/camp-123, got %q", location)
	}
	redirect := recorder.Header().Get("HX-Redirect")
	if redirect != "/campaigns/camp-123" {
		t.Fatalf("expected HX-Redirect to /campaigns/camp-123, got %q", redirect)
	}
}

func TestCampaignCreateValidationErrors(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	loc := i18n.Printer(i18n.Default())

	tests := []struct {
		name     string
		form     url.Values
		expected string
	}{
		{
			name: "empty system",
			form: url.Values{
				"user_id": []string{"user-123"},
				"name":    []string{"New Campaign"},
				"gm_mode": []string{"human"},
			},
			expected: loc.Sprintf("error.campaign_system_required"),
		},
		{
			name: "invalid system",
			form: url.Values{
				"user_id": []string{"user-123"},
				"name":    []string{"New Campaign"},
				"system":  []string{"bad"},
				"gm_mode": []string{"human"},
			},
			expected: loc.Sprintf("error.campaign_system_invalid"),
		},
		{
			name: "empty gm mode",
			form: url.Values{
				"user_id": []string{"user-123"},
				"name":    []string{"New Campaign"},
				"system":  []string{"daggerheart"},
			},
			expected: loc.Sprintf("error.campaign_gm_mode_required"),
		},
		{
			name: "invalid gm mode",
			form: url.Values{
				"user_id": []string{"user-123"},
				"name":    []string{"New Campaign"},
				"system":  []string{"daggerheart"},
				"gm_mode": []string{"robot"},
			},
			expected: loc.Sprintf("error.campaign_gm_mode_invalid"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(tc.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Origin", "http://example.com")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
			}
			assertContains(t, recorder.Body.String(), tc.expected)
		})
	}
}

func TestCampaignCreateImpersonationOverridesUserID(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	sessionID := "session-impersonate"
	webHandler.impersonation.Set(sessionID, impersonationSession{userID: "user-imp", displayName: "Impersonated"})

	form := url.Values{}
	form.Set("user_id", "user-other")
	form.Set("name", "Impersonated Campaign")
	form.Set("system", "daggerheart")
	form.Set("gm_mode", "human")
	form.Set("creator_display_name", "Other")

	req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: sessionID})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	values := campaignClient.lastMetadata.Get(grpcmeta.UserIDHeader)
	if len(values) != 1 || values[0] != "user-imp" {
		t.Fatalf("expected metadata %s to be impersonation user, got %v", grpcmeta.UserIDHeader, values)
	}
	if campaignClient.lastRequest == nil || campaignClient.lastRequest.GetCreatorDisplayName() != "Impersonated" {
		t.Fatalf("expected creator display name to be impersonation display name")
	}
}

func TestImpersonationFlow(t *testing.T) {
	user := &authv1.User{Id: "user-1", DisplayName: "Test User"}
	authClient := &testAuthClient{user: user}
	provider := testClientProvider{auth: authClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	t.Run("csrf required", func(t *testing.T) {
		body := strings.NewReader("user_id=user-1")
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
		}
	})

	t.Run("impersonate sets cookie and indicator", func(t *testing.T) {
		body := strings.NewReader("user_id=user-1")
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}
		cookies := recorder.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == impersonationCookieName {
				sessionCookie = cookie
				break
			}
		}
		if sessionCookie == nil || sessionCookie.Value == "" {
			t.Fatalf("expected impersonation cookie to be set")
		}
		bodyText := recorder.Body.String()
		assertContains(t, bodyText, "Impersonating")
		assertContains(t, bodyText, "Test User")
	})

	t.Run("impersonate clears previous session", func(t *testing.T) {
		previousSessionID := "session-old"
		webHandler.impersonation.Set(previousSessionID, impersonationSession{userID: "user-old"})

		form := url.Values{}
		form.Set("user_id", "user-1")
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: previousSessionID})
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if _, ok := webHandler.impersonation.Get(previousSessionID); ok {
			t.Fatalf("expected previous session to be cleared")
		}
	})

	t.Run("logout clears cookie", func(t *testing.T) {
		sessionID := "session-logout"
		webHandler.impersonation.Set(sessionID, impersonationSession{userID: "user-logout"})

		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/logout", nil)
		req.Header.Set("Origin", "http://example.com")
		req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: sessionID})
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}
		cookies := recorder.Result().Cookies()
		var clearedCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == impersonationCookieName {
				clearedCookie = cookie
				break
			}
		}
		if clearedCookie == nil || clearedCookie.MaxAge != -1 {
			t.Fatalf("expected impersonation cookie to be cleared")
		}
	})
}

func TestImpersonationMetadataInjection(t *testing.T) {
	user := &authv1.User{Id: "user-lookup", DisplayName: "Lookup"}
	authClient := &testAuthClient{user: user}
	provider := testClientProvider{auth: authClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	sessionID := "session-meta"
	webHandler.impersonation.Set(sessionID, impersonationSession{userID: "user-impersonated"})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/user-lookup", nil)
	req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: sessionID})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	values := authClient.lastMetadata.Get(grpcmeta.UserIDHeader)
	if len(values) != 1 || values[0] != "user-impersonated" {
		t.Fatalf("expected metadata %s to be set, got %v", grpcmeta.UserIDHeader, values)
	}
}

func TestUserDetailPendingInvitesUsesImpersonationMetadata(t *testing.T) {
	user := &authv1.User{Id: "user-lookup", DisplayName: "Lookup"}
	authClient := &testAuthClient{user: user}
	inviteClient := &testInviteClient{}
	provider := testClientProvider{auth: authClient, invite: inviteClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	sessionID := "session-invites"
	webHandler.impersonation.Set(sessionID, impersonationSession{userID: "user-lookup"})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/user-lookup", nil)
	req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: sessionID})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	values := inviteClient.lastMetadata.Get(grpcmeta.UserIDHeader)
	if len(values) != 1 || values[0] != "user-lookup" {
		t.Fatalf("expected metadata %s to be set, got %v", grpcmeta.UserIDHeader, values)
	}
	if inviteClient.lastPendingUserReq == nil {
		t.Fatalf("expected pending invites request to be captured")
	}
}

func TestInvitesTableUsesParticipantMetadataForImpersonation(t *testing.T) {
	participantClient := &testParticipantClient{participants: []*statev1.Participant{
		{Id: "participant-1", CampaignId: "camp-123", UserId: "user-imp"},
	}}
	inviteClient := &testInviteClient{}
	provider := testClientProvider{participant: participantClient, invite: inviteClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	sessionID := "session-invites"
	webHandler.impersonation.Set(sessionID, impersonationSession{userID: "user-imp"})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-123/invites/table", nil)
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: sessionID})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	values := inviteClient.lastListMetadata.Get(grpcmeta.ParticipantIDHeader)
	if len(values) != 1 || values[0] != "participant-1" {
		t.Fatalf("expected metadata %s to be set, got %v", grpcmeta.ParticipantIDHeader, values)
	}
}

// assertContains fails the test when the body lacks the expected fragment.
func assertContains(t *testing.T, body string, expected string) {
	t.Helper()
	if !strings.Contains(body, expected) {
		t.Fatalf("expected response to contain %q", expected)
	}
}

// assertNotContains fails the test when the body includes an unexpected fragment.
func assertNotContains(t *testing.T, body string, unexpected string) {
	t.Helper()
	if strings.Contains(body, unexpected) {
		t.Fatalf("expected response to not contain %q", unexpected)
	}
}

// TestEscapeAIP160StringLiteral verifies special character escaping.
func TestEscapeAIP160StringLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{`with"quote`, `with\"quote`},
		{`with\backslash`, `with\\backslash`},
		{`both\"chars`, `both\\\"chars`},
		{`a"b\c"d`, `a\"b\\c\"d`},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := escapeAIP160StringLiteral(tc.input)
			if result != tc.expected {
				t.Errorf("escapeAIP160StringLiteral(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFormatActorType(t *testing.T) {
	loc := i18n.Printer(i18n.Default())

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "system",
			input:    "system",
			expected: loc.Sprintf("filter.actor.system"),
		},
		{
			name:     "participant",
			input:    "participant",
			expected: loc.Sprintf("filter.actor.participant"),
		},
		{
			name:     "gm",
			input:    "gm",
			expected: loc.Sprintf("filter.actor.gm"),
		},
		{
			name:     "fallback",
			input:    "custom",
			expected: "custom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatActorType(tc.input, loc)
			if result != tc.expected {
				t.Errorf("formatActorType(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestFormatEventType(t *testing.T) {
	loc := i18n.Printer(i18n.Default())

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"campaign_created", "campaign.created", loc.Sprintf("event.campaign_created")},
		{"campaign_forked", "campaign.forked", loc.Sprintf("event.campaign_forked")},
		{"campaign_updated", "campaign.updated", loc.Sprintf("event.campaign_updated")},
		{"participant_joined", "participant.joined", loc.Sprintf("event.participant_joined")},
		{"participant_left", "participant.left", loc.Sprintf("event.participant_left")},
		{"participant_updated", "participant.updated", loc.Sprintf("event.participant_updated")},
		{"character_created", "character.created", loc.Sprintf("event.character_created")},
		{"character_deleted", "character.deleted", loc.Sprintf("event.character_deleted")},
		{"character_updated", "character.updated", loc.Sprintf("event.character_updated")},
		{"character_profile_updated", "character.profile_updated", loc.Sprintf("event.character_profile_updated")},
		{"session_started", "session.started", loc.Sprintf("event.session_started")},
		{"session_ended", "session.ended", loc.Sprintf("event.session_ended")},
		{"session_gate_opened", "session.gate_opened", loc.Sprintf("event.session_gate_opened")},
		{"session_gate_resolved", "session.gate_resolved", loc.Sprintf("event.session_gate_resolved")},
		{"session_gate_abandoned", "session.gate_abandoned", loc.Sprintf("event.session_gate_abandoned")},
		{"session_spotlight_set", "session.spotlight_set", loc.Sprintf("event.session_spotlight_set")},
		{"session_spotlight_cleared", "session.spotlight_cleared", loc.Sprintf("event.session_spotlight_cleared")},
		{"invite_created", "invite.created", loc.Sprintf("event.invite_created")},
		{"invite_updated", "invite.updated", loc.Sprintf("event.invite_updated")},
		{"action_roll_resolved", "action.roll_resolved", loc.Sprintf("event.action_roll_resolved")},
		{"action_outcome_applied", "action.outcome_applied", loc.Sprintf("event.action_outcome_applied")},
		{"action_outcome_rejected", "action.outcome_rejected", loc.Sprintf("event.action_outcome_rejected")},
		{"action_note_added", "action.note_added", loc.Sprintf("event.action_note_added")},
		{"action_character_state_patched", "action.character_state_patched", loc.Sprintf("event.action_character_state_patched")},
		{"action_gm_fear_changed", "action.gm_fear_changed", loc.Sprintf("event.action_gm_fear_changed")},
		{"action_death_move_resolved", "action.death_move_resolved", loc.Sprintf("event.action_death_move_resolved")},
		{"action_blaze_of_glory_resolved", "action.blaze_of_glory_resolved", loc.Sprintf("event.action_blaze_of_glory_resolved")},
		{"action_attack_resolved", "action.attack_resolved", loc.Sprintf("event.action_attack_resolved")},
		{"action_reaction_resolved", "action.reaction_resolved", loc.Sprintf("event.action_reaction_resolved")},
		{"action_damage_roll_resolved", "action.damage_roll_resolved", loc.Sprintf("event.action_damage_roll_resolved")},
		{"action_adversary_action_resolved", "action.adversary_action_resolved", loc.Sprintf("event.action_adversary_action_resolved")},
		{"fallback_underscore", "custom.some_event_type", "Some event type"},
		{"fallback_simple", "custom.hello", "Hello"},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatEventType(tc.input, loc)
			if result != tc.expected {
				t.Errorf("formatEventType(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestRequestScheme(t *testing.T) {
	// Default (no TLS, no header)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := requestScheme(r); got != "http" {
		t.Errorf("requestScheme default = %q, want %q", got, "http")
	}

	// X-Forwarded-Proto header
	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-Proto", "https")
	if got := requestScheme(r); got != "https" {
		t.Errorf("requestScheme X-Forwarded-Proto = %q, want %q", got, "https")
	}

	// X-Forwarded-Proto with multiple values (comma-separated)
	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Forwarded-Proto", "https, http")
	if got := requestScheme(r); got != "https" {
		t.Errorf("requestScheme X-Forwarded-Proto multi = %q, want %q", got, "https")
	}

	// Nil request
	if got := requestScheme(nil); got != "http" {
		t.Errorf("requestScheme nil = %q, want %q", got, "http")
	}
}

func TestSameOrigin(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://example.com/page", nil)
	r.Host = "example.com"

	// Valid same origin
	if !sameOrigin("http://example.com", r) {
		t.Error("expected same origin for matching host")
	}

	// Case-insensitive host
	if !sameOrigin("http://EXAMPLE.COM", r) {
		t.Error("expected same origin for case-insensitive host")
	}

	// Different host
	if sameOrigin("http://other.com", r) {
		t.Error("expected different origin for different host")
	}

	// Empty URL
	if sameOrigin("", r) {
		t.Error("expected false for empty URL")
	}

	// Null string
	if sameOrigin("null", r) {
		t.Error("expected false for null string")
	}

	// Invalid URL
	if sameOrigin("://invalid", r) {
		t.Error("expected false for invalid URL")
	}

	// Nil request
	if sameOrigin("http://example.com", nil) {
		t.Error("expected false for nil request")
	}

	// Different scheme
	r2 := httptest.NewRequest(http.MethodGet, "http://example.com/page", nil)
	r2.Host = "example.com"
	if sameOrigin("https://example.com", r2) {
		t.Error("expected different origin for different scheme")
	}

	// No scheme in URL (should pass if host matches)
	if !sameOrigin("//example.com/path", r) {
		t.Error("expected same origin when no scheme in URL")
	}
}

func TestRequireSameOrigin(t *testing.T) {
	loc := i18n.Printer(i18n.Default())

	// With valid Origin header
	r := httptest.NewRequest(http.MethodPost, "http://example.com/page", nil)
	r.Host = "example.com"
	r.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	if !requireSameOrigin(w, r, loc) {
		t.Error("expected true for valid Origin header")
	}

	// With valid Referer header (no Origin)
	r = httptest.NewRequest(http.MethodPost, "http://example.com/page", nil)
	r.Host = "example.com"
	r.Header.Set("Referer", "http://example.com/other")
	w = httptest.NewRecorder()
	if !requireSameOrigin(w, r, loc) {
		t.Error("expected true for valid Referer header")
	}

	// Missing both Origin and Referer
	r = httptest.NewRequest(http.MethodPost, "http://example.com/page", nil)
	r.Host = "example.com"
	w = httptest.NewRecorder()
	if requireSameOrigin(w, r, loc) {
		t.Error("expected false when both Origin and Referer missing")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 status, got %d", w.Code)
	}

	// Nil request
	w = httptest.NewRecorder()
	if requireSameOrigin(w, nil, loc) {
		t.Error("expected false for nil request")
	}

	// Invalid Origin
	r = httptest.NewRequest(http.MethodPost, "http://example.com/page", nil)
	r.Host = "example.com"
	r.Header.Set("Origin", "http://evil.com")
	w = httptest.NewRecorder()
	if requireSameOrigin(w, r, loc) {
		t.Error("expected false for invalid Origin")
	}

	// Invalid Referer (no Origin)
	r = httptest.NewRequest(http.MethodPost, "http://example.com/page", nil)
	r.Host = "example.com"
	r.Header.Set("Referer", "http://evil.com/page")
	w = httptest.NewRecorder()
	if requireSameOrigin(w, r, loc) {
		t.Error("expected false for invalid Referer")
	}
}
