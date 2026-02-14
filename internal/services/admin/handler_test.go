package admin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/branding"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"golang.org/x/text/language"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
	users         []*authv1.User
	lastMetadata  metadata.MD
	lastUserIDReq string
	getUserErr    error
	listUsersErr  error
	createUserErr error
}

type testSystemClient struct {
	listResponse *statev1.ListGameSystemsResponse
	getResponse  *statev1.GetGameSystemResponse
	listErr      error
	getErr       error
}

func (c *testSystemClient) ListGameSystems(ctx context.Context, in *statev1.ListGameSystemsRequest, opts ...grpc.CallOption) (*statev1.ListGameSystemsResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	return c.listResponse, nil
}

func (c *testSystemClient) GetGameSystem(ctx context.Context, in *statev1.GetGameSystemRequest, opts ...grpc.CallOption) (*statev1.GetGameSystemResponse, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}
	return c.getResponse, nil
}

func (c *testAuthClient) CreateUser(ctx context.Context, in *authv1.CreateUserRequest, opts ...grpc.CallOption) (*authv1.CreateUserResponse, error) {
	if c.createUserErr != nil {
		return nil, c.createUserErr
	}
	return &authv1.CreateUserResponse{User: c.user}, nil
}

func (c *testAuthClient) IssueJoinGrant(ctx context.Context, in *authv1.IssueJoinGrantRequest, opts ...grpc.CallOption) (*authv1.IssueJoinGrantResponse, error) {
	return &authv1.IssueJoinGrantResponse{}, nil
}

func (c *testAuthClient) GetUser(ctx context.Context, in *authv1.GetUserRequest, opts ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	c.lastUserIDReq = in.GetUserId()
	md, _ := metadata.FromOutgoingContext(ctx)
	c.lastMetadata = md
	if c.getUserErr != nil {
		return nil, c.getUserErr
	}
	return &authv1.GetUserResponse{User: c.user}, nil
}

func (c *testAuthClient) ListUsers(ctx context.Context, in *authv1.ListUsersRequest, opts ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	if c.listUsersErr != nil {
		return nil, c.listUsersErr
	}
	return &authv1.ListUsersResponse{Users: c.users}, nil
}

type testCampaignClient struct {
	lastMetadata    metadata.MD
	lastRequest     *statev1.CreateCampaignRequest
	response        *statev1.CreateCampaignResponse
	getCampaignResp *statev1.GetCampaignResponse
	listResponse    *statev1.ListCampaignsResponse
	createErr       error
	listErr         error
	getCampaignErr  error
}

func (c *testCampaignClient) CreateCampaign(ctx context.Context, in *statev1.CreateCampaignRequest, opts ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	c.lastRequest = in
	md, _ := metadata.FromOutgoingContext(ctx)
	c.lastMetadata = md
	if c.createErr != nil {
		return nil, c.createErr
	}
	if c.response != nil {
		return c.response, nil
	}
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "camp-123"}}, nil
}

func (c *testCampaignClient) ListCampaigns(ctx context.Context, in *statev1.ListCampaignsRequest, opts ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResponse != nil {
		return c.listResponse, nil
	}
	return &statev1.ListCampaignsResponse{}, nil
}

func (c *testCampaignClient) GetCampaign(ctx context.Context, in *statev1.GetCampaignRequest, opts ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	if c.getCampaignErr != nil {
		return nil, c.getCampaignErr
	}
	if c.getCampaignResp != nil {
		return c.getCampaignResp, nil
	}
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
	listErr      error
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
	if c.listErr != nil {
		return nil, c.listErr
	}
	return &statev1.ListParticipantsResponse{Participants: c.participants}, nil
}

type testInviteClient struct {
	lastMetadata        metadata.MD
	lastListMetadata    metadata.MD
	lastPendingUserReq  *statev1.ListPendingInvitesForUserRequest
	pendingUserResponse *statev1.ListPendingInvitesForUserResponse
	listInvitesResponse *statev1.ListInvitesResponse
	listInvitesErr      error
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
	if c.listInvitesErr != nil {
		return nil, c.listInvitesErr
	}
	if c.listInvitesResponse != nil {
		return c.listInvitesResponse, nil
	}
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

	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/user-lookup/invites", nil)
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
	assertContains(t, recorder.Body.String(), "Pending Invites")
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

// --- Additional test clients ---

type testSessionClient struct {
	listResponse *statev1.ListSessionsResponse
	getResponse  *statev1.GetSessionResponse
	listErr      error
}

func (c *testSessionClient) StartSession(ctx context.Context, in *statev1.StartSessionRequest, opts ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	return &statev1.StartSessionResponse{}, nil
}

func (c *testSessionClient) ListSessions(ctx context.Context, in *statev1.ListSessionsRequest, opts ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResponse != nil {
		return c.listResponse, nil
	}
	return &statev1.ListSessionsResponse{}, nil
}

func (c *testSessionClient) GetSession(ctx context.Context, in *statev1.GetSessionRequest, opts ...grpc.CallOption) (*statev1.GetSessionResponse, error) {
	if c.getResponse != nil {
		return c.getResponse, nil
	}
	return &statev1.GetSessionResponse{}, nil
}

func (c *testSessionClient) EndSession(ctx context.Context, in *statev1.EndSessionRequest, opts ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	return &statev1.EndSessionResponse{}, nil
}

func (c *testSessionClient) OpenSessionGate(ctx context.Context, in *statev1.OpenSessionGateRequest, opts ...grpc.CallOption) (*statev1.OpenSessionGateResponse, error) {
	return &statev1.OpenSessionGateResponse{}, nil
}

func (c *testSessionClient) ResolveSessionGate(ctx context.Context, in *statev1.ResolveSessionGateRequest, opts ...grpc.CallOption) (*statev1.ResolveSessionGateResponse, error) {
	return &statev1.ResolveSessionGateResponse{}, nil
}

func (c *testSessionClient) AbandonSessionGate(ctx context.Context, in *statev1.AbandonSessionGateRequest, opts ...grpc.CallOption) (*statev1.AbandonSessionGateResponse, error) {
	return &statev1.AbandonSessionGateResponse{}, nil
}

func (c *testSessionClient) GetSessionSpotlight(ctx context.Context, in *statev1.GetSessionSpotlightRequest, opts ...grpc.CallOption) (*statev1.GetSessionSpotlightResponse, error) {
	return &statev1.GetSessionSpotlightResponse{}, nil
}

func (c *testSessionClient) SetSessionSpotlight(ctx context.Context, in *statev1.SetSessionSpotlightRequest, opts ...grpc.CallOption) (*statev1.SetSessionSpotlightResponse, error) {
	return &statev1.SetSessionSpotlightResponse{}, nil
}

func (c *testSessionClient) ClearSessionSpotlight(ctx context.Context, in *statev1.ClearSessionSpotlightRequest, opts ...grpc.CallOption) (*statev1.ClearSessionSpotlightResponse, error) {
	return &statev1.ClearSessionSpotlightResponse{}, nil
}

type testCharacterClient struct {
	listResponse  *statev1.ListCharactersResponse
	sheetResponse *statev1.GetCharacterSheetResponse
	listErr       error
}

func (c *testCharacterClient) CreateCharacter(ctx context.Context, in *statev1.CreateCharacterRequest, opts ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
	return &statev1.CreateCharacterResponse{}, nil
}

func (c *testCharacterClient) UpdateCharacter(ctx context.Context, in *statev1.UpdateCharacterRequest, opts ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error) {
	return &statev1.UpdateCharacterResponse{}, nil
}

func (c *testCharacterClient) DeleteCharacter(ctx context.Context, in *statev1.DeleteCharacterRequest, opts ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error) {
	return &statev1.DeleteCharacterResponse{}, nil
}

func (c *testCharacterClient) ListCharacters(ctx context.Context, in *statev1.ListCharactersRequest, opts ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResponse != nil {
		return c.listResponse, nil
	}
	return &statev1.ListCharactersResponse{}, nil
}

func (c *testCharacterClient) SetDefaultControl(ctx context.Context, in *statev1.SetDefaultControlRequest, opts ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
	return &statev1.SetDefaultControlResponse{}, nil
}

func (c *testCharacterClient) GetCharacterSheet(ctx context.Context, in *statev1.GetCharacterSheetRequest, opts ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error) {
	if c.sheetResponse != nil {
		return c.sheetResponse, nil
	}
	return &statev1.GetCharacterSheetResponse{}, nil
}

func (c *testCharacterClient) PatchCharacterProfile(ctx context.Context, in *statev1.PatchCharacterProfileRequest, opts ...grpc.CallOption) (*statev1.PatchCharacterProfileResponse, error) {
	return &statev1.PatchCharacterProfileResponse{}, nil
}

type testEventClient struct {
	listResponse *statev1.ListEventsResponse
	listErr      error
}

func (c *testEventClient) AppendEvent(ctx context.Context, in *statev1.AppendEventRequest, opts ...grpc.CallOption) (*statev1.AppendEventResponse, error) {
	return &statev1.AppendEventResponse{}, nil
}

func (c *testEventClient) ListEvents(ctx context.Context, in *statev1.ListEventsRequest, opts ...grpc.CallOption) (*statev1.ListEventsResponse, error) {
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResponse != nil {
		return c.listResponse, nil
	}
	return &statev1.ListEventsResponse{}, nil
}

type testStatisticsClient struct {
	response *statev1.GetGameStatisticsResponse
}

func (c *testStatisticsClient) GetGameStatistics(ctx context.Context, in *statev1.GetGameStatisticsRequest, opts ...grpc.CallOption) (*statev1.GetGameStatisticsResponse, error) {
	if c.response != nil {
		return c.response, nil
	}
	return &statev1.GetGameStatisticsResponse{}, nil
}

// testFullClientProvider extends testClientProvider with all client types.
type testFullClientProvider struct {
	auth        authv1.AuthServiceClient
	campaign    statev1.CampaignServiceClient
	invite      statev1.InviteServiceClient
	participant statev1.ParticipantServiceClient
	system      statev1.SystemServiceClient
	session     statev1.SessionServiceClient
	character   statev1.CharacterServiceClient
	event       statev1.EventServiceClient
	statistics  statev1.StatisticsServiceClient
	snapshot    statev1.SnapshotServiceClient
}

func (p testFullClientProvider) CampaignClient() statev1.CampaignServiceClient   { return p.campaign }
func (p testFullClientProvider) SessionClient() statev1.SessionServiceClient     { return p.session }
func (p testFullClientProvider) CharacterClient() statev1.CharacterServiceClient { return p.character }
func (p testFullClientProvider) ParticipantClient() statev1.ParticipantServiceClient {
	return p.participant
}
func (p testFullClientProvider) InviteClient() statev1.InviteServiceClient     { return p.invite }
func (p testFullClientProvider) EventClient() statev1.EventServiceClient       { return p.event }
func (p testFullClientProvider) SnapshotClient() statev1.SnapshotServiceClient { return p.snapshot }
func (p testFullClientProvider) StatisticsClient() statev1.StatisticsServiceClient {
	return p.statistics
}
func (p testFullClientProvider) SystemClient() statev1.SystemServiceClient { return p.system }
func (p testFullClientProvider) AuthClient() authv1.AuthServiceClient      { return p.auth }

// --- Handler route tests ---

func TestUsersPage(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "<!doctype html>")
		assertContains(t, rec.Body.String(), "<h2>Users</h2>")
	})

	t.Run("htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "<h2>Users</h2>")
		assertNotContains(t, rec.Body.String(), "<!doctype html>")
	})

	t.Run("with message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users?message=hello+world", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "hello world")
	})

	t.Run("user_id redirects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users?user_id=u-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther && rec.Code != http.StatusMovedPermanently && rec.Code != http.StatusFound {
			t.Fatalf("expected redirect, got %d", rec.Code)
		}
	})
}

