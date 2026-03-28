//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	workerdomain "github.com/louisbranch/fracturing.space/internal/services/worker/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestInviteNotificationDeliveryIntegration(t *testing.T) {
	fixture := newSuiteFixture(t)

	fixture.startSocialServer(t)
	notificationsAddr := fixture.startNotificationsServer(t)
	inviteAddr := fixture.startInviteServer(t)
	_ = fixture.startWorkerRuntime(t)

	ownerUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "invite-owner"))
	recipientUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "invite-recipient"))

	inviteConn, err := grpc.NewClient(
		inviteAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial invite gRPC: %v", err)
	}
	t.Cleanup(func() { _ = inviteConn.Close() })

	gameConn, err := grpc.NewClient(
		fixture.grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial game gRPC: %v", err)
	}
	t.Cleanup(func() { _ = gameConn.Close() })

	notificationsConn, err := grpc.NewClient(
		notificationsAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial notifications gRPC: %v", err)
	}
	t.Cleanup(func() { _ = notificationsConn.Close() })

	suite := &inviteLifecycleSuite{
		invite:      invitev1.NewInviteServiceClient(inviteConn),
		participant: gamev1.NewParticipantServiceClient(gameConn),
		campaign:    gamev1.NewCampaignServiceClient(gameConn),
		authAddr:    fixture.authAddr,
		ownerUserID: ownerUserID,
	}

	campaignID, participantID := suite.createCampaignWithPlayerSeat(t)

	createCtx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createResp, err := suite.invite.CreateInvite(createCtx, &invitev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   participantID,
		RecipientUserId: recipientUserID,
	})
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}

	notificationsClient := notificationsv1.NewNotificationServiceClient(notificationsConn)
	notification := waitForRecipientNotification(
		t,
		notificationsClient,
		recipientUserID,
		workerdomain.InviteCreatedNotificationDedupeKey(createResp.GetInvite().GetId()),
	)

	if got := notification.GetRecipientUserId(); got != recipientUserID {
		t.Fatalf("notification recipient = %q, want %q", got, recipientUserID)
	}
	if got := notification.GetMessageType(); got != workerdomain.InviteNotificationCreatedMessageType {
		t.Fatalf("notification message type = %q, want %q", got, workerdomain.InviteNotificationCreatedMessageType)
	}
	if !strings.Contains(notification.GetPayloadJson(), createResp.GetInvite().GetId()) {
		t.Fatalf("notification payload = %q, want invite id %q", notification.GetPayloadJson(), createResp.GetInvite().GetId())
	}

	userCtx := withUserID(context.Background(), recipientUserID)
	unreadResp, err := notificationsClient.GetUnreadNotificationStatus(userCtx, &notificationsv1.GetUnreadNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("get unread notification status: %v", err)
	}
	if !unreadResp.GetHasUnread() || unreadResp.GetUnreadCount() != 1 {
		t.Fatalf("unread status = %+v, want has_unread=true unread_count=1", unreadResp)
	}

	_, err = notificationsClient.MarkNotificationRead(userCtx, &notificationsv1.MarkNotificationReadRequest{
		NotificationId: notification.GetId(),
	})
	if err != nil {
		t.Fatalf("mark notification read: %v", err)
	}

	unreadAfterResp, err := notificationsClient.GetUnreadNotificationStatus(userCtx, &notificationsv1.GetUnreadNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("get unread notification status after read: %v", err)
	}
	if unreadAfterResp.GetHasUnread() || unreadAfterResp.GetUnreadCount() != 0 {
		t.Fatalf("unread status after read = %+v, want has_unread=false unread_count=0", unreadAfterResp)
	}
}

func waitForRecipientNotification(
	t *testing.T,
	client notificationsv1.NotificationServiceClient,
	recipientUserID string,
	dedupeKey string,
) *notificationsv1.Notification {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	userCtx := withUserID(context.Background(), recipientUserID)

	for time.Now().Before(deadline) {
		callCtx, cancel := context.WithTimeout(userCtx, integrationTimeout())
		resp, err := client.ListNotifications(callCtx, &notificationsv1.ListNotificationsRequest{PageSize: 20})
		cancel()
		if err == nil {
			for _, notification := range resp.GetNotifications() {
				if notification.GetDedupeKey() == dedupeKey {
					return notification
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for notification dedupe key %q for recipient %q", dedupeKey, recipientUserID)
	return nil
}
