//go:build integration

package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	discoveryserver "github.com/louisbranch/fracturing.space/internal/services/discovery/app"
	"github.com/louisbranch/fracturing.space/internal/services/discovery/catalog"
	discoverystorage "github.com/louisbranch/fracturing.space/internal/services/discovery/storage"
	discoverysqlite "github.com/louisbranch/fracturing.space/internal/services/discovery/storage/sqlite"
	userhubapp "github.com/louisbranch/fracturing.space/internal/services/userhub/app"
	workerdomain "github.com/louisbranch/fracturing.space/internal/services/worker/domain"
	"github.com/louisbranch/fracturing.space/internal/test/testkit"
)

func TestInviteWorkerNotificationsUserhubIntegration(t *testing.T) {
	fixture := newSuiteFixture(t)

	fixture.startSocialServer(t)
	inviteAddr := fixture.startInviteServer(t)
	notificationsAddr := fixture.startNotificationsServer(t)
	_ = fixture.startWorkerRuntime(t)
	userhubAddr := fixture.startUserHubServer(t)

	recipientUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "chain-recipient"))
	ownerUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "chain-owner"))

	gameConn := dialRuntimeGRPC(t, fixture.grpcAddr)
	t.Cleanup(func() { _ = gameConn.Close() })

	inviteConn := dialRuntimeGRPC(t, inviteAddr)
	t.Cleanup(func() { _ = inviteConn.Close() })

	notificationsConn := dialRuntimeGRPC(t, notificationsAddr)
	t.Cleanup(func() { _ = notificationsConn.Close() })

	userhubConn := dialRuntimeGRPC(t, userhubAddr)
	t.Cleanup(func() { _ = userhubConn.Close() })

	ownerInviteSuite := &inviteLifecycleSuite{
		invite:      invitev1.NewInviteServiceClient(inviteConn),
		participant: gamev1.NewParticipantServiceClient(gameConn),
		campaign:    gamev1.NewCampaignServiceClient(gameConn),
		authAddr:    fixture.authAddr,
		ownerUserID: ownerUserID,
	}
	notificationsClient := notificationsv1.NewNotificationServiceClient(notificationsConn)
	userhubClient := userhubv1.NewUserHubServiceClient(userhubConn)

	inviteID := createPendingInviteForRecipient(t, ownerInviteSuite, recipientUserID)
	notification := waitForRecipientNotification(
		t,
		notificationsClient,
		recipientUserID,
		workerdomain.InviteCreatedNotificationDedupeKey(inviteID),
	)
	if got := notification.GetRecipientUserId(); got != recipientUserID {
		t.Fatalf("notification recipient = %q, want %q", got, recipientUserID)
	}

	dashboard := waitForDashboard(t, userhubClient, recipientUserID, func(resp *userhubv1.GetDashboardResponse) bool {
		return resp.GetInvites().GetListedCount() == 1 && resp.GetNotifications().GetUnreadCount() == 1
	})
	if dashboard.GetMetadata().GetDegraded() {
		t.Fatalf("dashboard degraded = true, want false (%v)", dashboard.GetMetadata().GetDegradedDependencies())
	}
	if got := dashboard.GetInvites().GetPending()[0].GetInviteId(); got != inviteID {
		t.Fatalf("dashboard invite id = %q, want %q", got, inviteID)
	}
	if dashboard.GetNotifications().GetUnreadCount() != 1 {
		t.Fatalf("dashboard unread count = %d, want 1", dashboard.GetNotifications().GetUnreadCount())
	}
}

