package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type extraAuthClient struct {
	authv1.AuthServiceClient
	listUsersResp *authv1.ListUsersResponse
	listUsersErr  error
	listUsersReqs []*authv1.ListUsersRequest

	getUserResp *authv1.GetUserResponse
	getUserErr  error

	lookupResp *authv1.LookupUserByUsernameResponse
	lookupErr  error
}

func (c *extraAuthClient) ListUsers(_ context.Context, req *authv1.ListUsersRequest, _ ...grpc.CallOption) (*authv1.ListUsersResponse, error) {
	c.listUsersReqs = append(c.listUsersReqs, &authv1.ListUsersRequest{
		PageSize:  req.GetPageSize(),
		PageToken: req.GetPageToken(),
	})
	return c.listUsersResp, c.listUsersErr
}

func (c *extraAuthClient) GetUser(context.Context, *authv1.GetUserRequest, ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	return c.getUserResp, c.getUserErr
}

func (c *extraAuthClient) LookupUserByUsername(context.Context, *authv1.LookupUserByUsernameRequest, ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	return c.lookupResp, c.lookupErr
}

type extraInviteClient struct {
	invitev1.InviteServiceClient
	responses []*invitev1.ListPendingInvitesForUserResponse
	err       error
	requests  []*invitev1.ListPendingInvitesForUserRequest
}

func (c *extraInviteClient) ListPendingInvitesForUser(_ context.Context, req *invitev1.ListPendingInvitesForUserRequest, _ ...grpc.CallOption) (*invitev1.ListPendingInvitesForUserResponse, error) {
	c.requests = append(c.requests, &invitev1.ListPendingInvitesForUserRequest{
		PageSize:  req.GetPageSize(),
		PageToken: req.GetPageToken(),
	})
	if c.err != nil {
		return nil, c.err
	}
	if len(c.responses) == 0 {
		return &invitev1.ListPendingInvitesForUserResponse{}, nil
	}
	resp := c.responses[0]
	c.responses = c.responses[1:]
	return resp, nil
}

func TestUsersModuleIDAndNewHandlers(t *testing.T) {
	t.Parallel()

	if got := New(nil).ID(); got != "users" {
		t.Fatalf("Module.ID() = %q, want %q", got, "users")
	}

	h := NewHandlers(modulehandler.NewBase(), nil, nil)
	if _, ok := h.(*handlers); !ok {
		t.Fatalf("NewHandlers() type = %T, want *handlers", h)
	}
}

func TestUsersTableAndRouteHelpers(t *testing.T) {
	t.Parallel()

	base := modulehandler.NewBase()
	authClient := &extraAuthClient{}
	svc := &handlers{base: base, authClient: authClient}

	req := httptest.NewRequest(http.MethodGet, "/app/users?fragment=rows", nil)
	rec := httptest.NewRecorder()
	svc.HandleUsersTable(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleUsersTable(empty) status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No users yet.") {
		t.Fatalf("HandleUsersTable(empty) body = %q, want empty-users message", rec.Body.String())
	}
	if len(authClient.listUsersReqs) != 1 || authClient.listUsersReqs[0].GetPageSize() != 50 {
		t.Fatalf("ListUsers requests = %#v, want one page-size-50 request", authClient.listUsersReqs)
	}

	authClient.listUsersResp = &authv1.ListUsersResponse{
		Users: []*authv1.User{
			nil,
			{Id: "user-7"},
		},
	}
	rec = httptest.NewRecorder()
	svc.HandleUsersTable(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleUsersTable(success) status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "user-7") {
		t.Fatalf("HandleUsersTable(success) body = %q, want user row", rec.Body.String())
	}

	if wantsRowsFragment(nil) {
		t.Fatal("wantsRowsFragment(nil) = true, want false")
	}
	if wantsRowsFragment(httptest.NewRequest(http.MethodGet, "/app/users?fragment=%20cards%20", nil)) {
		t.Fatal("wantsRowsFragment(non-rows) = true, want false")
	}
	if !wantsRowsFragment(httptest.NewRequest(http.MethodGet, "/app/users?fragment=ROWS", nil)) {
		t.Fatal("wantsRowsFragment(rows) = false, want true")
	}
}

