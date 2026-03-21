package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeAuthClient struct {
	authv1.AuthServiceClient
	lookupResp *authv1.LookupUserByUsernameResponse
}

func (c *fakeAuthClient) LookupUserByUsername(context.Context, *authv1.LookupUserByUsernameRequest, ...grpc.CallOption) (*authv1.LookupUserByUsernameResponse, error) {
	return c.lookupResp, nil
}

// testUnavailableConn implements grpc.ClientConnInterface and returns
// codes.Unavailable for every RPC, simulating a disconnected backend.
type testUnavailableConn struct{}

func (testUnavailableConn) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	return status.Error(codes.Unavailable, "test: service not connected")
}

func (testUnavailableConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Error(codes.Unavailable, "test: service not connected")
}

func TestUserHelpersBuildersAndFormatters(t *testing.T) {
	loc := i18nhttp.Printer(i18nhttp.Default())
	now := timestamppb.New(time.Date(2026, time.March, 2, 15, 4, 5, 0, time.UTC))

	rows := buildUserRows([]*authv1.User{
		nil,
		{Id: "user-1", Username: "alice", CreatedAt: now, UpdatedAt: now},
	})
	if len(rows) != 1 || rows[0].ID != "user-1" || rows[0].Username != "alice" {
		t.Fatalf("buildUserRows() = %#v", rows)
	}

	detail := buildUserDetail(&authv1.User{
		Id:        "user-1",
		Username:  "alice",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if detail == nil || detail.ID != "user-1" || detail.Username != "alice" {
		t.Fatalf("buildUserDetail() = %#v", detail)
	}
	if got := buildUserDetail(nil); got != nil {
		t.Fatalf("buildUserDetail(nil) = %#v", got)
	}

	label, variant := formatInviteStatus(invitev1.InviteStatus_PENDING, loc)
	if label != loc.Sprintf("label.invite_pending") || variant != "warning" {
		t.Fatalf("formatInviteStatus(pending) = (%q,%q)", label, variant)
	}
	label, variant = formatInviteStatus(invitev1.InviteStatus_INVITE_STATUS_UNSPECIFIED, loc)
	if label != loc.Sprintf("label.unspecified") || variant != "secondary" {
		t.Fatalf("formatInviteStatus(unspecified) = (%q,%q)", label, variant)
	}

	if got := eventview.FormatTimestamp(now); got != "2026-03-02 15:04:05" {
		t.Fatalf("eventview.FormatTimestamp() = %q", got)
	}
	if got := eventview.FormatTimestamp(nil); got != "" {
		t.Fatalf("eventview.FormatTimestamp(nil) = %q", got)
	}
}

func TestUserServiceUnavailableClients(t *testing.T) {
	var conn testUnavailableConn
	svc := &handlers{
		base:         modulehandler.NewBase(),
		authClient:   authv1.NewAuthServiceClient(conn),
		inviteClient: invitev1.NewInviteServiceClient(conn),
	}
	loc := i18nhttp.Printer(i18nhttp.Default())

	rec := httptest.NewRecorder()
	svc.HandleUsersPage(rec, httptest.NewRequest(http.MethodGet, "/app/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleUsersPage() status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.HandleUsersPage(rec, httptest.NewRequest(http.MethodGet, "/app/users?user_id=user-1", nil))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("HandleUsersPage(redirect) status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.HandleUsersTable(rec, httptest.NewRequest(http.MethodGet, "/app/users?fragment=rows", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleUsersTable(nil auth client) status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.HandleUserLookup(rec, httptest.NewRequest(http.MethodPost, "/app/users/lookup", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("HandleUserLookup(method) status = %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("HandleUserLookup(method) Allow = %q", allow)
	}

	rec = httptest.NewRecorder()
	svc.HandleUserLookup(rec, httptest.NewRequest(http.MethodGet, "/app/users/lookup", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleUserLookup(empty id) status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.HandleUserDetail(rec, httptest.NewRequest(http.MethodGet, "/app/users/user-1", nil), "user-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleUserDetail(nil auth client) status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.HandleUserInvites(rec, httptest.NewRequest(http.MethodGet, "/app/users/user-1/invites", nil), "user-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleUserInvites(nil clients) status = %d", rec.Code)
	}

	stubReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rows, message := svc.listPendingInvitesForUser(stubReq, stubReq.Context(), "", loc)
	if rows != nil || message == "" {
		t.Fatalf("listPendingInvitesForUser(empty id) = (%#v,%q)", rows, message)
	}
	rows, message = svc.listPendingInvitesForUser(stubReq, stubReq.Context(), "user-1", loc)
	if rows != nil || message == "" {
		t.Fatalf("listPendingInvitesForUser(nil client) = (%#v,%q)", rows, message)
	}
}

func TestHandleUserLookupByUsernameRedirectsToResolvedUser(t *testing.T) {
	svc := &handlers{
		base: modulehandler.NewBase(),
		authClient: &fakeAuthClient{
			lookupResp: &authv1.LookupUserByUsernameResponse{
				User: &authv1.User{Id: "user-1", Username: "alice"},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/app/users/lookup?username=alice", nil)
	rec := httptest.NewRecorder()
	svc.HandleUserLookup(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("HandleUserLookup(username) status = %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/app/users/user-1" {
		t.Fatalf("HandleUserLookup(username) location = %q", got)
	}
}

func TestHandleUserLookupByUsernameRedirectsBackWhenMissing(t *testing.T) {
	svc := &handlers{
		base:       modulehandler.NewBase(),
		authClient: &fakeAuthClient{},
	}

	req := httptest.NewRequest(http.MethodGet, "/app/users/lookup?username=missing", nil)
	rec := httptest.NewRecorder()
	svc.HandleUserLookup(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("HandleUserLookup(missing username) status = %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if location == "" || location == "/app/users/user-1" {
		t.Fatalf("HandleUserLookup(missing username) location = %q", location)
	}
}
