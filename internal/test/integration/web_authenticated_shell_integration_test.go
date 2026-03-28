//go:build integration

package integration

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/playtest"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/webtest"
	"github.com/louisbranch/fracturing.space/internal/test/testkit"
)

func TestWebAuthenticatedShellIntegration(t *testing.T) {
	fixture := newSuiteFixture(t)

	socialAddr := fixture.startSocialServer(t)
	notificationsAddr := fixture.startNotificationsServer(t)
	inviteAddr := fixture.startInviteServer(t)
	userhubAddr := fixture.startUserHubServer(t)

	play := playtest.StartRuntime(t, fixture.authAddr, fixture.grpcAddr)
	web := webtest.StartRuntime(t, webtest.RuntimeConfig{
		PlayHTTPAddr:      strings.TrimPrefix(play.BaseURL, "http://"),
		AuthAddr:          fixture.authAddr,
		SocialAddr:        socialAddr,
		GameAddr:          fixture.grpcAddr,
		InviteAddr:        inviteAddr,
		NotificationsAddr: notificationsAddr,
		UserHubAddr:       userhubAddr,
	})

	recipientUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "web-recipient"))
	ownerUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "web-owner"))
	webSessionID := testkit.CreateAuthWebSession(t, fixture.authAddr, recipientUserID)

	recipientGame := fixture.newGameSuite(t, recipientUserID)

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

	setRecipientProfile(t, socialClient, recipientUserID, "Web Recipient")
	activeCampaignID, activeSessionID := createRecipientActiveCampaign(t, recipientGame)
	_ = createPendingInviteForRecipient(t, ownerInviteSuite, recipientUserID)
	createSystemNotification(t, notificationsClient, recipientUserID, "web.integration.seed", `{"kind":"seed"}`, "web-seed-1")

	waitForDashboard(t, userhubClient, recipientUserID, func(resp *userhubv1.GetDashboardResponse) bool {
		return resp.GetInvites().GetListedCount() == 1 &&
			resp.GetNotifications().GetUnreadCount() == 1 &&
			resp.GetActiveSessions().GetListedCount() == 1
	})

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	dashboardReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, web.BaseURL+"/app/dashboard", nil)
	if err != nil {
		t.Fatalf("create dashboard request: %v", err)
	}
	dashboardReq.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: webSessionID})

	dashboardResp, err := httpClient.Do(dashboardReq)
	if err != nil {
		t.Fatalf("get dashboard: %v", err)
	}
	defer dashboardResp.Body.Close()

	dashboardBodyBytes, err := io.ReadAll(dashboardResp.Body)
	if err != nil {
		t.Fatalf("read dashboard body: %v", err)
	}
	dashboardBody := string(dashboardBodyBytes)

	if dashboardResp.StatusCode != http.StatusOK {
		t.Fatalf("dashboard status = %d, want %d", dashboardResp.StatusCode, http.StatusOK)
	}
	for _, fragment := range []string{
		`href="/app/dashboard"`,
		`href="/app/campaigns"`,
		`href="/app/notifications"`,
		`action="/logout"`,
		`data-dashboard-block="pending-invites"`,
		`data-dashboard-block="active-sessions"`,
		"Dashboard Session",
	} {
		if !strings.Contains(dashboardBody, fragment) {
			t.Fatalf("dashboard body missing %q", fragment)
		}
	}

	launchReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, web.BaseURL+"/app/campaigns/"+url.PathEscape(activeCampaignID)+"/game", nil)
	if err != nil {
		t.Fatalf("create web launch request: %v", err)
	}
	launchReq.AddCookie(&http.Cookie{Name: sessioncookie.Name, Value: webSessionID})

	launchResp, err := httpClient.Do(launchReq)
	if err != nil {
		t.Fatalf("launch game from web: %v", err)
	}
	defer launchResp.Body.Close()

	if launchResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("web launch status = %d, want %d", launchResp.StatusCode, http.StatusSeeOther)
	}

	playLaunchURL := launchResp.Header.Get("Location")
	if strings.TrimSpace(playLaunchURL) == "" {
		t.Fatal("web launch location is empty")
	}

	parsedLaunchURL, err := url.Parse(playLaunchURL)
	if err != nil {
		t.Fatalf("parse web launch location: %v", err)
	}
	if got, want := parsedLaunchURL.Scheme, "http"; got != want {
		t.Fatalf("web launch scheme = %q, want %q", got, want)
	}
	if got, want := parsedLaunchURL.Host, strings.TrimPrefix(play.BaseURL, "http://"); got != want {
		t.Fatalf("web launch host = %q, want %q", got, want)
	}
	if got, want := parsedLaunchURL.Path, "/campaigns/"+activeCampaignID; got != want {
		t.Fatalf("web launch path = %q, want %q", got, want)
	}
	if got := strings.TrimSpace(parsedLaunchURL.Query().Get("launch")); got == "" {
		t.Fatal("web launch query missing launch grant")
	}

	playLaunchResp, err := httpClient.Get(playLaunchURL)
	if err != nil {
		t.Fatalf("follow play launch url: %v", err)
	}
	defer playLaunchResp.Body.Close()

	if playLaunchResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("play launch status = %d, want %d", playLaunchResp.StatusCode, http.StatusSeeOther)
	}
	if got, want := playLaunchResp.Header.Get("Location"), "/campaigns/"+activeCampaignID; got != want {
		t.Fatalf("play launch location = %q, want %q", got, want)
	}

	playSessionID := playtest.RequireCookieValue(t, playLaunchResp.Cookies(), "play_session")

	playShellReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, play.BaseURL+"/campaigns/"+url.PathEscape(activeCampaignID), nil)
	if err != nil {
		t.Fatalf("create play shell request: %v", err)
	}
	playShellReq.AddCookie(&http.Cookie{Name: "play_session", Value: playSessionID})

	playShellResp, err := httpClient.Do(playShellReq)
	if err != nil {
		t.Fatalf("load play shell: %v", err)
	}
	defer playShellResp.Body.Close()

	playShellBodyBytes, err := io.ReadAll(playShellResp.Body)
	if err != nil {
		t.Fatalf("read play shell body: %v", err)
	}
	playShellBody := string(playShellBodyBytes)

	if playShellResp.StatusCode != http.StatusOK {
		t.Fatalf("play shell status = %d, want %d", playShellResp.StatusCode, http.StatusOK)
	}
	if !strings.Contains(playShellBody, "/api/campaigns/"+activeCampaignID+"/bootstrap") {
		t.Fatalf("play shell body missing bootstrap url for campaign %q", activeCampaignID)
	}

	bootstrapReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, play.BaseURL+"/api/campaigns/"+url.PathEscape(activeCampaignID)+"/bootstrap", nil)
	if err != nil {
		t.Fatalf("create play bootstrap request: %v", err)
	}
	bootstrapReq.AddCookie(&http.Cookie{Name: "play_session", Value: playSessionID})

	bootstrapResp, err := httpClient.Do(bootstrapReq)
	if err != nil {
		t.Fatalf("load play bootstrap: %v", err)
	}
	defer bootstrapResp.Body.Close()

	if bootstrapResp.StatusCode != http.StatusOK {
		t.Fatalf("play bootstrap status = %d, want %d", bootstrapResp.StatusCode, http.StatusOK)
	}

	var bootstrap playprotocol.Bootstrap
	if err := decodeHTTPJSON(bootstrapResp, &bootstrap); err != nil {
		t.Fatalf("decode play bootstrap: %v", err)
	}
	if bootstrap.CampaignID != activeCampaignID {
		t.Fatalf("bootstrap campaign id = %q, want %q", bootstrap.CampaignID, activeCampaignID)
	}
	if bootstrap.InteractionState.ActiveSession == nil || bootstrap.InteractionState.ActiveSession.SessionID != activeSessionID {
		t.Fatalf("bootstrap active session = %#v, want %q", bootstrap.InteractionState.ActiveSession, activeSessionID)
	}
}
