package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/i18n"
	"golang.org/x/text/language"
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

func TestToNotificationListItems_UnknownTopicAndUnknownSignupUseSafeRenderedBody(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	items := toNotificationListItems(webi18n.Printer(language.AmericanEnglish), []*notificationsv1.Notification{
		{
			Id:          "notif-signup-unknown",
			Topic:       "auth.onboarding.welcome",
			PayloadJson: `{"signup_method":"oauth"}`,
			Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
			CreatedAt:   timestamppb.New(now.Add(2 * time.Minute)),
		},
		{
			Id:          "notif-topic-unknown",
			Topic:       "unknown.topic.slug",
			PayloadJson: `{"foo":"bar"}`,
			Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
			CreatedAt:   timestamppb.New(now.Add(1 * time.Minute)),
		},
	}, now)

	if got := len(items); got != 2 {
		t.Fatalf("items = %d, want 2", got)
	}
	if items[0].Topic != "Welcome to Fracturing Space" {
		t.Fatalf("items[0].Topic = %q, want %q", items[0].Topic, "Welcome to Fracturing Space")
	}
	if items[0].BodyText != "Your account is ready. Sign-in method: another method." {
		t.Fatalf("items[0].BodyText = %q, want safe unknown-signup label", items[0].BodyText)
	}
	if strings.Contains(items[0].BodyText, "oauth") {
		t.Fatalf("items[0].BodyText = %q, expected raw signup method to stay hidden", items[0].BodyText)
	}

	if items[1].Topic != "Notification" {
		t.Fatalf("items[1].Topic = %q, want %q", items[1].Topic, "Notification")
	}
	if items[1].BodyText != "You have a new notification." {
		t.Fatalf("items[1].BodyText = %q, want generic unknown-topic copy", items[1].BodyText)
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
					Id:          "notif-old",
					Topic:       "auth.onboarding.welcome.v1",
					Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
					CreatedAt:   timestamppb.New(oldCreatedAt),
					ReadAt:      timestamppb.New(readAt),
					PayloadJson: `{"signup_method":"magic_link"}`,
				},
				{
					Id:          "notif-new",
					Topic:       "auth.onboarding.welcome",
					Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
					CreatedAt:   timestamppb.New(newCreatedAt),
					PayloadJson: `{"signup_method":"passkey"}`,
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

	req := httptest.NewRequest(http.MethodGet, "/notifications?filter=all", nil)
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
	if !strings.Contains(body, `data-notifications-layout="split"`) {
		t.Fatalf("expected split layout marker")
	}
	if strings.Contains(body, `class="breadcrumbs text-sm"`) {
		t.Fatalf("expected notifications page to render with empty breadcrumbs")
	}
	if !strings.Contains(body, `data-notification-tab-content`) {
		t.Fatalf("expected connected tab-content marker")
	}
	if strings.Contains(body, `border border-base-300 border-t-0 bg-base-200/70`) {
		t.Fatalf("expected notifications list wrapper to avoid extra layered border/background styles")
	}
	if !strings.Contains(body, `class="menu bg-base-200 rounded-box w-full"`) {
		t.Fatalf("expected notifications list to use standard app menu classes")
	}
	if !strings.Contains(body, `data-notification-detail`) {
		t.Fatalf("expected detail panel marker")
	}
	if !strings.Contains(body, `data-notifications-filter="all"`) {
		t.Fatalf("expected all filter state marker")
	}
	if !strings.Contains(body, "Welcome to Fracturing Space") {
		t.Fatalf("expected localized notification title in response")
	}
	newIdx := strings.Index(body, `data-notification-id="notif-new"`)
	oldIdx := strings.Index(body, `data-notification-id="notif-old"`)
	if newIdx < 0 || oldIdx < 0 {
		t.Fatalf("expected both notification ids in rendered output")
	}
	if newIdx > oldIdx {
		t.Fatalf("expected newer notification to render first")
	}
	if !strings.Contains(body, `data-notification-id="notif-new"`) {
		t.Fatalf("expected notification id marker for newest message")
	}
	if !strings.Contains(body, `href="/notifications?filter=all&amp;selected=notif-new"`) {
		t.Fatalf("expected notification row GET link with preserved list state")
	}
	if strings.Contains(body, `form method="POST" action="/notifications/notif-new`) {
		t.Fatalf("expected no POST row form")
	}
	if !strings.Contains(body, `data-notification-unread="true"`) {
		t.Fatalf("expected unread marker for unread message")
	}
	newRowID := `data-notification-id="notif-new"`
	newRowPos := strings.Index(body, newRowID)
	if newRowPos < 0 {
		t.Fatalf("expected notification row markup for notif-new")
	}
	newRowStart := strings.LastIndex(body[:newRowPos], "<a")
	if newRowStart < 0 {
		t.Fatalf("expected notif-new row to render as anchor")
	}
	newRowEndOffset := strings.Index(body[newRowPos:], "</a>")
	if newRowEndOffset < 0 {
		t.Fatalf("expected notif-new row to have closing anchor tag")
	}
	newRowMarkup := body[newRowStart : newRowPos+newRowEndOffset]
	if strings.Contains(newRowMarkup, `class="text-xs opacity-70 whitespace-nowrap"`) {
		t.Fatalf("expected notification list row to omit created time metadata")
	}
	if strings.Contains(newRowMarkup, `class="text-sm opacity-75"`) {
		t.Fatalf("expected notification list row to omit source metadata")
	}
	if !strings.Contains(body, "font-semibold") {
		t.Fatalf("expected unread message to render with emphasized class")
	}
	if !strings.Contains(body, `data-notification-selected="true"`) {
		t.Fatalf("expected selected row marker")
	}
	if !strings.Contains(body, "Your account is ready. Sign-in method: passkey.") {
		t.Fatalf("expected localized onboarding detail body")
	}
	if strings.Contains(body, "\"signup_method\"") {
		t.Fatalf("expected payload JSON keys to stay hidden from end-user copy")
	}
	if !strings.Contains(body, `class="text-sm opacity-80">System</p>`) {
		t.Fatalf("expected localized source label in detail pane")
	}
	if !strings.Contains(body, "<time datetime=") {
		t.Fatalf("expected created time element in response")
	}
	if !strings.Contains(body, "ago</time>") {
		t.Fatalf("expected relative time in \"... ago\" format")
	}
}

func TestAppNotificationsPageRendersLocalizedNotificationCopyInsteadOfArtifactValues(t *testing.T) {
	now := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)
	createdAt := now.Add(-10 * time.Minute)

	fakeClient := &fakeWebNotificationClient{
		listResp: &notificationsv1.ListNotificationsResponse{
			Notifications: []*notificationsv1.Notification{
				{
					Id:          "notif-welcome",
					Topic:       "auth.onboarding.welcome",
					Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
					CreatedAt:   timestamppb.New(createdAt),
					PayloadJson: `{"signup_method":"passkey","event_type":"auth.signup_completed"}`,
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

	req := httptest.NewRequest(http.MethodGet, "/notifications?filter=all&selected=notif-welcome", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppNotifications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Welcome to Fracturing Space") {
		t.Fatalf("expected localized onboarding title")
	}
	if !strings.Contains(body, "Your account is ready. Sign-in method: passkey.") {
		t.Fatalf("expected localized onboarding detail body")
	}
	if strings.Contains(body, "auth.onboarding.welcome") {
		t.Fatalf("expected artifact topic slug to stay hidden from end-user copy")
	}
	if strings.Contains(body, "\"signup_method\"") {
		t.Fatalf("expected payload JSON keys to stay hidden from end-user copy")
	}
}

func TestAppNotificationsPageSelectedUnreadMarksRead(t *testing.T) {
	now := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)
	newCreatedAt := now.Add(-10 * time.Minute)

	fakeClient := &fakeWebNotificationClient{
		listResp: &notificationsv1.ListNotificationsResponse{
			Notifications: []*notificationsv1.Notification{
				{
					Id:          "notif-new",
					Topic:       "Newest unread",
					Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
					CreatedAt:   timestamppb.New(newCreatedAt),
					PayloadJson: `{"event_type":"auth.signup_completed"}`,
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
	req := httptest.NewRequest(http.MethodGet, "/notifications?filter=all&selected=notif-new", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppNotifications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if fakeClient.markReq == nil {
		t.Fatal("expected mark-read call for selected unread notification")
	}
	if got := fakeClient.markReq.GetNotificationId(); got != "notif-new" {
		t.Fatalf("notification_id = %q, want %q", got, "notif-new")
	}
	userIDs := fakeClient.markMD.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "user-1" {
		t.Fatalf("metadata %s = %v, want [user-1]", grpcmeta.UserIDHeader, userIDs)
	}
}

func TestAppNotificationsPageDefaultsToUnreadFilterAndSelectsNewestUnread(t *testing.T) {
	now := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)
	oldCreatedAt := now.Add(-2 * time.Hour)
	newCreatedAt := now.Add(-10 * time.Minute)
	readAt := now.Add(-30 * time.Minute)

	fakeClient := &fakeWebNotificationClient{
		listResp: &notificationsv1.ListNotificationsResponse{
			Notifications: []*notificationsv1.Notification{
				{
					Id:          "notif-old",
					Topic:       "auth.onboarding.welcome.v1",
					Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
					CreatedAt:   timestamppb.New(oldCreatedAt),
					ReadAt:      timestamppb.New(readAt),
					PayloadJson: `{"signup_method":"magic_link"}`,
				},
				{
					Id:          "notif-new",
					Topic:       "auth.onboarding.welcome",
					Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
					CreatedAt:   timestamppb.New(newCreatedAt),
					PayloadJson: `{"signup_method":"passkey"}`,
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
	body := w.Body.String()
	if !strings.Contains(body, `data-notifications-filter="unread"`) {
		t.Fatalf("expected unread filter to be default")
	}
	if strings.Contains(body, `data-notification-id="notif-old"`) {
		t.Fatalf("expected read-only row to be hidden under unread filter")
	}
	if !strings.Contains(body, `data-notification-id="notif-new"`) {
		t.Fatalf("expected newest unread row")
	}
	if !strings.Contains(body, `data-notification-selected="true"`) {
		t.Fatalf("expected selected unread row marker")
	}
	if !strings.Contains(body, "Your account is ready. Sign-in method: passkey.") {
		t.Fatalf("expected selected unread detail content")
	}
	if fakeClient.markReq == nil {
		t.Fatal("expected rendered default-selected unread notification to mark as read")
	}
	if got := fakeClient.markReq.GetNotificationId(); got != "notif-new" {
		t.Fatalf("notification_id = %q, want %q", got, "notif-new")
	}
}

func TestAppNotificationsPageRenderedUnreadClearsUnreadCacheAfterMarkRead(t *testing.T) {
	now := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)
	newCreatedAt := now.Add(-10 * time.Minute)

	fakeClient := &fakeWebNotificationClient{
		listResp: &notificationsv1.ListNotificationsResponse{
			Notifications: []*notificationsv1.Notification{
				{
					Id:          "notif-new",
					Topic:       "Newest unread",
					Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
					CreatedAt:   timestamppb.New(newCreatedAt),
					PayloadJson: `{"event_type":"auth.signup_completed"}`,
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
	if fakeClient.markReq == nil {
		t.Fatal("expected rendered default-selected unread notification to mark as read")
	}
	if _, ok := sess.cachedUnreadNotifications(unreadNotificationProbeTTL); ok {
		t.Fatalf("expected unread cache to be invalidated after rendered read mutation")
	}
}

func TestAppNotificationsPageSelectsExplicitNotificationInAllFilter(t *testing.T) {
	now := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)
	oldCreatedAt := now.Add(-2 * time.Hour)
	newCreatedAt := now.Add(-10 * time.Minute)
	readAt := now.Add(-30 * time.Minute)

	fakeClient := &fakeWebNotificationClient{
		listResp: &notificationsv1.ListNotificationsResponse{
			Notifications: []*notificationsv1.Notification{
				{
					Id:          "notif-old",
					Topic:       "auth.onboarding.welcome.v1",
					Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
					CreatedAt:   timestamppb.New(oldCreatedAt),
					ReadAt:      timestamppb.New(readAt),
					PayloadJson: `{"signup_method":"magic_link"}`,
				},
				{
					Id:          "notif-new",
					Topic:       "auth.onboarding.welcome",
					Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
					CreatedAt:   timestamppb.New(newCreatedAt),
					PayloadJson: `{"signup_method":"passkey"}`,
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

	req := httptest.NewRequest(http.MethodGet, "/notifications?filter=all&selected=notif-old", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppNotifications(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `data-notifications-filter="all"`) {
		t.Fatalf("expected all filter state marker")
	}
	if !strings.Contains(body, `data-notification-id="notif-old"`) {
		t.Fatalf("expected selected row id marker")
	}
	if !strings.Contains(body, `data-notification-selected="true"`) {
		t.Fatalf("expected selected row marker")
	}
	if !strings.Contains(body, "Your account is ready. Sign-in method: magic link.") {
		t.Fatalf("expected explicit selected detail content")
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

	req := httptest.NewRequest(http.MethodPost, "/notifications/notif-1?filter=unread", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppNotificationsRoutes(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	location := w.Header().Get("Location")
	parsedLocation, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location %q: %v", location, err)
	}
	if parsedLocation.Path != "/notifications" {
		t.Fatalf("location path = %q, want %q", parsedLocation.Path, "/notifications")
	}
	if got := parsedLocation.Query().Get("filter"); got != "unread" {
		t.Fatalf("location filter query = %q, want %q", got, "unread")
	}
	if got := parsedLocation.Query().Get("selected"); got != "notif-1" {
		t.Fatalf("location selected query = %q, want %q", got, "notif-1")
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
