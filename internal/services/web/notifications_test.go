package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeWebNotificationClient struct {
	listResp        *notificationsv1.ListNotificationsResponse
	listErr         error
	listReq         *notificationsv1.ListNotificationsRequest
	listMD          metadata.MD
	listCalls       int
	listHasDeadline bool
	listDeadline    time.Time

	unreadResp        *notificationsv1.GetUnreadNotificationStatusResponse
	unreadErr         error
	unreadReq         *notificationsv1.GetUnreadNotificationStatusRequest
	unreadMD          metadata.MD
	unreadCalls       int
	unreadHasDeadline bool
	unreadDeadline    time.Time

	markResp *notificationsv1.MarkNotificationReadResponse
	markErr  error
	markReq  *notificationsv1.MarkNotificationReadRequest
	markMD   metadata.MD
}

func (f *fakeWebNotificationClient) CreateNotificationIntent(context.Context, *notificationsv1.CreateNotificationIntentRequest, ...grpc.CallOption) (*notificationsv1.CreateNotificationIntentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebNotificationClient) ListNotifications(ctx context.Context, req *notificationsv1.ListNotificationsRequest, _ ...grpc.CallOption) (*notificationsv1.ListNotificationsResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.listMD = md
	f.listReq = req
	f.listCalls++
	if deadline, ok := ctx.Deadline(); ok {
		f.listHasDeadline = true
		f.listDeadline = deadline
	}
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &notificationsv1.ListNotificationsResponse{}, nil
}

func (f *fakeWebNotificationClient) GetUnreadNotificationStatus(ctx context.Context, req *notificationsv1.GetUnreadNotificationStatusRequest, _ ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.unreadMD = md
	f.unreadReq = req
	f.unreadCalls++
	if deadline, ok := ctx.Deadline(); ok {
		f.unreadHasDeadline = true
		f.unreadDeadline = deadline
	}
	if f.unreadErr != nil {
		return nil, f.unreadErr
	}
	if f.unreadResp != nil {
		return f.unreadResp, nil
	}
	return &notificationsv1.GetUnreadNotificationStatusResponse{}, nil
}

func (f *fakeWebNotificationClient) MarkNotificationRead(ctx context.Context, req *notificationsv1.MarkNotificationReadRequest, _ ...grpc.CallOption) (*notificationsv1.MarkNotificationReadResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.markMD = md
	f.markReq = req
	if f.markErr != nil {
		return nil, f.markErr
	}
	if f.markResp != nil {
		return f.markResp, nil
	}
	return &notificationsv1.MarkNotificationReadResponse{}, nil
}

func TestAppNotificationsPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppNotificationsPageRendersNewestFirstWithUnreadStateAndRelativeTime(t *testing.T) {
	now := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)
	oldCreatedAt := now.Add(-2 * time.Hour)
	newCreatedAt := now.Add(-10 * time.Minute)
	readAt := now.Add(-30 * time.Minute)

	fakeClient := &fakeWebNotificationClient{
		listResp: &notificationsv1.ListNotificationsResponse{
			Notifications: []*notificationsv1.Notification{
				{
					Id:        "notif-old",
					Topic:     "Welcome to Fracturing Space",
					Source:    "onboarding",
					CreatedAt: timestamppb.New(oldCreatedAt),
					ReadAt:    timestamppb.New(readAt),
				},
				{
					Id:        "notif-new",
					Topic:     "Campaign invite accepted",
					Source:    "campaign",
					CreatedAt: timestamppb.New(newCreatedAt),
				},
			},
		},
	}

	h := &handler{
		config:             Config{AuthBaseURL: "http://auth.local"},
		sessions:           newSessionStore(),
		pendingFlows:       newPendingFlowStore(),
		notificationClient: fakeClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	originalNow := notificationsNow
	notificationsNow = func() time.Time { return now }
	t.Cleanup(func() {
		notificationsNow = originalNow
	})

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppNotifications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if fakeClient.listReq == nil {
		t.Fatal("expected ListNotifications call")
	}
	if got := fakeClient.listReq.GetPageSize(); got != 50 {
		t.Fatalf("page_size = %d, want %d", got, 50)
	}
	userIDs := fakeClient.listMD.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-1" {
		t.Fatalf("metadata %s = %v, want [user-1]", grpcmeta.UserIDHeader, userIDs)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Campaign invite accepted") {
		t.Fatalf("expected newest notification topic in response")
	}
	if !strings.Contains(body, "Welcome to Fracturing Space") {
		t.Fatalf("expected oldest notification topic in response")
	}
	newIdx := strings.Index(body, "Campaign invite accepted")
	oldIdx := strings.Index(body, "Welcome to Fracturing Space")
	if newIdx < 0 || oldIdx < 0 {
		t.Fatalf("expected both notification topics in rendered output")
	}
	if newIdx > oldIdx {
		t.Fatalf("expected newer notification to render first")
	}
	if !strings.Contains(body, `data-notification-id="notif-new"`) {
		t.Fatalf("expected notification id marker for newest message")
	}
	if !strings.Contains(body, `form method="POST" action="/notifications/notif-new"`) {
		t.Fatalf("expected notification row submit form for mark-read action")
	}
	if !strings.Contains(body, `type="submit"`) {
		t.Fatalf("expected submit button for notification row")
	}
	if strings.Contains(body, `href="/notifications/notif-new"`) {
		t.Fatalf("expected no direct GET mutation link for notification row")
	}
	if !strings.Contains(body, `data-notification-unread="true"`) {
		t.Fatalf("expected unread marker for unread message")
	}
	if !strings.Contains(body, "font-semibold") {
		t.Fatalf("expected unread message to render with emphasized class")
	}
	if !strings.Contains(body, "<time datetime=") {
		t.Fatalf("expected created time element in response")
	}
	if !strings.Contains(body, "ago</time>") {
		t.Fatalf("expected relative time in \"... ago\" format")
	}
}

func TestAppNotificationOpenMarksReadAndRedirects(t *testing.T) {
	fakeClient := &fakeWebNotificationClient{
		markResp: &notificationsv1.MarkNotificationReadResponse{},
	}

	h := &handler{
		config:             Config{AuthBaseURL: "http://auth.local"},
		sessions:           newSessionStore(),
		pendingFlows:       newPendingFlowStore(),
		notificationClient: fakeClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	req := httptest.NewRequest(http.MethodPost, "/notifications/notif-1", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppNotificationsRoutes(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/notifications" {
		t.Fatalf("location = %q, want %q", location, "/notifications")
	}
	if fakeClient.markReq == nil {
		t.Fatal("expected MarkNotificationRead call")
	}
	if got := fakeClient.markReq.GetNotificationId(); got != "notif-1" {
		t.Fatalf("notification_id = %q, want %q", got, "notif-1")
	}
	userIDs := fakeClient.markMD.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-1" {
		t.Fatalf("metadata %s = %v, want [user-1]", grpcmeta.UserIDHeader, userIDs)
	}
}

func TestAppNotificationOpenRejectsGetMutationRoute(t *testing.T) {
	fakeClient := &fakeWebNotificationClient{
		markResp: &notificationsv1.MarkNotificationReadResponse{},
	}

	h := &handler{
		config:             Config{AuthBaseURL: "http://auth.local"},
		sessions:           newSessionStore(),
		pendingFlows:       newPendingFlowStore(),
		notificationClient: fakeClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	sess := h.sessions.get(sessionID, "token-1")
	sess.cachedUserID = "user-1"
	sess.cachedUserIDResolved = true

	req := httptest.NewRequest(http.MethodGet, "/notifications/notif-1", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppNotificationsRoutes(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
	if fakeClient.markReq != nil {
		t.Fatalf("expected no mark-read call for GET request")
	}
}
