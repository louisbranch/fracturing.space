package server

import (
	"context"
	"testing"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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
		Topic:           "campaign.invite",
		PayloadJson:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "invite:inv-1",
		Source:          "game",
	})
	if err != nil {
		t.Fatalf("create notification intent: %v", err)
	}
	if createResp.GetNotification().GetId() == "" {
		t.Fatal("expected notification id")
	}

	dupResp, err := client.CreateNotificationIntent(context.Background(), &notificationsv1.CreateNotificationIntentRequest{
		RecipientUserId: "user-1",
		Topic:           "campaign.invite",
		PayloadJson:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "invite:inv-1",
		Source:          "game",
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