func TestUserLookup(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("empty user_id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/lookup", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("valid user_id redirects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/lookup?user_id=u-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther && rec.Code != http.StatusMovedPermanently && rec.Code != http.StatusFound {
			t.Fatalf("expected redirect, got %d", rec.Code)
		}
	})

	t.Run("post not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/lookup", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})
}

func TestCreateUser(t *testing.T) {
	authClient := &testAuthClient{user: &authv1.User{Id: "new-user", DisplayName: "Test"}}
	provider := testClientProvider{auth: authClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	t.Run("success", func(t *testing.T) {
		form := url.Values{"display_name": {"Test User"}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/create", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("expected 303, got %d", rec.Code)
		}
		location := rec.Header().Get("Location")
		if !strings.HasPrefix(location, "/users/new-user") {
			t.Errorf("expected redirect to /users/new-user, got %q", location)
		}
	})

	t.Run("empty display name", func(t *testing.T) {
		form := url.Values{"display_name": {""}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/create", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("get not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/create", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("nil auth client", func(t *testing.T) {
		noAuthHandler := &Handler{clientProvider: testClientProvider{}, impersonation: newImpersonationStore()}
		h := noAuthHandler.routes()
		form := url.Values{"display_name": {"Test"}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/create", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestUsersTable(t *testing.T) {
	authClient := &testAuthClient{}
	provider := testFullClientProvider{auth: authClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCampaignsTable(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testFullClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSessionsTable(t *testing.T) {
	sessionClient := &testSessionClient{
		listResponse: &statev1.ListSessionsResponse{},
	}
	provider := testFullClientProvider{session: sessionClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/sessions/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestParticipantsTable(t *testing.T) {
	participantClient := &testParticipantClient{}
	provider := testFullClientProvider{participant: participantClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/participants/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCharactersPage(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "<!doctype html>")
	})

	t.Run("htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertNotContains(t, rec.Body.String(), "<!doctype html>")
	})
}

func TestCharactersTable(t *testing.T) {
	characterClient := &testCharacterClient{
		listResponse: &statev1.ListCharactersResponse{},
	}
	participantClient := &testParticipantClient{}
	provider := testFullClientProvider{character: characterClient, participant: participantClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestEventLogPage(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/events", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), "<!doctype html>")
}

func TestEventLogTable(t *testing.T) {
	eventClient := &testEventClient{
		listResponse: &statev1.ListEventsResponse{},
	}
	provider := testFullClientProvider{event: eventClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/events/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestInvitesPage(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "<!doctype html>")
	})
}

func TestParticipantsPage(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/participants", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "<!doctype html>")
	})
}

func TestDashboardContent(t *testing.T) {
	statsClient := &testStatisticsClient{
		response: &statev1.GetGameStatisticsResponse{},
	}
	systemClient := &testSystemClient{
		listResponse: &statev1.ListGameSystemsResponse{},
	}
	authClient := &testAuthClient{}
	campaignClient := &testCampaignClient{}
	eventClient := &testEventClient{
		listResponse: &statev1.ListEventsResponse{},
	}
	provider := testFullClientProvider{
		statistics: statsClient,
		system:     systemClient,
		auth:       authClient,
		campaign:   campaignClient,
		event:      eventClient,
	}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/dashboard/content", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSystemDetail(t *testing.T) {
	systemClient := &testSystemClient{
		getResponse: &statev1.GetGameSystemResponse{
			System: &statev1.GameSystemInfo{
				Id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
				Name:    "Daggerheart",
				Version: "1.0.0",
			},
		},
	}
	provider := testFullClientProvider{system: systemClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/GAME_SYSTEM_DAGGERHEART", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Daggerheart")
	})

	t.Run("htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/GAME_SYSTEM_DAGGERHEART", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertNotContains(t, rec.Body.String(), "<!doctype html>")
	})
}

func TestCampaignDetailWithClients(t *testing.T) {
	campaignClient := &testCampaignClient{}
	provider := testFullClientProvider{campaign: campaignClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), "<!doctype html>")
}

func TestTrailingSlashRedirect(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/u-1/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", rec.Code)
	}
}

func TestSplitPathParts(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single", "a", 1},
		{"multiple", "a/b/c", 3},
		{"trailing slash", "a/b/", 2},
		{"whitespace only parts", "  / /a", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts := splitPathParts(tc.input)
			if len(parts) != tc.want {
				t.Errorf("splitPathParts(%q) = %d parts, want %d", tc.input, len(parts), tc.want)
			}
		})
	}
}

func TestIsHTMXRequest(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		if isHTMXRequest(nil) {
			t.Error("expected false for nil request")
		}
	})

	t.Run("no header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if isHTMXRequest(req) {
			t.Error("expected false for missing header")
		}
	})

	t.Run("htmx true", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("HX-Request", "true")
		if !isHTMXRequest(req) {
			t.Error("expected true for HX-Request: true")
		}
	})
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

func TestFormatImplementationStage(t *testing.T) {
	loc := i18n.Printer(language.English)
	tests := []struct {
		stage commonv1.GameSystemImplementationStage
		want  string
	}{
		{commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PLANNED, "Planned"},
		{commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL, "Partial"},
		{commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_COMPLETE, "Complete"},
		{commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_DEPRECATED, "Deprecated"},
		{commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_UNSPECIFIED, "Unspecified"},
	}
	for _, tc := range tests {
		got := formatImplementationStage(tc.stage, loc)
		if got != tc.want {
			t.Errorf("formatImplementationStage(%v) = %q, want %q", tc.stage, got, tc.want)
		}
	}
}