func TestDiscoveryStarterReconciliationIntegration(t *testing.T) {
	fixture := newSuiteFixture(t)
	dbPath := filepath.Join(t.TempDir(), "discovery.db")
	t.Setenv("FRACTURING_SPACE_DISCOVERY_DB_PATH", dbPath)
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", fixture.grpcAddr)

	gameConn := dialRuntimeGRPC(t, fixture.grpcAddr)
	t.Cleanup(func() { _ = gameConn.Close() })
	forkClient := gamev1.NewForkServiceClient(gameConn)
	launcherUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "discovery-launcher"))

	ctx, cancel := context.WithCancel(context.Background())
	srv, err := discoveryserver.NewWithAddr("127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatalf("new discovery server: %v", err)
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ctx)
	}()
	testkit.WaitForGRPCHealth(t, srv.Addr())
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-serveErr:
			if err != nil {
				t.Fatalf("discovery serve: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for discovery server shutdown")
		}
	})

	starters, err := catalog.BuiltinStarters()
	if err != nil {
		t.Fatalf("builtin starters: %v", err)
	}
	if len(starters) == 0 {
		t.Fatal("expected builtin starters")
	}

	entryIDs := make([]string, 0, len(starters))
	for _, starter := range starters {
		entryIDs = append(entryIDs, starter.Entry.EntryID)
	}
	entry := waitForDiscoveryStarterTemplate(t, dbPath, entryIDs...)
	if got := entry.SourceID; got == "" {
		t.Fatal("starter source_id is empty")
	}

	forkCtx, forkCancel := context.WithTimeout(withUserID(context.Background(), launcherUserID), integrationTimeout())
	defer forkCancel()
	forkResp, err := forkClient.ForkCampaign(forkCtx, &gamev1.ForkCampaignRequest{
		SourceCampaignId: entry.SourceID,
		NewCampaignName:  "Discovery Starter Launch",
		CopyParticipants: true,
	})
	if err != nil {
		t.Fatalf("fork starter template: %v", err)
	}
	if forkResp.GetCampaign() == nil || forkResp.GetCampaign().GetId() == "" {
		t.Fatal("forked starter campaign id is empty")
	}
	if got := forkResp.GetLineage().GetParentCampaignId(); got != entry.SourceID {
		t.Fatalf("fork lineage parent campaign id = %q, want %q", got, entry.SourceID)
	}
	if got := forkResp.GetCampaign().GetSystem(); got != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("forked starter campaign system = %v, want %v", got, commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	}
}

func TestUserhubDashboardDegradedNotificationsIntegration(t *testing.T) {
	fixture := newSuiteFixture(t)

	socialAddr := fixture.startSocialServer(t)
	inviteAddr := fixture.startInviteServer(t)
	userhubAddr := fixture.mesh.StartUserHubServer(userhubapp.RuntimeConfig{
		AuthAddr:          fixture.authAddr,
		GameAddr:          fixture.grpcAddr,
		InviteAddr:        inviteAddr,
		SocialAddr:        socialAddr,
		NotificationsAddr: testkit.PickUnusedAddress(t),
		StatusAddr:        testkit.PickUnusedAddress(t),
		CacheFreshTTL:     time.Minute,
		CacheStaleTTL:     5 * time.Minute,
	})

	recipientUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "degraded-recipient"))

	gameConn := dialRuntimeGRPC(t, fixture.grpcAddr)
	t.Cleanup(func() { _ = gameConn.Close() })

	socialConn := dialRuntimeGRPC(t, socialAddr)
	t.Cleanup(func() { _ = socialConn.Close() })

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
	setRecipientProfile(t, socialv1.NewSocialServiceClient(socialConn), recipientUserID, "Degraded Recipient")
	activeCampaignID, _ := createRecipientActiveCampaign(t, recipientGame)

	userhubClient := userhubv1.NewUserHubServiceClient(userhubConn)
	dashboard := waitForDashboard(t, userhubClient, recipientUserID, func(resp *userhubv1.GetDashboardResponse) bool {
		return resp.GetMetadata().GetDegraded() && !resp.GetNotifications().GetAvailable() && resp.GetCampaigns().GetListedCount() == 1
	})
	if !dashboard.GetMetadata().GetDegraded() {
		t.Fatal("dashboard degraded = false, want true")
	}
	if dashboard.GetNotifications().GetAvailable() {
		t.Fatal("dashboard notifications available = true, want false")
	}
	if got := dashboard.GetCampaigns().GetCampaigns()[0].GetCampaignId(); got != activeCampaignID {
		t.Fatalf("dashboard campaign id = %q, want %q", got, activeCampaignID)
	}
}

func waitForDiscoveryStarterTemplate(
	t *testing.T,
	dbPath string,
	entryIDs ...string,
) *discoverystorage.DiscoveryEntry {
	t.Helper()

	allowed := make(map[string]struct{}, len(entryIDs))
	for _, entryID := range entryIDs {
		allowed[entryID] = struct{}{}
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		store, err := discoverysqlite.Open(dbPath)
		if err == nil {
			for entryID := range allowed {
				entry, getErr := store.GetDiscoveryEntry(context.Background(), entryID)
				if getErr != nil {
					continue
				}
				if entry.SourceID != "" {
					_ = store.Close()
					return &entry
				}
			}
			_ = store.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for discovery starter template %v", entryIDs)
	return nil
}