func TestUsersDetailAndInvitePaginationBranches(t *testing.T) {
	t.Parallel()

	loc := i18nhttp.Printer(i18nhttp.Default())
	now := timestamppb.New(time.Date(2026, time.March, 28, 12, 0, 0, 0, time.UTC))
	inviteClient := &extraInviteClient{
		responses: []*invitev1.ListPendingInvitesForUserResponse{
			{
				Invites: []*invitev1.PendingInviteForUserEntry{
					nil,
					{},
					{
						Invite: &invitev1.Invite{
							Id:         "invite-2",
							CampaignId: "camp-2",
							Status:     invitev1.InviteStatus_CLAIMED,
							CreatedAt:  now,
						},
						Campaign:    &invitev1.InviteCampaignSummary{},
						Participant: &invitev1.InviteParticipantSummary{},
					},
				},
				NextPageToken: "next-page",
			},
			{
				Invites: []*invitev1.PendingInviteForUserEntry{
					{
						Invite: &invitev1.Invite{
							Id:         "invite-3",
							CampaignId: "camp-3",
							Status:     invitev1.InviteStatus_REVOKED,
						},
						Campaign: &invitev1.InviteCampaignSummary{
							Id:   "camp-3",
							Name: "Stars Above",
						},
						Participant: &invitev1.InviteParticipantSummary{Name: "Rogue"},
					},
				},
			},
		},
	}
	svc := &handlers{base: modulehandler.NewBase(), inviteClient: inviteClient}
	req := httptest.NewRequest(http.MethodGet, "/app/users/user-1/invites", nil)

	rows, message := svc.listPendingInvitesForUser(req, req.Context(), "user-1", loc)
	if message != "" {
		t.Fatalf("listPendingInvitesForUser() message = %q, want empty", message)
	}
	if len(rows) != 3 {
		t.Fatalf("listPendingInvitesForUser() rows = %#v, want 3 rows", rows)
	}
	if rows[0].CampaignName != loc.Sprintf("label.unknown") || rows[0].Participant != loc.Sprintf("label.unknown") {
		t.Fatalf("rows[0] = %#v, want unknown fallbacks", rows[0])
	}
	if rows[0].StatusVariant != "secondary" || rows[0].Status != loc.Sprintf("label.unspecified") {
		t.Fatalf("rows[0] status = %#v", rows[0])
	}
	if rows[1].CampaignID != "camp-2" || rows[1].CampaignName != "camp-2" {
		t.Fatalf("rows[1] = %#v, want campaign-id fallback", rows[1])
	}
	if rows[1].StatusVariant != "success" || rows[1].Status != loc.Sprintf("label.invite_claimed") || rows[1].CreatedAt == "" {
		t.Fatalf("rows[1] = %#v, want claimed status and timestamp", rows[1])
	}
	if rows[2].CampaignName != "Stars Above" || rows[2].Participant != "Rogue" {
		t.Fatalf("rows[2] = %#v, want campaign and participant names", rows[2])
	}
	if rows[2].StatusVariant != "error" || rows[2].Status != loc.Sprintf("label.invite_revoked") {
		t.Fatalf("rows[2] = %#v, want revoked status", rows[2])
	}
	if len(inviteClient.requests) != 2 || inviteClient.requests[0].GetPageSize() != inviteListPageSize || inviteClient.requests[1].GetPageToken() != "next-page" {
		t.Fatalf("ListPendingInvitesForUser requests = %#v", inviteClient.requests)
	}

	detail := &templates.UserDetail{ID: "user-1"}
	svc.populateUserInvites(req, req.Context(), detail, loc)
	if len(detail.PendingInvites) != 0 || detail.PendingInvitesMessage != loc.Sprintf("users.invites.empty") {
		t.Fatalf("populateUserInvites() detail = %#v, want empty-invites message on exhausted fake client", detail)
	}

	svc.populateUserInvites(req, req.Context(), nil, loc)
}

func TestUsersRedirectAndLoadDetailBranches(t *testing.T) {
	t.Parallel()

	loc := i18nhttp.Printer(i18nhttp.Default())
	authClient := &extraAuthClient{
		lookupErr:  grpcstatus.Error(codes.Unavailable, "lookup unavailable"),
		getUserErr: grpcstatus.Error(codes.NotFound, "missing user"),
	}
	svc := &handlers{base: modulehandler.NewBase(), authClient: authClient}

	rec := httptest.NewRecorder()
	svc.redirectToUserDetail(rec, httptest.NewRequest(http.MethodGet, "/app/users/", nil), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("redirectToUserDetail(empty) status = %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/users/user-1", nil)
	req.Header.Set("HX-Request", "true")
	rec = httptest.NewRecorder()
	svc.redirectToUserDetail(rec, req, "user-1")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("HX-Redirect") != "/app/users/user-1" {
		t.Fatalf("redirectToUserDetail(htmx) = status %d headers %#v", rec.Code, rec.Header())
	}

	rec = httptest.NewRecorder()
	svc.redirectToUsernameDetail(rec, req, "", loc)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("redirectToUsernameDetail(empty) status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/users?username=missing", nil)
	req.Header.Set("HX-Request", "true")
	rec = httptest.NewRecorder()
	svc.redirectToUsernameDetail(rec, req, "missing", loc)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("redirectToUsernameDetail(missing) status = %d", rec.Code)
	}
	if location := rec.Header().Get("HX-Redirect"); !strings.Contains(location, "message=") {
		t.Fatalf("redirectToUsernameDetail(missing) location = %q, want error message query", location)
	}

	if detail, message := svc.loadUserDetail(req, req.Context(), "", loc); detail != nil || message != loc.Sprintf("error.user_id_required") {
		t.Fatalf("loadUserDetail(empty) = (%#v, %q)", detail, message)
	}
	if detail, message := svc.loadUserDetail(req, req.Context(), "user-404", loc); detail != nil || message != loc.Sprintf("error.user_not_found") {
		t.Fatalf("loadUserDetail(missing) = (%#v, %q)", detail, message)
	}
}