func TestFormatOperationalStatus(t *testing.T) {
	loc := i18n.Printer(language.English)
	tests := []struct {
		status commonv1.GameSystemOperationalStatus
		want   string
	}{
		{commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OFFLINE, "Offline"},
		{commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_DEGRADED, "Degraded"},
		{commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL, "Operational"},
		{commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_MAINTENANCE, "Maintenance"},
		{commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_UNSPECIFIED, "Unspecified"},
	}
	for _, tc := range tests {
		got := formatOperationalStatus(tc.status, loc)
		if got != tc.want {
			t.Errorf("formatOperationalStatus(%v) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestFormatAccessLevel(t *testing.T) {
	loc := i18n.Printer(language.English)
	tests := []struct {
		level commonv1.GameSystemAccessLevel
		want  string
	}{
		{commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_INTERNAL, "Internal"},
		{commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA, "Beta"},
		{commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_PUBLIC, "Public"},
		{commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_RETIRED, "Retired"},
		{commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_UNSPECIFIED, "Unspecified"},
	}
	for _, tc := range tests {
		got := formatAccessLevel(tc.level, loc)
		if got != tc.want {
			t.Errorf("formatAccessLevel(%v) = %q, want %q", tc.level, got, tc.want)
		}
	}
}

func TestFormatCharacterController(t *testing.T) {
	loc := i18n.Printer(language.English)
	names := map[string]string{"p1": "Alice"}

	t.Run("nil character", func(t *testing.T) {
		got := formatCharacterController(nil, names, loc)
		if got != "Unassigned" {
			t.Errorf("got %q, want Unassigned", got)
		}
	})

	t.Run("no participant ID", func(t *testing.T) {
		char := &statev1.Character{}
		got := formatCharacterController(char, names, loc)
		if got != "Unassigned" {
			t.Errorf("got %q, want Unassigned", got)
		}
	})

	t.Run("participant found", func(t *testing.T) {
		char := &statev1.Character{
			ParticipantId: wrapperspb.String("p1"),
		}
		got := formatCharacterController(char, names, loc)
		if got != "Alice" {
			t.Errorf("got %q, want Alice", got)
		}
	})

	t.Run("participant not found", func(t *testing.T) {
		char := &statev1.Character{
			ParticipantId: wrapperspb.String("unknown-p"),
		}
		got := formatCharacterController(char, names, loc)
		if got != "Unknown" {
			t.Errorf("got %q, want Unknown", got)
		}
	})
}

func TestBuildUserRows(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		rows := buildUserRows(nil)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("skips nil", func(t *testing.T) {
		rows := buildUserRows([]*authv1.User{nil, {Id: "u1", DisplayName: "Bob"}})
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].ID != "u1" {
			t.Errorf("ID = %q, want u1", rows[0].ID)
		}
		if rows[0].DisplayName != "Bob" {
			t.Errorf("DisplayName = %q, want Bob", rows[0].DisplayName)
		}
	})
}

func TestBuildCampaignRows(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("empty", func(t *testing.T) {
		rows := buildCampaignRows(nil, loc)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("skips nil", func(t *testing.T) {
		campaigns := []*statev1.Campaign{
			nil,
			{Id: "c1", Name: "Test", ParticipantCount: 3, CharacterCount: 2},
		}
		rows := buildCampaignRows(campaigns, loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].ID != "c1" {
			t.Errorf("ID = %q, want c1", rows[0].ID)
		}
		if rows[0].Name != "Test" {
			t.Errorf("Name = %q, want Test", rows[0].Name)
		}
		if rows[0].ParticipantCount != "3" {
			t.Errorf("ParticipantCount = %q, want 3", rows[0].ParticipantCount)
		}
	})
}

func TestBuildCampaignDetail(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("nil campaign", func(t *testing.T) {
		detail := buildCampaignDetail(nil, loc)
		if detail.ID != "" {
			t.Errorf("expected empty ID, got %q", detail.ID)
		}
	})

	t.Run("populated", func(t *testing.T) {
		campaign := &statev1.Campaign{
			Id:               "c1",
			Name:             "My Campaign",
			ParticipantCount: 5,
			CharacterCount:   3,
			ThemePrompt:      "A dark adventure",
		}
		detail := buildCampaignDetail(campaign, loc)
		if detail.ID != "c1" {
			t.Errorf("ID = %q, want c1", detail.ID)
		}
		if detail.ThemePrompt != "A dark adventure" {
			t.Errorf("ThemePrompt = %q, want full text", detail.ThemePrompt)
		}
	})
}

func TestBuildCampaignSessionRows(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("empty", func(t *testing.T) {
		rows := buildCampaignSessionRows(nil, loc)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("active session gets success badge", func(t *testing.T) {
		sessions := []*statev1.Session{
			{Id: "s1", CampaignId: "c1", Status: statev1.SessionStatus_SESSION_ACTIVE},
		}
		rows := buildCampaignSessionRows(sessions, loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].StatusBadge != "success" {
			t.Errorf("StatusBadge = %q, want success", rows[0].StatusBadge)
		}
	})

	t.Run("ended session gets secondary badge", func(t *testing.T) {
		sessions := []*statev1.Session{
			{Id: "s2", CampaignId: "c1", Status: statev1.SessionStatus_SESSION_ENDED},
		}
		rows := buildCampaignSessionRows(sessions, loc)
		if rows[0].StatusBadge != "secondary" {
			t.Errorf("StatusBadge = %q, want secondary", rows[0].StatusBadge)
		}
	})
}

func TestBuildCharacterRows(t *testing.T) {
	loc := i18n.Printer(language.English)
	names := map[string]string{"p1": "Alice"}

	t.Run("empty", func(t *testing.T) {
		rows := buildCharacterRows(nil, names, loc)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("populated with controller", func(t *testing.T) {
		chars := []*statev1.Character{
			{
				Id:            "ch1",
				CampaignId:    "c1",
				Name:          "Warrior",
				Kind:          statev1.CharacterKind_PC,
				ParticipantId: wrapperspb.String("p1"),
			},
		}
		rows := buildCharacterRows(chars, names, loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].Controller != "Alice" {
			t.Errorf("Controller = %q, want Alice", rows[0].Controller)
		}
		if rows[0].Kind != "PC" {
			t.Errorf("Kind = %q, want PC", rows[0].Kind)
		}
	})
}

func TestBuildCharacterSheet(t *testing.T) {
	loc := i18n.Printer(language.English)
	char := &statev1.Character{Id: "ch1", Name: "Warrior"}
	events := []templates.EventRow{{Seq: 1, Type: "test"}}

	sheet := buildCharacterSheet("c1", "My Campaign", char, events, "Alice", loc)
	if sheet.CampaignID != "c1" {
		t.Errorf("CampaignID = %q, want c1", sheet.CampaignID)
	}
	if sheet.CampaignName != "My Campaign" {
		t.Errorf("CampaignName = %q, want My Campaign", sheet.CampaignName)
	}
	if sheet.Controller != "Alice" {
		t.Errorf("Controller = %q, want Alice", sheet.Controller)
	}
	if len(sheet.RecentEvents) != 1 {
		t.Errorf("RecentEvents = %d, want 1", len(sheet.RecentEvents))
	}
}

func TestParseGameSystem(t *testing.T) {
	t.Run("daggerheart", func(t *testing.T) {
		system, ok := parseGameSystem("daggerheart")
		if !ok || system != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
			t.Errorf("got %v, %v", system, ok)
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, ok := parseGameSystem("unknown")
		if ok {
			t.Error("expected false for unknown system")
		}
	})
}

func TestParseGmMode(t *testing.T) {
	tests := []struct {
		input string
		want  statev1.GmMode
		ok    bool
	}{
		{"human", statev1.GmMode_HUMAN, true},
		{"ai", statev1.GmMode_AI, true},
		{"hybrid", statev1.GmMode_HYBRID, true},
		{"unknown", 0, false},
	}
	for _, tc := range tests {
		got, ok := parseGmMode(tc.input)
		if ok != tc.ok || got != tc.want {
			t.Errorf("parseGmMode(%q) = %v, %v; want %v, %v", tc.input, got, ok, tc.want, tc.ok)
		}
	}
}

func TestBuildInviteRows(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("nil invite skipped", func(t *testing.T) {
		rows := buildInviteRows([]*statev1.Invite{nil}, nil, nil, loc)
		if len(rows) != 0 {
			t.Fatalf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("participant and recipient resolved", func(t *testing.T) {
		invites := []*statev1.Invite{
			{
				Id:              "inv-1",
				CampaignId:      "camp-1",
				ParticipantId:   "part-1",
				RecipientUserId: "user-1",
				Status:          statev1.InviteStatus_PENDING,
				CreatedAt:       timestamppb.Now(),
			},
		}
		partNames := map[string]string{"part-1": "Alice"}
		recipNames := map[string]string{"user-1": "Bob"}
		rows := buildInviteRows(invites, partNames, recipNames, loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].Participant != "Alice" {
			t.Errorf("participant = %q, want Alice", rows[0].Participant)
		}
		if rows[0].Recipient != "Bob" {
			t.Errorf("recipient = %q, want Bob", rows[0].Recipient)
		}
	})

	t.Run("unknown participant and unassigned recipient", func(t *testing.T) {
		invites := []*statev1.Invite{
			{
				Id:            "inv-2",
				ParticipantId: "part-unknown",
				Status:        statev1.InviteStatus_CLAIMED,
			},
		}
		rows := buildInviteRows(invites, nil, nil, loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].Participant != "Unknown" {
			t.Errorf("participant = %q, want Unknown", rows[0].Participant)
		}
		if rows[0].Recipient != "Unassigned" {
			t.Errorf("recipient = %q, want Unassigned", rows[0].Recipient)
		}
	})

	t.Run("recipient ID fallback", func(t *testing.T) {
		invites := []*statev1.Invite{
			{
				Id:              "inv-3",
				RecipientUserId: "user-unknown",
				Status:          statev1.InviteStatus_REVOKED,
			},
		}
		rows := buildInviteRows(invites, nil, map[string]string{}, loc)
		if rows[0].Recipient != "user-unknown" {
			t.Errorf("recipient = %q, want user-unknown", rows[0].Recipient)
		}
	})
}

func TestBuildEventFilterExpression(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := buildEventFilterExpression(templates.EventFilterOptions{})
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("single field", func(t *testing.T) {
		got := buildEventFilterExpression(templates.EventFilterOptions{
			SessionID: "sess-1",
		})
		want := `session_id = "sess-1"`
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("all fields", func(t *testing.T) {
		got := buildEventFilterExpression(templates.EventFilterOptions{
			SessionID:  "s1",
			EventType:  "created",
			ActorType:  "user",
			EntityType: "character",
			StartDate:  "2024-01-01",
			EndDate:    "2024-12-31",
		})
		if !strings.Contains(got, `session_id = "s1"`) {
			t.Error("missing session_id filter")
		}
		if !strings.Contains(got, `type = "created"`) {
			t.Error("missing type filter")
		}
		if !strings.Contains(got, `actor_type = "user"`) {
			t.Error("missing actor_type filter")
		}
		if !strings.Contains(got, `entity_type = "character"`) {
			t.Error("missing entity_type filter")
		}
		if !strings.Contains(got, `ts >= timestamp("2024-01-01T00:00:00Z")`) {
			t.Error("missing start_date filter")
		}
		if !strings.Contains(got, `ts <= timestamp("2024-12-31T23:59:59Z")`) {
			t.Error("missing end_date filter")
		}
		if strings.Count(got, " AND ") != 5 {
			t.Errorf("expected 5 AND joins, got %d", strings.Count(got, " AND "))
		}
	})

	t.Run("escapes special characters", func(t *testing.T) {
		got := buildEventFilterExpression(templates.EventFilterOptions{
			EventType: `a"b\c`,
		})
		if !strings.Contains(got, `a\"b\\c`) {
			t.Errorf("got %q, expected escaped quotes and backslashes", got)
		}
	})
}

func TestHandlerClientAccessorsNilSafe(t *testing.T) {
	// nil handler
	var nilH *Handler
	if nilH.snapshotClient() != nil {
		t.Error("expected nil snapshot client from nil handler")
	}
	if nilH.characterClient() != nil {
		t.Error("expected nil character client from nil handler")
	}
	if nilH.inviteClient() != nil {
		t.Error("expected nil invite client from nil handler")
	}
	if nilH.eventClient() != nil {
		t.Error("expected nil event client from nil handler")
	}
	if nilH.statisticsClient() != nil {
		t.Error("expected nil statistics client from nil handler")
	}

	// handler with nil provider
	h := &Handler{}
	if h.snapshotClient() != nil {
		t.Error("expected nil snapshot client from nil provider")
	}
	if h.characterClient() != nil {
		t.Error("expected nil character client from nil provider")
	}
	if h.inviteClient() != nil {
		t.Error("expected nil invite client from nil provider")
	}
}

func TestLocaleFromTag(t *testing.T) {
	t.Run("valid tag", func(t *testing.T) {
		locale := localeFromTag("en-US")
		if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
			t.Error("expected specific locale for en-US")
		}
	})

	t.Run("invalid tag falls back to default", func(t *testing.T) {
		locale := localeFromTag("zzz")
		if locale == commonv1.Locale_LOCALE_UNSPECIFIED {
			t.Error("expected default locale, not unspecified")
		}
	})
}

func TestImpersonationStoreCleanup(t *testing.T) {
	store := newImpersonationStore()

	// Add an expired session.
	store.sessions["expired"] = impersonationSession{
		expiresAt: time.Now().Add(-time.Hour),
	}
	// Add a valid session.
	store.sessions["valid"] = impersonationSession{
		expiresAt: time.Now().Add(time.Hour),
	}

	// Force cleanup by setting lastCleanup far in the past.
	store.lastCleanup = time.Now().Add(-2 * impersonationCleanupInterval)
	store.cleanupLocked(time.Now())

	if _, ok := store.sessions["expired"]; ok {
		t.Error("expected expired session to be cleaned up")
	}
	if _, ok := store.sessions["valid"]; !ok {
		t.Error("expected valid session to remain")
	}
}

func TestImpersonationStoreSkipsRecentCleanup(t *testing.T) {
	store := newImpersonationStore()
	store.sessions["expired"] = impersonationSession{
		expiresAt: time.Now().Add(-time.Hour),
	}
	// Set lastCleanup to now so cleanup is skipped.
	store.lastCleanup = time.Now()
	store.cleanupLocked(time.Now())

	if _, ok := store.sessions["expired"]; !ok {
		t.Error("expected expired session to remain (cleanup skipped)")
	}
}

func TestImpersonationStoreGetExpired(t *testing.T) {
	store := newImpersonationStore()
	store.sessions["sess1"] = impersonationSession{
		expiresAt: time.Now().Add(-time.Hour),
	}
	_, ok := store.Get("sess1")
	if ok {
		t.Error("expected expired session to not be returned")
	}
}

func TestImpersonationStoreNilSafe(t *testing.T) {
	var store *impersonationStore
	if _, ok := store.Get("x"); ok {
		t.Error("expected false from nil store Get")
	}
	store.Set("x", impersonationSession{})
	store.Delete("x")
}

func TestLocalizerPersistsCookie(t *testing.T) {
	handler := &Handler{impersonation: newImpersonationStore()}
	req := httptest.NewRequest(http.MethodGet, "/?lang=pt-BR", nil)
	w := httptest.NewRecorder()
	loc, lang := handler.localizer(w, req)
	if loc == nil {
		t.Fatal("expected non-nil printer")
	}
	if lang == "" {
		t.Fatal("expected non-empty language tag")
	}
	// Should have set a cookie since lang param was present.
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == i18n.LangCookieName {
			found = true
		}
	}
	if !found {
		t.Error("expected language cookie to be set")
	}
}

func TestLocalizerNoCookie(t *testing.T) {
	handler := &Handler{impersonation: newImpersonationStore()}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	loc, _ := handler.localizer(w, req)
	if loc == nil {
		t.Fatal("expected non-nil printer")
	}
	// No lang param, so no cookie should be set.
	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == i18n.LangCookieName {
			t.Error("expected no language cookie without lang param")
		}
	}
}

// --- handleLogout tests ---

func TestHandleLogout(t *testing.T) {
	t.Run("GET not allowed", func(t *testing.T) {
		handler := NewHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/logout", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})

	t.Run("clears impersonation cookie", func(t *testing.T) {
		webHandler := &Handler{impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/logout", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		found := false
		for _, c := range rec.Result().Cookies() {
			if c.Name == impersonationCookieName {
				found = true
				if c.MaxAge != -1 {
					t.Errorf("expected MaxAge=-1, got %d", c.MaxAge)
				}
			}
		}
		if !found {
			t.Error("expected impersonation cookie to be cleared")
		}
	})

	t.Run("with user_id loads detail", func(t *testing.T) {
		authClient := &testAuthClient{
			user: &authv1.User{Id: "u-1", DisplayName: "Alice"},
		}
		provider := testFullClientProvider{auth: authClient}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		form := url.Values{"user_id": {"u-1"}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/logout", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Alice")
	})

	t.Run("deletes session from store", func(t *testing.T) {
		store := newImpersonationStore()
		store.Set("sess-1", impersonationSession{
			userID:    "u-1",
			expiresAt: time.Now().Add(time.Hour),
		})
		webHandler := &Handler{impersonation: store}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/logout", nil)
		req.Header.Set("Origin", "http://example.com")
		req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: "sess-1"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if _, ok := store.Get("sess-1"); ok {
			t.Error("expected session to be deleted")
		}
	})
}

// --- handleSystemsTable with data ---

func TestSystemsTableWithData(t *testing.T) {
	systemClient := &testSystemClient{
		listResponse: &statev1.ListGameSystemsResponse{
			Systems: []*statev1.GameSystemInfo{
				{
					Id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
					Name:    "Daggerheart",
					Version: "1.0.0",
				},
			},
		},
	}
	provider := testFullClientProvider{system: systemClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/table", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), "Daggerheart")
}

func TestSystemsTableEmpty(t *testing.T) {
	systemClient := &testSystemClient{
		listResponse: &statev1.ListGameSystemsResponse{},
	}
	provider := testFullClientProvider{system: systemClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/table", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// --- handleSystemRoutes edge cases ---

func TestSystemRoutesTrailingSlash(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/daggerheart/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", rec.Code)
	}
}

func TestSystemRoutesInvalidID(t *testing.T) {
	systemClient := &testSystemClient{}
	provider := testFullClientProvider{system: systemClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/invalid_system", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSystemRoutesDeepPath(t *testing.T) {
	handler := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/daggerheart/extra", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSystemDetailWithVersion(t *testing.T) {
	systemClient := &testSystemClient{
		getResponse: &statev1.GetGameSystemResponse{
			System: &statev1.GameSystemInfo{
				Id:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
				Name:    "Daggerheart",
				Version: "2.0.0",
			},
		},
	}
	provider := testFullClientProvider{system: systemClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/GAME_SYSTEM_DAGGERHEART?version=2.0.0", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), "2.0.0")
}

// --- handleInvitesTable ---

func TestInvitesTable(t *testing.T) {
	t.Run("with invites", func(t *testing.T) {
		inviteClient := &testInviteClient{
			listInvitesResponse: &statev1.ListInvitesResponse{
				Invites: []*statev1.Invite{
					{
						Id:              "inv-1",
						CampaignId:      "camp-1",
						ParticipantId:   "p-1",
						RecipientUserId: "u-1",
						Status:          statev1.InviteStatus_PENDING,
						CreatedAt:       timestamppb.Now(),
					},
				},
			},
		}
		participantClient := &testParticipantClient{
			participants: []*statev1.Participant{
				{Id: "p-1", CampaignId: "camp-1", DisplayName: "Alice"},
			},
		}
		authClient := &testAuthClient{
			user: &authv1.User{Id: "u-1", DisplayName: "Bob"},
		}
		provider := testFullClientProvider{
			invite:      inviteClient,
			participant: participantClient,
			auth:        authClient,
		}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites/table", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Alice")
		assertContains(t, rec.Body.String(), "Bob")
	})

	t.Run("empty invites", func(t *testing.T) {
		inviteClient := &testInviteClient{
			listInvitesResponse: &statev1.ListInvitesResponse{},
		}
		provider := testFullClientProvider{invite: inviteClient}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites/table", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("no participant client", func(t *testing.T) {
		inviteClient := &testInviteClient{
			listInvitesResponse: &statev1.ListInvitesResponse{
				Invites: []*statev1.Invite{
					{Id: "inv-1", ParticipantId: "p-1", Status: statev1.InviteStatus_CLAIMED},
				},
			},
		}
		provider := testFullClientProvider{invite: inviteClient}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites/table", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

// --- listPendingInvitesForUser ---

func TestListPendingInvitesForUser(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("empty user ID", func(t *testing.T) {
		h := &Handler{}
		rows, msg := h.listPendingInvitesForUser(context.Background(), "", loc)
		if rows != nil {
			t.Error("expected nil rows")
		}
		if msg == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("no invite client", func(t *testing.T) {
		h := &Handler{}
		rows, msg := h.listPendingInvitesForUser(context.Background(), "u-1", loc)
		if rows != nil {
			t.Error("expected nil rows")
		}
		if msg == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("with pending invites", func(t *testing.T) {
		inviteClient := &testInviteClient{
			pendingUserResponse: &statev1.ListPendingInvitesForUserResponse{
				Invites: []*statev1.PendingUserInvite{
					{
						Invite: &statev1.Invite{
							Id:         "inv-1",
							CampaignId: "camp-1",
							Status:     statev1.InviteStatus_PENDING,
							CreatedAt:  timestamppb.Now(),
						},
						Campaign:    &statev1.Campaign{Id: "camp-1", Name: "My Campaign"},
						Participant: &statev1.Participant{Id: "p-1", DisplayName: "Alice"},
					},
				},
			},
		}
		provider := testClientProvider{invite: inviteClient}
		h := &Handler{clientProvider: provider}
		rows, msg := h.listPendingInvitesForUser(context.Background(), "u-1", loc)
		if msg != "" {
			t.Fatalf("unexpected message: %q", msg)
		}
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].CampaignName != "My Campaign" {
			t.Errorf("CampaignName = %q, want My Campaign", rows[0].CampaignName)
		}
		if rows[0].Participant != "Alice" {
			t.Errorf("Participant = %q, want Alice", rows[0].Participant)
		}
	})

	t.Run("empty response", func(t *testing.T) {
		inviteClient := &testInviteClient{
			pendingUserResponse: &statev1.ListPendingInvitesForUserResponse{},
		}
		provider := testClientProvider{invite: inviteClient}
		h := &Handler{clientProvider: provider}
		rows, msg := h.listPendingInvitesForUser(context.Background(), "u-1", loc)
		if rows != nil {
			t.Error("expected nil rows for empty response")
		}
		if msg == "" {
			t.Error("expected non-empty message for empty response")
		}
	})

	t.Run("nil invite in pending", func(t *testing.T) {
		inviteClient := &testInviteClient{
			pendingUserResponse: &statev1.ListPendingInvitesForUserResponse{
				Invites: []*statev1.PendingUserInvite{
					{
						Campaign:    &statev1.Campaign{Id: "camp-1", Name: "Campaign"},
						Participant: &statev1.Participant{Id: "p-1", DisplayName: "Alice"},
					},
				},
			},
		}
		provider := testClientProvider{invite: inviteClient}
		h := &Handler{clientProvider: provider}
		rows, msg := h.listPendingInvitesForUser(context.Background(), "u-1", loc)
		if msg != "" {
			t.Fatalf("unexpected message: %q", msg)
		}
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		// With nil invite, ID and createdAt should be empty.
		if rows[0].ID != "" {
			t.Errorf("expected empty ID, got %q", rows[0].ID)
		}
	})

	t.Run("missing campaign name falls back to ID", func(t *testing.T) {
		inviteClient := &testInviteClient{
			pendingUserResponse: &statev1.ListPendingInvitesForUserResponse{
				Invites: []*statev1.PendingUserInvite{
					{
						Invite:      &statev1.Invite{Id: "inv-1", CampaignId: "camp-x"},
						Campaign:    &statev1.Campaign{Id: "camp-x"},
						Participant: &statev1.Participant{DisplayName: "Bob"},
					},
				},
			},
		}
		provider := testClientProvider{invite: inviteClient}
		h := &Handler{clientProvider: provider}
		rows, _ := h.listPendingInvitesForUser(context.Background(), "u-1", loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].CampaignName != "camp-x" {
			t.Errorf("CampaignName = %q, want camp-x (ID fallback)", rows[0].CampaignName)
		}
	})
}

// --- renderCharacterSheet ---

func TestRenderCharacterSheet(t *testing.T) {
	t.Run("no character client", func(t *testing.T) {
		handler := NewHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/ch-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", rec.Code)
		}
	})

	t.Run("nil character in response", func(t *testing.T) {
		characterClient := &testCharacterClient{
			sheetResponse: &statev1.GetCharacterSheetResponse{},
		}
		provider := testFullClientProvider{character: characterClient}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/ch-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("success with all clients", func(t *testing.T) {
		characterClient := &testCharacterClient{
			sheetResponse: &statev1.GetCharacterSheetResponse{
				Character: &statev1.Character{
					Id:            "ch-1",
					CampaignId:    "camp-1",
					Name:          "Warrior",
					Kind:          statev1.CharacterKind_PC,
					ParticipantId: wrapperspb.String("p-1"),
				},
			},
		}
		eventClient := &testEventClient{
			listResponse: &statev1.ListEventsResponse{
				Events: []*statev1.Event{
					{Seq: 1, Type: "character.created", CampaignId: "camp-1"},
				},
			},
		}
		participantClient := &testParticipantClient{
			participants: []*statev1.Participant{
				{Id: "p-1", DisplayName: "Alice"},
			},
		}
		campaignClient := &testCampaignClient{
			getCampaignResp: &statev1.GetCampaignResponse{
				Campaign: &statev1.Campaign{Id: "camp-1", Name: "Test Campaign"},
			},
		}
		provider := testFullClientProvider{
			character:   characterClient,
			event:       eventClient,
			participant: participantClient,
			campaign:    campaignClient,
		}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/ch-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Warrior")
		assertContains(t, rec.Body.String(), "Test Campaign")
	})

	t.Run("htmx request", func(t *testing.T) {
		characterClient := &testCharacterClient{
			sheetResponse: &statev1.GetCharacterSheetResponse{
				Character: &statev1.Character{
					Id:         "ch-1",
					CampaignId: "camp-1",
					Name:       "Rogue",
				},
			},
		}
		provider := testFullClientProvider{character: characterClient}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/ch-1", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertNotContains(t, rec.Body.String(), "<!doctype html>")
		assertContains(t, rec.Body.String(), "Rogue")
	})

	t.Run("no event client still renders", func(t *testing.T) {
		characterClient := &testCharacterClient{
			sheetResponse: &statev1.GetCharacterSheetResponse{
				Character: &statev1.Character{
					Id:   "ch-1",
					Name: "Mage",
				},
			},
		}
		provider := testFullClientProvider{character: characterClient}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/ch-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Mage")
	})
}

// --- resolveParticipantIDForUser ---

func TestResolveParticipantIDForUser(t *testing.T) {
	t.Run("empty user ID", func(t *testing.T) {
		h := &Handler{}
		id, err := h.resolveParticipantIDForUser(context.Background(), "camp-1", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "" {
			t.Errorf("expected empty, got %q", id)
		}
	})

	t.Run("no participant client", func(t *testing.T) {
		h := &Handler{}
		_, err := h.resolveParticipantIDForUser(context.Background(), "camp-1", "u-1")
		if err == nil {
			t.Fatal("expected error for nil client")
		}
	})

	t.Run("found participant", func(t *testing.T) {
		participantClient := &testParticipantClient{
			participants: []*statev1.Participant{
				{Id: "p-1", UserId: "u-1", DisplayName: "Alice"},
				{Id: "p-2", UserId: "u-2", DisplayName: "Bob"},
			},
		}
		provider := testClientProvider{participant: participantClient}
		h := &Handler{clientProvider: provider}
		id, err := h.resolveParticipantIDForUser(context.Background(), "camp-1", "u-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "p-1" {
			t.Errorf("expected p-1, got %q", id)
		}
	})

	t.Run("not found", func(t *testing.T) {
		participantClient := &testParticipantClient{
			participants: []*statev1.Participant{
				{Id: "p-1", UserId: "u-2", DisplayName: "Bob"},
			},
		}
		provider := testClientProvider{participant: participantClient}
		h := &Handler{clientProvider: provider}
		id, err := h.resolveParticipantIDForUser(context.Background(), "camp-1", "u-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "" {
			t.Errorf("expected empty, got %q", id)
		}
	})
}

// --- renderUserDetail ---

func TestRenderUserDetail(t *testing.T) {
	t.Run("htmx renders fragment", func(t *testing.T) {
		webHandler := &Handler{impersonation: newImpersonationStore()}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/u-1", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		view := templates.UserDetailPageView{}
		loc := i18n.Printer(language.English)
		pageCtx := templates.PageContext{}
		webHandler.renderUserDetail(rec, req, view, pageCtx, loc, "details")

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertNotContains(t, rec.Body.String(), "<!doctype html>")
	})

	t.Run("full page", func(t *testing.T) {
		webHandler := &Handler{impersonation: newImpersonationStore()}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/u-1", nil)
		rec := httptest.NewRecorder()
		view := templates.UserDetailPageView{}
		loc := i18n.Printer(language.English)
		pageCtx := templates.PageContext{}
		webHandler.renderUserDetail(rec, req, view, pageCtx, loc, "details")

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "<!doctype html>")
	})
}

// --- redirectToUserDetail ---

func TestRedirectToUserDetail(t *testing.T) {
	t.Run("empty user ID", func(t *testing.T) {
		h := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/lookup?user_id=", nil)
		rec := httptest.NewRecorder()
		h.redirectToUserDetail(rec, req, "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("htmx redirect", func(t *testing.T) {
		h := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/lookup?user_id=u-1", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		h.redirectToUserDetail(rec, req, "u-1")
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("expected 303, got %d", rec.Code)
		}
		if got := rec.Header().Get("HX-Redirect"); got != "/users/u-1" {
			t.Errorf("HX-Redirect = %q, want /users/u-1", got)
		}
	})

	t.Run("standard redirect", func(t *testing.T) {
		h := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/lookup?user_id=u-1", nil)
		rec := httptest.NewRecorder()
		h.redirectToUserDetail(rec, req, "u-1")
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("expected 303, got %d", rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/users/u-1" {
			t.Errorf("Location = %q, want /users/u-1", got)
		}
	})
}

// --- loadUserDetail ---

func TestLoadUserDetail(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("empty user ID", func(t *testing.T) {
		h := &Handler{}
		detail, msg := h.loadUserDetail(context.Background(), "", loc)
		if detail != nil {
			t.Error("expected nil detail")
		}
		if msg == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("no auth client", func(t *testing.T) {
		h := &Handler{}
		detail, msg := h.loadUserDetail(context.Background(), "u-1", loc)
		if detail != nil {
			t.Error("expected nil detail")
		}
		if msg == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("success", func(t *testing.T) {
		authClient := &testAuthClient{
			user: &authv1.User{Id: "u-1", DisplayName: "Alice"},
		}
		provider := testClientProvider{auth: authClient}
		h := &Handler{clientProvider: provider}
		detail, msg := h.loadUserDetail(context.Background(), "u-1", loc)
		if msg != "" {
			t.Fatalf("unexpected message: %q", msg)
		}
		if detail == nil {
			t.Fatal("expected non-nil detail")
		}
		if detail.ID != "u-1" {
			t.Errorf("ID = %q, want u-1", detail.ID)
		}
	})
}

// --- getCampaignName ---

func TestGetCampaignName(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("no campaign client", func(t *testing.T) {
		h := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		name := getCampaignName(h, req, "camp-1", loc)
		if name == "" {
			t.Error("expected fallback label")
		}
	})

	t.Run("with campaign", func(t *testing.T) {
		campaignClient := &testCampaignClient{
			getCampaignResp: &statev1.GetCampaignResponse{
				Campaign: &statev1.Campaign{Id: "camp-1", Name: "My Campaign"},
			},
		}
		provider := testClientProvider{campaign: campaignClient}
		h := &Handler{clientProvider: provider}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		name := getCampaignName(h, req, "camp-1", loc)
		if name != "My Campaign" {
			t.Errorf("expected My Campaign, got %q", name)
		}
	})

	t.Run("nil campaign in response", func(t *testing.T) {
		campaignClient := &testCampaignClient{
			getCampaignResp: &statev1.GetCampaignResponse{},
		}
		provider := testClientProvider{campaign: campaignClient}
		h := &Handler{clientProvider: provider}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		name := getCampaignName(h, req, "camp-1", loc)
		if name == "" {
			t.Error("expected fallback label")
		}
	})
}

// --- buildSystemDetail ---

func TestBuildSystemDetail(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("nil system", func(t *testing.T) {
		detail := buildSystemDetail(nil, loc)
		if detail.Name != "" {
			t.Errorf("expected empty name, got %q", detail.Name)
		}
	})

	t.Run("populated", func(t *testing.T) {
		system := &statev1.GameSystemInfo{
			Id:                  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			Name:                "Daggerheart",
			Version:             "1.0.0",
			ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_PARTIAL,
			OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
			AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_BETA,
			IsDefault:           true,
		}
		detail := buildSystemDetail(system, loc)
		if detail.Name != "Daggerheart" {
			t.Errorf("Name = %q, want Daggerheart", detail.Name)
		}
		if detail.Version != "1.0.0" {
			t.Errorf("Version = %q, want 1.0.0", detail.Version)
		}
		if !detail.IsDefault {
			t.Error("expected IsDefault to be true")
		}
	})
}

// --- buildUserDetail ---

func TestBuildUserDetail(t *testing.T) {
	t.Run("nil user", func(t *testing.T) {
		if got := buildUserDetail(nil); got != nil {
			t.Error("expected nil for nil user")
		}
	})

	t.Run("populated", func(t *testing.T) {
		u := &authv1.User{
			Id:          "u-1",
			DisplayName: "Alice",
			CreatedAt:   timestamppb.Now(),
			UpdatedAt:   timestamppb.Now(),
		}
		detail := buildUserDetail(u)
		if detail == nil {
			t.Fatal("expected non-nil detail")
		}
		if detail.ID != "u-1" {
			t.Errorf("ID = %q, want u-1", detail.ID)
		}
		if detail.DisplayName != "Alice" {
			t.Errorf("DisplayName = %q, want Alice", detail.DisplayName)
		}
	})
}

// --- buildSessionDetail ---

func TestBuildSessionDetail(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("nil session", func(t *testing.T) {
		detail := buildSessionDetail("camp-1", "Campaign", nil, 0, loc)
		if detail.ID != "" {
			t.Errorf("expected empty ID, got %q", detail.ID)
		}
	})

	t.Run("active session", func(t *testing.T) {
		session := &statev1.Session{
			Id:         "s-1",
			CampaignId: "camp-1",
			Name:       "Session 1",
			Status:     statev1.SessionStatus_SESSION_ACTIVE,
			StartedAt:  timestamppb.Now(),
		}
		detail := buildSessionDetail("camp-1", "My Campaign", session, 10, loc)
		if detail.ID != "s-1" {
			t.Errorf("ID = %q, want s-1", detail.ID)
		}
		if detail.StatusBadge != "success" {
			t.Errorf("StatusBadge = %q, want success", detail.StatusBadge)
		}
		if detail.EventCount != 10 {
			t.Errorf("EventCount = %d, want 10", detail.EventCount)
		}
	})

	t.Run("ended session", func(t *testing.T) {
		session := &statev1.Session{
			Id:        "s-2",
			Status:    statev1.SessionStatus_SESSION_ENDED,
			StartedAt: timestamppb.Now(),
			EndedAt:   timestamppb.Now(),
		}
		detail := buildSessionDetail("camp-1", "Campaign", session, 5, loc)
		if detail.StatusBadge != "secondary" {
			t.Errorf("StatusBadge = %q, want secondary", detail.StatusBadge)
		}
		if detail.EndedAt == "" {
			t.Error("expected non-empty EndedAt")
		}
	})
}

// --- buildEventRows ---

func TestBuildEventRows(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("empty", func(t *testing.T) {
		rows := buildEventRows(nil, loc)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("skips nil", func(t *testing.T) {
		events := []*statev1.Event{nil, {Seq: 1, Type: "campaign.created", CampaignId: "c-1"}}
		rows := buildEventRows(events, loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].CampaignID != "c-1" {
			t.Errorf("CampaignID = %q, want c-1", rows[0].CampaignID)
		}
	})

	t.Run("populates all fields", func(t *testing.T) {
		events := []*statev1.Event{
			{
				Seq:         5,
				Hash:        "abc",
				Type:        "character.created",
				CampaignId:  "c-1",
				SessionId:   "s-1",
				ActorType:   "participant",
				EntityType:  "character",
				EntityId:    "ch-1",
				Ts:          timestamppb.Now(),
				PayloadJson: []byte(`{"name":"Warrior"}`),
			},
		}
		rows := buildEventRows(events, loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		r := rows[0]
		if r.Seq != 5 {
			t.Errorf("Seq = %d, want 5", r.Seq)
		}
		if r.Hash != "abc" {
			t.Errorf("Hash = %q, want abc", r.Hash)
		}
		if r.SessionID != "s-1" {
			t.Errorf("SessionID = %q, want s-1", r.SessionID)
		}
		if r.EntityType != "character" {
			t.Errorf("EntityType = %q, want character", r.EntityType)
		}
	})
}

// --- parseEventFilters ---

func TestParseEventFilters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?session_id=s-1&event_type=created&actor_type=system&entity_type=character&start_date=2026-01-01&end_date=2026-12-31", nil)
	filters := parseEventFilters(req)
	if filters.SessionID != "s-1" {
		t.Errorf("SessionID = %q, want s-1", filters.SessionID)
	}
	if filters.EventType != "created" {
		t.Errorf("EventType = %q, want created", filters.EventType)
	}
	if filters.ActorType != "system" {
		t.Errorf("ActorType = %q, want system", filters.ActorType)
	}
	if filters.EntityType != "character" {
		t.Errorf("EntityType = %q, want character", filters.EntityType)
	}
	if filters.StartDate != "2026-01-01" {
		t.Errorf("StartDate = %q, want 2026-01-01", filters.StartDate)
	}
	if filters.EndDate != "2026-12-31" {
		t.Errorf("EndDate = %q, want 2026-12-31", filters.EndDate)
	}
}

// --- handleCharacterActivity ---

func TestCharacterActivityRoute(t *testing.T) {
	characterClient := &testCharacterClient{
		sheetResponse: &statev1.GetCharacterSheetResponse{
			Character: &statev1.Character{
				Id:         "ch-1",
				CampaignId: "camp-1",
				Name:       "Rogue",
			},
		},
	}
	provider := testFullClientProvider{character: characterClient}
	webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/ch-1/activity", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), "Rogue")
}

// --- handleSessionDetail ---

func TestSessionDetail(t *testing.T) {
	t.Run("no session client", func(t *testing.T) {
		handler := NewHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/sessions/s-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", rec.Code)
		}
	})

	t.Run("nil session in response", func(t *testing.T) {
		sessionClient := &testSessionClient{
			getResponse: &statev1.GetSessionResponse{},
		}
		provider := testFullClientProvider{session: sessionClient}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/sessions/s-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("success with all clients", func(t *testing.T) {
		sessionClient := &testSessionClient{
			getResponse: &statev1.GetSessionResponse{
				Session: &statev1.Session{
					Id:         "s-1",
					CampaignId: "camp-1",
					Name:       "Session 1",
					Status:     statev1.SessionStatus_SESSION_ACTIVE,
					StartedAt:  timestamppb.Now(),
				},
			},
		}
		eventClient := &testEventClient{
			listResponse: &statev1.ListEventsResponse{TotalSize: 42},
		}
		campaignClient := &testCampaignClient{
			getCampaignResp: &statev1.GetCampaignResponse{
				Campaign: &statev1.Campaign{Id: "camp-1", Name: "Test Campaign"},
			},
		}
		provider := testFullClientProvider{
			session:  sessionClient,
			event:    eventClient,
			campaign: campaignClient,
		}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/sessions/s-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "Session 1")
	})

	t.Run("htmx request", func(t *testing.T) {
		sessionClient := &testSessionClient{
			getResponse: &statev1.GetSessionResponse{
				Session: &statev1.Session{
					Id:        "s-1",
					Name:      "S1",
					Status:    statev1.SessionStatus_SESSION_ACTIVE,
					StartedAt: timestamppb.Now(),
				},
			},
		}
		provider := testFullClientProvider{session: sessionClient}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/sessions/s-1", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertNotContains(t, rec.Body.String(), "<!doctype html>")
	})
}

// --- handleSessionEvents ---

func TestSessionEventsRoute(t *testing.T) {
	t.Run("no event client", func(t *testing.T) {
		handler := NewHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/sessions/s-1/events", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "unavailable")
	})

	t.Run("with events", func(t *testing.T) {
		eventClient := &testEventClient{
			listResponse: &statev1.ListEventsResponse{
				Events: []*statev1.Event{
					{Seq: 1, Type: "session.started", CampaignId: "camp-1", SessionId: "s-1"},
				},
				TotalSize: 1,
			},
		}
		provider := testFullClientProvider{event: eventClient}
		webHandler := &Handler{clientProvider: provider, impersonation: newImpersonationStore()}
		handler := webHandler.routes()

		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/sessions/s-1/events", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

// --- getSessionName ---

func TestGetSessionName(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("no session client", func(t *testing.T) {
		h := &Handler{}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		name := getSessionName(h, req, "camp-1", "s-1", loc)
		if name == "" {
			t.Error("expected fallback label")
		}
	})

	t.Run("with session", func(t *testing.T) {
		sessionClient := &testSessionClient{
			getResponse: &statev1.GetSessionResponse{
				Session: &statev1.Session{Id: "s-1", Name: "Session 1"},
			},
		}
		provider := testFullClientProvider{session: sessionClient}
		h := &Handler{clientProvider: provider}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		name := getSessionName(h, req, "camp-1", "s-1", loc)
		if name != "Session 1" {
			t.Errorf("expected Session 1, got %q", name)
		}
	})

	t.Run("nil session in response", func(t *testing.T) {
		sessionClient := &testSessionClient{
			getResponse: &statev1.GetSessionResponse{},
		}
		provider := testFullClientProvider{session: sessionClient}
		h := &Handler{clientProvider: provider}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		name := getSessionName(h, req, "camp-1", "s-1", loc)
		if name == "" {
			t.Error("expected fallback label")
		}
	})
}

// --- populateUserInvites ---

func TestPopulateUserInvites(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("nil detail", func(t *testing.T) {
		h := &Handler{}
		// Should not panic.
		h.populateUserInvites(context.Background(), nil, loc)
	})

	t.Run("populates from client", func(t *testing.T) {
		inviteClient := &testInviteClient{
			pendingUserResponse: &statev1.ListPendingInvitesForUserResponse{},
		}
		provider := testClientProvider{invite: inviteClient}
		h := &Handler{clientProvider: provider}
		detail := &templates.UserDetail{ID: "u-1"}
		h.populateUserInvites(context.Background(), detail, loc)
		// With empty response, message should be set.
		if detail.PendingInvitesMessage == "" {
			t.Error("expected non-empty message for empty invites")
		}
	})
}

// --- populateUserInvitesIfImpersonating ---

func TestPopulateUserInvitesIfImpersonating(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("nil detail", func(t *testing.T) {
		h := &Handler{}
		h.populateUserInvitesIfImpersonating(context.Background(), nil, nil, loc)
	})

	t.Run("nil impersonation", func(t *testing.T) {
		h := &Handler{}
		detail := &templates.UserDetail{ID: "u-1"}
		h.populateUserInvitesIfImpersonating(context.Background(), detail, nil, loc)
		if detail.PendingInvitesMessage == "" {
			t.Error("expected require_impersonation message")
		}
	})

	t.Run("wrong user ID", func(t *testing.T) {
		h := &Handler{}
		detail := &templates.UserDetail{ID: "u-1"}
		imp := &templates.ImpersonationView{UserID: "u-2"}
		h.populateUserInvitesIfImpersonating(context.Background(), detail, imp, loc)
		if detail.PendingInvitesMessage == "" {
			t.Error("expected require_impersonation message for mismatched user")
		}
	})

	t.Run("matching user triggers fetch", func(t *testing.T) {
		inviteClient := &testInviteClient{
			pendingUserResponse: &statev1.ListPendingInvitesForUserResponse{},
		}
		provider := testClientProvider{invite: inviteClient}
		h := &Handler{clientProvider: provider}
		detail := &templates.UserDetail{ID: "u-1"}
		imp := &templates.ImpersonationView{UserID: "u-1"}
		h.populateUserInvitesIfImpersonating(context.Background(), detail, imp, loc)
		// Should have called listPendingInvitesForUser, which returns empty message.
		if detail.PendingInvitesMessage == "" {
			t.Error("expected message from pending invites fetch")
		}
	})
}

// --- buildParticipantRows ---

func TestBuildParticipantRows(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("empty", func(t *testing.T) {
		rows := buildParticipantRows(nil, loc)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("with data", func(t *testing.T) {
		participants := []*statev1.Participant{
			nil,
			{
				Id:             "p-1",
				CampaignId:     "camp-1",
				DisplayName:    "Alice",
				Role:           statev1.ParticipantRole_GM,
				CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
				Controller:     statev1.Controller_CONTROLLER_HUMAN,
				UserId:         "u-1",
				CreatedAt:      timestamppb.Now(),
			},
		}
		rows := buildParticipantRows(participants, loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].DisplayName != "Alice" {
			t.Errorf("DisplayName = %q, want Alice", rows[0].DisplayName)
		}
		if rows[0].ID != "p-1" {
			t.Errorf("ID = %q, want p-1", rows[0].ID)
		}
	})
}

// --- buildSystemRows ---

func TestBuildSystemRows(t *testing.T) {
	loc := i18n.Printer(language.English)

	t.Run("empty", func(t *testing.T) {
		rows := buildSystemRows(nil, loc)
		if len(rows) != 0 {
			t.Errorf("expected 0 rows, got %d", len(rows))
		}
	})

	t.Run("with data", func(t *testing.T) {
		systems := []*statev1.GameSystemInfo{
			nil,
			{
				Id:                commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
				Name:              "Daggerheart",
				Version:           "1.0.0",
				OperationalStatus: commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
			},
		}
		rows := buildSystemRows(systems, loc)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		if rows[0].Name != "Daggerheart" {
			t.Errorf("Name = %q, want Daggerheart", rows[0].Name)
		}
	})
}

// --- Error path tests for nil-client and RPC-failure branches ---

// TestUsersTableNilClient verifies the handler renders an error when authClient is nil.
func TestUsersTableNilClient(t *testing.T) {
	handler := NewHandler(nil) // nil provider → nil authClient
	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	assertContains(t, rec.Body.String(), "User service unavailable")
}

// TestUsersTableListError verifies the handler renders an error when ListUsers fails.
func TestUsersTableListError(t *testing.T) {
	authClient := &testAuthClient{listUsersErr: fmt.Errorf("connection refused")}
	handler := NewHandler(testClientProvider{auth: authClient})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "unavailable")
}

// TestUsersTableWithUsers verifies the handler renders user rows when users exist.
func TestUsersTableWithUsers(t *testing.T) {
	authClient := &testAuthClient{
		users: []*authv1.User{
			{Id: "user-1", DisplayName: "Alice"},
			{Id: "user-2", DisplayName: "Bob"},
		},
	}
	handler := NewHandler(testClientProvider{auth: authClient})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "Alice")
	assertContains(t, rec.Body.String(), "Bob")
}

// TestCampaignsTableNilClient verifies the handler renders an error when campaignClient is nil.
func TestCampaignsTableNilClient(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "Campaign service unavailable")
}

// TestCampaignsTableListError verifies the handler renders an error when ListCampaigns fails.
func TestCampaignsTableListError(t *testing.T) {
	campaignClient := &testCampaignClient{listErr: fmt.Errorf("unavailable")}
	handler := NewHandler(testClientProvider{campaign: campaignClient})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "unavailable")
}

// TestCampaignsTableWithData verifies campaign rows are rendered.
func TestCampaignsTableWithData(t *testing.T) {
	campaignClient := &testCampaignClient{
		listResponse: &statev1.ListCampaignsResponse{
			Campaigns: []*statev1.Campaign{
				{Id: "camp-1", Name: "Adventure", CreatedAt: timestamppb.Now()},
			},
		},
	}
	handler := NewHandler(testClientProvider{campaign: campaignClient})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "Adventure")
}

// TestSystemsTableListError verifies the handler renders an error when ListGameSystems fails.
func TestSystemsTableListError(t *testing.T) {
	systemClient := &testSystemClient{listErr: fmt.Errorf("unavailable")}
	handler := NewHandler(testClientProvider{system: systemClient})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "unavailable")
}

// TestSystemDetailNilClient verifies the handler renders an error when systemClient is nil.
func TestSystemDetailNilClient(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/daggerheart", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "System service unavailable")
}

// TestSystemDetailGetError verifies the handler renders an error when GetGameSystem fails.
func TestSystemDetailGetError(t *testing.T) {
	systemClient := &testSystemClient{getErr: fmt.Errorf("internal error")}
	handler := NewHandler(testClientProvider{system: systemClient})
	req := httptest.NewRequest(http.MethodGet, "http://example.com/systems/daggerheart", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "unavailable")
}

// TestCampaignCreateNilClient verifies the handler shows an error when campaignClient is nil.
func TestCampaignCreateNilClient(t *testing.T) {
	handler := NewHandler(testClientProvider{}) // no campaign client
	form := url.Values{
		"user_id": {"user-1"},
		"name":    {"Test Campaign"},
		"system":  {"daggerheart"},
		"gm_mode": {"human"},
	}
	req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "Campaign service unavailable")
}

// TestCampaignCreateRPCError verifies the handler renders an error when CreateCampaign fails.
func TestCampaignCreateRPCError(t *testing.T) {
	campaignClient := &testCampaignClient{createErr: fmt.Errorf("deadline exceeded")}
	handler := NewHandler(testClientProvider{campaign: campaignClient})
	form := url.Values{
		"user_id": {"user-1"},
		"name":    {"Test Campaign"},
		"system":  {"daggerheart"},
		"gm_mode": {"human"},
	}
	req := httptest.NewRequest(http.MethodPost, "http://example.com/campaigns/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "Unable to create campaign")
}

// TestCampaignCreateGetRendersPage verifies the GET path of campaign create.
func TestCampaignCreateGetRendersPage(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/create", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/create", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "<!doctype html>")
	})

	t.Run("with message", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/create?message=test+msg", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "test msg")
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "http://example.com/campaigns/create", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})
}

// TestSessionsTableListError verifies the handler renders an error when ListSessions fails.
func TestSessionsTableListError(t *testing.T) {
	sessionClient := &testSessionClient{listErr: fmt.Errorf("unavailable")}
	provider := testFullClientProvider{session: sessionClient}
	handler := NewHandler(provider)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/sessions/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "unavailable")
}

// TestCharactersTableErrorPaths verifies error branches in handleCharactersTable.
func TestCharactersTableErrorPaths(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := NewHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "Character service unavailable")
	})

	t.Run("list error", func(t *testing.T) {
		charClient := &testCharacterClient{listErr: fmt.Errorf("unavailable")}
		provider := testFullClientProvider{character: charClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "unavailable")
	})

	t.Run("with characters and participants", func(t *testing.T) {
		charClient := &testCharacterClient{
			listResponse: &statev1.ListCharactersResponse{
				Characters: []*statev1.Character{
					{Id: "char-1", Name: "Elara", ParticipantId: wrapperspb.String("part-1")},
				},
			},
		}
		partClient := &testParticipantClient{
			participants: []*statev1.Participant{
				{Id: "part-1", DisplayName: "Player1"},
			},
		}
		provider := testFullClientProvider{character: charClient, participant: partClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "Elara")
	})

	t.Run("participant list error", func(t *testing.T) {
		charClient := &testCharacterClient{
			listResponse: &statev1.ListCharactersResponse{
				Characters: []*statev1.Character{
					{Id: "char-1", Name: "Elara", ParticipantId: wrapperspb.String("part-1")},
				},
			},
		}
		partClient := &testParticipantClient{listErr: fmt.Errorf("unavailable")}
		provider := testFullClientProvider{character: charClient, participant: partClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/characters/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Should still render characters, just without participant names.
		assertContains(t, rec.Body.String(), "Elara")
	})
}

// TestInvitesTableErrorPaths verifies error branches in handleInvitesTable.
func TestInvitesTableErrorPaths(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		handler := NewHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "Invite service unavailable")
	})

	t.Run("list error", func(t *testing.T) {
		inviteClient := &testInviteClient{listInvitesErr: fmt.Errorf("unavailable")}
		provider := testFullClientProvider{invite: inviteClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "unavailable")
	})

	t.Run("with invites and recipient enrichment", func(t *testing.T) {
		inviteClient := &testInviteClient{
			listInvitesResponse: &statev1.ListInvitesResponse{
				Invites: []*statev1.Invite{
					{Id: "inv-1", RecipientUserId: "user-1", ParticipantId: "part-1",
						Status: statev1.InviteStatus_PENDING},
					nil, // nil invite in the list
					{Id: "inv-2", RecipientUserId: "", ParticipantId: "part-1",
						Status: statev1.InviteStatus_PENDING}, // no recipient
				},
			},
		}
		authClient := &testAuthClient{user: &authv1.User{Id: "user-1", DisplayName: "Alice"}}
		partClient := &testParticipantClient{
			participants: []*statev1.Participant{
				{Id: "part-1", DisplayName: "GM"},
			},
		}
		provider := testFullClientProvider{invite: inviteClient, auth: authClient, participant: partClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "Alice")
	})

	t.Run("auth client error for recipient", func(t *testing.T) {
		inviteClient := &testInviteClient{
			listInvitesResponse: &statev1.ListInvitesResponse{
				Invites: []*statev1.Invite{
					{Id: "inv-1", RecipientUserId: "user-1",
						Status: statev1.InviteStatus_PENDING},
				},
			},
		}
		authClient := &testAuthClient{getUserErr: fmt.Errorf("not found")}
		provider := testFullClientProvider{invite: inviteClient, auth: authClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Should still render without panic.
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

// TestInvitesListRendering verifies handleInvitesList renders both HTMX and full page paths.
func TestInvitesListRendering(t *testing.T) {
	handler := NewHandler(nil)

	t.Run("htmx", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertNotContains(t, rec.Body.String(), "<!doctype html>")
	})

	t.Run("full page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/invites", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		assertContains(t, rec.Body.String(), "<!doctype html>")
	})
}

// TestImpersonateUserNilAuthClient verifies error when authClient is nil.
func TestImpersonateUserNilAuthClient(t *testing.T) {
	webHandler := &Handler{clientProvider: testClientProvider{}, impersonation: newImpersonationStore()}
	handler := webHandler.routes()
	form := url.Values{"user_id": {"user-1"}}
	req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "User service unavailable")
}

// TestImpersonateUserGetUserError verifies error when GetUser RPC fails.
func TestImpersonateUserGetUserError(t *testing.T) {
	authClient := &testAuthClient{getUserErr: fmt.Errorf("not found")}
	webHandler := &Handler{clientProvider: testClientProvider{auth: authClient}, impersonation: newImpersonationStore()}
	handler := webHandler.routes()
	form := url.Values{"user_id": {"user-1"}}
	req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "not found")
}

// TestImpersonateUserSuccessDeletesOldSession verifies old session cleanup.
func TestImpersonateUserSuccessDeletesOldSession(t *testing.T) {
	authClient := &testAuthClient{user: &authv1.User{Id: "user-1", DisplayName: "Alice"}}
	webHandler := &Handler{clientProvider: testClientProvider{auth: authClient}, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	// Set up an existing impersonation session.
	webHandler.impersonation.Set("old-session", impersonationSession{userID: "user-old"})

	form := url.Values{"user_id": {"user-1"}}
	req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	req.AddCookie(&http.Cookie{Name: impersonationCookieName, Value: "old-session"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "Alice")
	// Old session should be deleted.
	if _, ok := webHandler.impersonation.Get("old-session"); ok {
		t.Error("expected old session to be deleted")
	}
}

// TestHandleImpersonateUserEmptyUserID verifies error for missing user_id.
func TestHandleImpersonateUserEmptyUserID(t *testing.T) {
	webHandler := &Handler{clientProvider: testClientProvider{auth: &testAuthClient{}}, impersonation: newImpersonationStore()}
	handler := webHandler.routes()
	form := url.Values{"user_id": {""}}
	req := httptest.NewRequest(http.MethodPost, "http://example.com/users/impersonate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Expect error message about user_id required.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// TestHandleImpersonateUserMethodNotAllowed verifies 405 for non-POST.
func TestHandleImpersonateUserMethodNotAllowed(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/impersonate", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// TestCurrentImpersonationNilStore verifies nil impersonation store returns nil.
func TestCurrentImpersonationNilStore(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	if h.currentImpersonation(req) != nil {
		t.Error("expected nil for handler with nil impersonation store")
	}
}

// TestSnapshotClientNilProvider verifies snapshotClient returns nil when no provider.
func TestSnapshotClientNilProvider(t *testing.T) {
	h := &Handler{}
	if h.snapshotClient() != nil {
		t.Error("expected nil for handler with nil clientProvider")
	}
}

// TestEventLogTableNilClient verifies the event log table renders an error when eventClient is nil.
func TestEventLogTableNilClient(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/events/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "Event service unavailable")
}

// TestParticipantsTableNilClient verifies participants table renders error when client is nil.
func TestParticipantsTableNilClient(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/participants/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "Participant service unavailable")
}

// TestCreateUserErrorPaths verifies error branches in handleCreateUser.
func TestCreateUserErrorPaths(t *testing.T) {
	t.Run("nil auth client", func(t *testing.T) {
		handler := NewHandler(testClientProvider{})
		form := url.Values{"display_name": {"Alice"}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/create", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "User service unavailable")
	})

	t.Run("empty display name", func(t *testing.T) {
		handler := NewHandler(testClientProvider{auth: &testAuthClient{}})
		form := url.Values{"display_name": {""}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/create", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("create user RPC error", func(t *testing.T) {
		authClient := &testAuthClient{createUserErr: fmt.Errorf("deadline exceeded")}
		handler := NewHandler(testClientProvider{auth: authClient})
		form := url.Values{"display_name": {"Alice"}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/create", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "Unable to create user")
	})

	t.Run("success htmx redirect", func(t *testing.T) {
		authClient := &testAuthClient{user: &authv1.User{Id: "new-user", DisplayName: "Alice"}}
		handler := NewHandler(testClientProvider{auth: authClient})
		form := url.Values{"display_name": {"Alice"}}
		req := httptest.NewRequest(http.MethodPost, "http://example.com/users/create", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Fatalf("expected 303, got %d", rec.Code)
		}
		if !strings.Contains(rec.Header().Get("HX-Redirect"), "/users/new-user") {
			t.Fatalf("expected HX-Redirect to /users/new-user, got %q", rec.Header().Get("HX-Redirect"))
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		handler := NewHandler(nil)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/users/create", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", rec.Code)
		}
	})
}

// TestCampaignDetailErrorPaths verifies error branches in handleCampaignDetail.
func TestCampaignDetailErrorPaths(t *testing.T) {
	t.Run("nil campaign client", func(t *testing.T) {
		handler := NewHandler(testClientProvider{})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "Campaign service unavailable")
	})

	t.Run("get campaign RPC error", func(t *testing.T) {
		campaignClient := &testCampaignClient{getCampaignErr: fmt.Errorf("unavailable")}
		handler := NewHandler(testClientProvider{campaign: campaignClient})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "unavailable")
	})

	t.Run("nil campaign in response", func(t *testing.T) {
		campaignClient := &testCampaignClient{getCampaignResp: &statev1.GetCampaignResponse{}}
		handler := NewHandler(testClientProvider{campaign: campaignClient})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "not found")
	})

	t.Run("success with campaign data", func(t *testing.T) {
		campaignClient := &testCampaignClient{
			getCampaignResp: &statev1.GetCampaignResponse{
				Campaign: &statev1.Campaign{
					Id: "camp-1", Name: "Test Campaign",
					CreatedAt: timestamppb.Now(),
				},
			},
		}
		handler := NewHandler(testClientProvider{campaign: campaignClient})
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "Test Campaign")
	})
}

// TestSessionsTableWithData verifies sessions table renders sessions.
func TestSessionsTableWithData(t *testing.T) {
	sessionClient := &testSessionClient{
		listResponse: &statev1.ListSessionsResponse{
			Sessions: []*statev1.Session{
				{Id: "session-1", Name: "Session 1", StartedAt: timestamppb.Now()},
			},
		},
	}
	provider := testFullClientProvider{session: sessionClient}
	handler := NewHandler(provider)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/sessions/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "Session 1")
}

// TestParticipantsTableErrorPaths verifies error branches in handleParticipantsTable.
func TestParticipantsTableErrorPaths(t *testing.T) {
	t.Run("list error", func(t *testing.T) {
		partClient := &testParticipantClient{listErr: fmt.Errorf("unavailable")}
		provider := testFullClientProvider{participant: partClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/participants/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "unavailable")
	})

	t.Run("with data", func(t *testing.T) {
		partClient := &testParticipantClient{
			participants: []*statev1.Participant{
				{Id: "part-1", DisplayName: "Player 1"},
			},
		}
		provider := testFullClientProvider{participant: partClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/participants/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "Player 1")
	})
}

// TestEventLogTableErrorPaths verifies error branches in handleEventLogTable.
func TestEventLogTableErrorPaths(t *testing.T) {
	t.Run("list error", func(t *testing.T) {
		eventClient := &testEventClient{listErr: fmt.Errorf("unavailable")}
		provider := testFullClientProvider{event: eventClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/events/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assertContains(t, rec.Body.String(), "unavailable")
	})

	t.Run("with events", func(t *testing.T) {
		eventClient := &testEventClient{
			listResponse: &statev1.ListEventsResponse{
				Events: []*statev1.Event{
					{Seq: 1, Type: "campaign.started", ActorType: "system", Ts: timestamppb.Now()},
				},
			},
		}
		provider := testFullClientProvider{event: eventClient, campaign: &testCampaignClient{}}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/events/table", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("with filters", func(t *testing.T) {
		eventClient := &testEventClient{}
		provider := testFullClientProvider{event: eventClient}
		handler := NewHandler(provider)
		req := httptest.NewRequest(http.MethodGet, "http://example.com/campaigns/camp-1/events/table?actor_type=system&event_type=campaign", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

// TestUserDetailWithMessage verifies the message query parameter in user detail.
func TestUserDetailWithMessage(t *testing.T) {
	authClient := &testAuthClient{user: &authv1.User{Id: "user-1", DisplayName: "Alice"}}
	webHandler := &Handler{clientProvider: testClientProvider{auth: authClient}, impersonation: newImpersonationStore()}
	handler := webHandler.routes()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/user-1?message=hello+world", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertContains(t, rec.Body.String(), "hello world")
}

// TestDashboardContentNilClients verifies dashboard renders with missing clients.
func TestDashboardContentNilClients(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/dashboard/content", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
