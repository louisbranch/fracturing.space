//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestUserhubDashboardIntegration(t *testing.T) {
	fixture := newSuiteFixture(t)

	socialAddr := fixture.startSocialServer(t)
	notificationsAddr := fixture.startNotificationsServer(t)
	inviteAddr := fixture.startInviteServer(t)
	userhubAddr := fixture.startUserHubServer(t)

	recipientUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "userhub-recipient"))
	ownerUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "userhub-owner"))

	gameConn := dialRuntimeGRPC(t, fixture.grpcAddr)
	t.Cleanup(func() { _ = gameConn.Close() })

	inviteConn := dialRuntimeGRPC(t, inviteAddr)
	t.Cleanup(func() { _ = inviteConn.Close() })

	socialConn := dialRuntimeGRPC(t, socialAddr)
	t.Cleanup(func() { _ = socialConn.Close() })

	notificationsConn := dialRuntimeGRPC(t, notificationsAddr)
	t.Cleanup(func() { _ = notificationsConn.Close() })

	userhubConn := dialRuntimeGRPC(t, userhubAddr)
	t.Cleanup(func() { _ = userhubConn.Close() })

	recipientGame := &integrationSuite{
		conn:        gameConn,
		campaign:    gamev1.NewCampaignServiceClient(gameConn),
		participant: gamev1.NewParticipantServiceClient(gameConn),
		character:   gamev1.NewCharacterServiceClient(gameConn),
		session:     gamev1.NewSessionServiceClient(gameConn),
		userID:      recipientUserID,
	}
	ownerInviteSuite := &inviteLifecycleSuite{
		invite:      invitev1.NewInviteServiceClient(inviteConn),
		participant: gamev1.NewParticipantServiceClient(gameConn),
		campaign:    gamev1.NewCampaignServiceClient(gameConn),
		authAddr:    fixture.authAddr,
		ownerUserID: ownerUserID,
	}
	socialClient := socialv1.NewSocialServiceClient(socialConn)
	notificationsClient := notificationsv1.NewNotificationServiceClient(notificationsConn)
	userhubClient := userhubv1.NewUserHubServiceClient(userhubConn)
	userhubControlClient := userhubv1.NewUserHubControlServiceClient(userhubConn)

	setRecipientProfile(t, socialClient, recipientUserID, "Dashboard Recipient")
	activeCampaignID, activeSessionID := createRecipientActiveCampaign(t, recipientGame)
	inviteID := createPendingInviteForRecipient(t, ownerInviteSuite, recipientUserID)
	createSystemNotification(t, notificationsClient, recipientUserID, "dashboard.integration.seed", `{"kind":"seed"}`, "dashboard-seed-1")

	initialDashboard := waitForDashboard(t, userhubClient, recipientUserID, func(resp *userhubv1.GetDashboardResponse) bool {
		return resp.GetInvites().GetListedCount() == 1 &&
			resp.GetNotifications().GetUnreadCount() == 1 &&
			resp.GetCampaigns().GetListedCount() == 1 &&
			resp.GetActiveSessions().GetListedCount() == 1
	})

	if initialDashboard.GetMetadata().GetFreshness() != userhubv1.DashboardFreshness_DASHBOARD_FRESHNESS_FRESH {
		t.Fatalf("dashboard freshness = %v, want fresh", initialDashboard.GetMetadata().GetFreshness())
	}
	if initialDashboard.GetMetadata().GetCacheHit() {
		t.Fatal("dashboard cache_hit = true, want false")
	}
	if initialDashboard.GetMetadata().GetDegraded() {
		t.Fatalf("dashboard degraded = true, want false (dependencies=%v)", initialDashboard.GetMetadata().GetDegradedDependencies())
	}

	user := initialDashboard.GetUser()
	if got := user.GetUserId(); got != recipientUserID {
		t.Fatalf("dashboard user_id = %q, want %q", got, recipientUserID)
	}
	if got := user.GetName(); got != "Dashboard Recipient" {
		t.Fatalf("dashboard name = %q, want %q", got, "Dashboard Recipient")
	}
	if got := user.GetUsername(); got == "" {
		t.Fatal("dashboard username is empty")
	}
	if !user.GetProfileAvailable() {
		t.Fatal("dashboard profile_available = false, want true")
	}
	if user.GetNeedsProfileCompletion() {
		t.Fatal("dashboard needs_profile_completion = true, want false")
	}

	invites := initialDashboard.GetInvites()
	if !invites.GetAvailable() {
		t.Fatal("dashboard invites available = false, want true")
	}
	if invites.GetListedCount() != 1 {
		t.Fatalf("dashboard invite count = %d, want 1", invites.GetListedCount())
	}
	if len(invites.GetPending()) != 1 {
		t.Fatalf("dashboard invite previews = %d, want 1", len(invites.GetPending()))
	}
	if got := invites.GetPending()[0].GetInviteId(); got != inviteID {
		t.Fatalf("dashboard invite id = %q, want %q", got, inviteID)
	}

	notifications := initialDashboard.GetNotifications()
	if !notifications.GetAvailable() {
		t.Fatal("dashboard notifications available = false, want true")
	}
	if !notifications.GetHasUnread() || notifications.GetUnreadCount() != 1 {
		t.Fatalf("dashboard notifications = %+v, want has_unread=true unread_count=1", notifications)
	}

	campaigns := initialDashboard.GetCampaigns()
	if !campaigns.GetAvailable() {
		t.Fatal("dashboard campaigns available = false, want true")
	}
	if campaigns.GetListedCount() != 1 || campaigns.GetActiveCount() != 1 {
		t.Fatalf("dashboard campaigns listed/active = %d/%d, want 1/1", campaigns.GetListedCount(), campaigns.GetActiveCount())
	}
	if len(campaigns.GetCampaigns()) != 1 {
		t.Fatalf("dashboard campaign previews = %d, want 1", len(campaigns.GetCampaigns()))
	}
	if got := campaigns.GetCampaigns()[0].GetCampaignId(); got != activeCampaignID {
		t.Fatalf("dashboard campaign id = %q, want %q", got, activeCampaignID)
	}

	activeSessions := initialDashboard.GetActiveSessions()
	if !activeSessions.GetAvailable() {
		t.Fatal("dashboard active_sessions available = false, want true")
	}
	if activeSessions.GetListedCount() != 1 || len(activeSessions.GetSessions()) != 1 {
		t.Fatalf("dashboard active sessions = %d/%d, want 1/1", activeSessions.GetListedCount(), len(activeSessions.GetSessions()))
	}
	if got := activeSessions.GetSessions()[0].GetSessionId(); got != activeSessionID {
		t.Fatalf("dashboard active session id = %q, want %q", got, activeSessionID)
	}

	wantActions := []userhubv1.DashboardActionID{
		userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_REVIEW_PENDING_INVITES,
		userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_CONTINUE_ACTIVE_CAMPAIGN,
		userhubv1.DashboardActionID_DASHBOARD_ACTION_ID_REVIEW_NOTIFICATIONS,
	}
	if got := dashboardActionIDs(initialDashboard); !equalDashboardActionIDs(got, wantActions) {
		t.Fatalf("dashboard action ids = %v, want %v", got, wantActions)
	}

	createSystemNotification(t, notificationsClient, recipientUserID, "dashboard.integration.followup", `{"kind":"followup"}`, "dashboard-seed-2")

	cachedDashboard := getDashboard(t, userhubClient, recipientUserID)
	if !cachedDashboard.GetMetadata().GetCacheHit() {
		t.Fatal("cached dashboard cache_hit = false, want true")
	}
	if cachedDashboard.GetNotifications().GetUnreadCount() != 1 {
		t.Fatalf("cached dashboard unread_count = %d, want cached value 1", cachedDashboard.GetNotifications().GetUnreadCount())
	}

	invalidateResp, err := userhubControlClient.InvalidateDashboards(context.Background(), &userhubv1.InvalidateDashboardsRequest{
		UserIds: []string{recipientUserID},
		Reason:  "integration-test-refresh",
	})
	if err != nil {
		t.Fatalf("invalidate dashboards: %v", err)
	}
	if invalidateResp.GetInvalidatedEntries() < 1 {
		t.Fatalf("invalidated entries = %d, want at least 1", invalidateResp.GetInvalidatedEntries())
	}

	refreshedDashboard := waitForDashboard(t, userhubClient, recipientUserID, func(resp *userhubv1.GetDashboardResponse) bool {
		return !resp.GetMetadata().GetCacheHit() && resp.GetNotifications().GetUnreadCount() == 2
	})
	if refreshedDashboard.GetNotifications().GetUnreadCount() != 2 {
		t.Fatalf("refreshed dashboard unread_count = %d, want 2", refreshedDashboard.GetNotifications().GetUnreadCount())
	}
}

