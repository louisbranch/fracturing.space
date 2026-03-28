package server

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	notificationssqlite "github.com/louisbranch/fracturing.space/internal/services/notifications/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewWithDepsRequiresRuntimeHooks(t *testing.T) {
	t.Parallel()

	testEnv := func() serverEnv { return serverEnv{DBPath: filepath.Join("testdata", "notifications.db")} }
	tests := []struct {
		name string
		deps runtimeDeps
		want string
	}{
		{name: "env loader", deps: runtimeDeps{listen: net.Listen, openStore: openNotificationsStore}, want: "notifications server env loader is required"},
		{name: "listener", deps: runtimeDeps{loadEnv: testEnv, openStore: openNotificationsStore}, want: "notifications listener constructor is required"},
		{name: "store", deps: runtimeDeps{loadEnv: testEnv, listen: net.Listen}, want: "notifications store opener is required"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := newWithDeps("127.0.0.1:0", tc.deps)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("newWithDeps error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestNewWithDepsClosesListenerWhenStoreOpenFails(t *testing.T) {
	t.Parallel()

	listener := &notificationsListenerStub{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 32011}}
	_, err := newWithDeps("127.0.0.1:0", runtimeDeps{
		loadEnv: func() serverEnv { return serverEnv{DBPath: filepath.Join("testdata", "notifications.db")} },
		listen: func(string, string) (net.Listener, error) {
			return listener, nil
		},
		openStore: func(string) (*notificationssqlite.Store, error) {
			return nil, errors.New("open boom")
		},
		logf: func(string, ...any) {},
	})
	if err == nil || !strings.Contains(err.Error(), "open boom") {
		t.Fatalf("newWithDeps error = %v, want store failure", err)
	}
	if !listener.closed {
		t.Fatal("listener closed = false, want true")
	}
}

func TestServer_CreateListAndMarkReadRoundTrip(t *testing.T) {
	dbPath := t.TempDir() + "/notifications.db"
	t.Setenv("FRACTURING_SPACE_NOTIFICATIONS_DB_PATH", dbPath)

	srv, err := NewWithAddr("127.0.0.1:0")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	serveDone := make(chan error, 1)
	go func() {
		serveDone <- srv.Serve(runCtx)
	}()
	t.Cleanup(func() {
		runCancel()
		select {
		case serveErr := <-serveDone:
			if serveErr != nil {
				t.Fatalf("serve: %v", serveErr)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for server shutdown")
		}
	})

	conn, err := grpc.NewClient(srv.Addr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial notifications server: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Fatalf("close gRPC connection: %v", closeErr)
		}
	})

	client := notificationsv1.NewNotificationServiceClient(conn)

	createResp, err := client.CreateNotificationIntent(context.Background(), &notificationsv1.CreateNotificationIntentRequest{
		RecipientUserId: "user-1",
		MessageType:     "campaign.invite",
		PayloadJson:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "invite:inv-1",
		Source:          notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
	})
	if err != nil {
		t.Fatalf("create notification intent: %v", err)
	}
	if createResp.GetNotification().GetId() == "" {
		t.Fatal("expected notification id")
	}

	dupResp, err := client.CreateNotificationIntent(context.Background(), &notificationsv1.CreateNotificationIntentRequest{
		RecipientUserId: "user-1",
		MessageType:     "campaign.invite",
		PayloadJson:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "invite:inv-1",
		Source:          notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
	})
	if err != nil {
		t.Fatalf("create dedupe notification intent: %v", err)
	}
	if dupResp.GetNotification().GetId() != createResp.GetNotification().GetId() {
		t.Fatalf("dedupe id = %q, want %q", dupResp.GetNotification().GetId(), createResp.GetNotification().GetId())
	}

	userCtx := grpcauthctx.WithUserID(context.Background(), "user-1")
	listResp, err := client.ListNotifications(userCtx, &notificationsv1.ListNotificationsRequest{
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(listResp.GetNotifications()) != 1 {
		t.Fatalf("notifications len = %d, want 1", len(listResp.GetNotifications()))
	}
	unreadResp, err := client.GetUnreadNotificationStatus(userCtx, &notificationsv1.GetUnreadNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("get unread status: %v", err)
	}
	if !unreadResp.GetHasUnread() {
		t.Fatal("expected has_unread true before mark read")
	}
	if unreadResp.GetUnreadCount() != 1 {
		t.Fatalf("unread_count = %d, want 1", unreadResp.GetUnreadCount())
	}

	markResp, err := client.MarkNotificationRead(userCtx, &notificationsv1.MarkNotificationReadRequest{
		NotificationId: createResp.GetNotification().GetId(),
	})
	if err != nil {
		t.Fatalf("mark notification read: %v", err)
	}
	if markResp.GetNotification().GetReadAt() == nil {
		t.Fatal("expected read_at timestamp")
	}
	unreadAfterResp, err := client.GetUnreadNotificationStatus(userCtx, &notificationsv1.GetUnreadNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("get unread status after mark read: %v", err)
	}
	if unreadAfterResp.GetHasUnread() {
		t.Fatal("expected has_unread false after mark read")
	}
	if unreadAfterResp.GetUnreadCount() != 0 {
		t.Fatalf("unread_count after mark read = %d, want 0", unreadAfterResp.GetUnreadCount())
	}
}

type notificationsListenerStub struct {
	addr   net.Addr
	closed bool
}

func (l *notificationsListenerStub) Accept() (net.Conn, error) {
	return nil, errors.New("not implemented")
}
func (l *notificationsListenerStub) Close() error {
	l.closed = true
	return nil
}
func (l *notificationsListenerStub) Addr() net.Addr { return l.addr }
