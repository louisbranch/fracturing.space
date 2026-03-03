package users

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUserHelpersBuildersAndFormatters(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	now := timestamppb.New(time.Date(2026, time.March, 2, 15, 4, 5, 0, time.UTC))

	rows := buildUserRows([]*authv1.User{
		nil,
		{Id: "user-1", Email: "alice@example.com", CreatedAt: now, UpdatedAt: now},
	})
	if len(rows) != 1 || rows[0].ID != "user-1" || rows[0].Email != "alice@example.com" {
		t.Fatalf("buildUserRows() = %#v", rows)
	}

	detail := buildUserDetail(&authv1.User{
		Id:        "user-1",
		Email:     "alice@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if detail == nil || detail.ID != "user-1" {
		t.Fatalf("buildUserDetail() = %#v", detail)
	}
	if got := buildUserDetail(nil); got != nil {
		t.Fatalf("buildUserDetail(nil) = %#v", got)
	}

	emailRows := buildUserEmailRows([]*authv1.UserEmail{
		nil,
		{Email: "alice@example.com", CreatedAt: now, UpdatedAt: now},
		{Email: "alice+verified@example.com", CreatedAt: now, UpdatedAt: now, VerifiedAt: now},
	}, loc)
	if len(emailRows) != 2 || emailRows[0].VerifiedAt != "-" || emailRows[1].VerifiedAt == "-" {
		t.Fatalf("buildUserEmailRows() = %#v", emailRows)
	}
	if empty := buildUserEmailRows(nil, loc); empty != nil {
		t.Fatalf("buildUserEmailRows(nil) = %#v", empty)
	}

	label, variant := formatInviteStatus(statev1.InviteStatus_PENDING, loc)
	if label != loc.Sprintf("label.invite_pending") || variant != "warning" {
		t.Fatalf("formatInviteStatus(pending) = (%q,%q)", label, variant)
	}
	label, variant = formatInviteStatus(statev1.InviteStatus_INVITE_STATUS_UNSPECIFIED, loc)
	if label != loc.Sprintf("label.unspecified") || variant != "secondary" {
		t.Fatalf("formatInviteStatus(unspecified) = (%q,%q)", label, variant)
	}

	if got := formatTimestamp(now); got != "2026-03-02 15:04:05" {
		t.Fatalf("formatTimestamp() = %q", got)
	}
	if got := formatTimestamp(nil); got != "" {
		t.Fatalf("formatTimestamp(nil) = %q", got)
	}
}

func TestUserServiceNilClients(t *testing.T) {
	svc := &service{base: modulehandler.NewBase(nil)}
	loc := i18n.Printer(i18n.Default())

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

	rows, message := svc.listPendingInvitesForUser(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "", loc)
	if rows != nil || message == "" {
		t.Fatalf("listPendingInvitesForUser(empty id) = (%#v,%q)", rows, message)
	}
	rows, message = svc.listPendingInvitesForUser(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "user-1", loc)
	if rows != nil || message == "" {
		t.Fatalf("listPendingInvitesForUser(nil client) = (%#v,%q)", rows, message)
	}
}