func dialRuntimeGRPC(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC %q: %v", addr, err)
	}
	return conn
}

func setRecipientProfile(t *testing.T, client socialv1.SocialServiceClient, userID string, name string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	if _, err := client.SetUserProfile(ctx, &socialv1.SetUserProfileRequest{
		UserId: userID,
		Name:   name,
	}); err != nil {
		t.Fatalf("set user profile: %v", err)
	}
}

func createRecipientActiveCampaign(t *testing.T, suite *integrationSuite) (campaignID string, sessionID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(suite.ctx(context.Background()), integrationTimeout())
	defer cancel()

	campaignResp, err := suite.campaign.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:   "userhub-active-" + t.Name(),
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: gamev1.GmMode_HUMAN,
	})
	if err != nil {
		t.Fatalf("create recipient campaign: %v", err)
	}

	campaignID = campaignResp.GetCampaign().GetId()
	if campaignID == "" {
		t.Fatal("recipient campaign id is empty")
	}

	participantResp, err := suite.participant.CreateParticipant(ctx, &gamev1.CreateParticipantRequest{
		CampaignId: campaignID,
		Name:       "Recipient Player",
		Role:       gamev1.ParticipantRole_PLAYER,
		Controller: gamev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create recipient participant: %v", err)
	}
	participantID := participantResp.GetParticipant().GetId()
	if participantID == "" {
		t.Fatal("recipient participant id is empty")
	}

	characterResp, err := suite.character.CreateCharacter(ctx, &gamev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       "Recipient Hero",
		Kind:       gamev1.CharacterKind_PC,
	})
	if err != nil {
		t.Fatalf("create recipient character: %v", err)
	}
	characterID := characterResp.GetCharacter().GetId()
	if characterID == "" {
		t.Fatal("recipient character id is empty")
	}

	setCharacterOwner(t, ctx, suite.character, campaignID, characterID, participantID)
	ensureDaggerheartCreationReadiness(t, ctx, suite.character, campaignID, characterID)

	sessionResp := startSessionWithDefaultControllers(t, ctx, suite.session, suite.character, campaignID, "Dashboard Session")

	sessionID = sessionResp.GetSession().GetId()
	if sessionID == "" {
		t.Fatal("recipient session id is empty")
	}

	return campaignID, sessionID
}

func createPendingInviteForRecipient(t *testing.T, suite *inviteLifecycleSuite, recipientUserID string) string {
	t.Helper()

	campaignID, participantID := suite.createCampaignWithPlayerSeat(t)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createResp, err := suite.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   participantID,
		RecipientUserId: recipientUserID,
	})
	if err != nil {
		t.Fatalf("create pending invite: %v", err)
	}

	inviteID := createResp.GetInvite().GetId()
	if inviteID == "" {
		t.Fatal("pending invite id is empty")
	}
	return inviteID
}

func createSystemNotification(
	t *testing.T,
	client notificationsv1.NotificationServiceClient,
	recipientUserID string,
	messageType string,
	payloadJSON string,
	dedupeKey string,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	if _, err := client.CreateNotificationIntent(ctx, &notificationsv1.CreateNotificationIntentRequest{
		RecipientUserId: recipientUserID,
		MessageType:     messageType,
		PayloadJson:     payloadJSON,
		DedupeKey:       dedupeKey,
		Source:          notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
	}); err != nil {
		t.Fatalf("create notification intent: %v", err)
	}
}

func getDashboard(t *testing.T, client userhubv1.UserHubServiceClient, userID string) *userhubv1.GetDashboardResponse {
	t.Helper()

	ctx, cancel := context.WithTimeout(withUserID(context.Background(), userID), integrationTimeout())
	defer cancel()

	resp, err := client.GetDashboard(ctx, &userhubv1.GetDashboardRequest{
		CampaignPreviewLimit: 5,
		InvitePreviewLimit:   5,
	})
	if err != nil {
		t.Fatalf("get dashboard: %v", err)
	}
	return resp
}

func waitForDashboard(
	t *testing.T,
	client userhubv1.UserHubServiceClient,
	userID string,
	match func(*userhubv1.GetDashboardResponse) bool,
) *userhubv1.GetDashboardResponse {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp := getDashboard(t, client, userID)
		if match(resp) {
			return resp
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for dashboard match for user %q", userID)
	return nil
}

func dashboardActionIDs(resp *userhubv1.GetDashboardResponse) []userhubv1.DashboardActionID {
	if resp == nil {
		return nil
	}
	actions := resp.GetNextActions()
	ids := make([]userhubv1.DashboardActionID, 0, len(actions))
	for _, action := range actions {
		if action == nil {
			continue
		}
		ids = append(ids, action.GetId())
	}
	return ids
}

func equalDashboardActionIDs(left, right []userhubv1.DashboardActionID) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
