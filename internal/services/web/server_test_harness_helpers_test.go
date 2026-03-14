package web

import (
	"context"
	"net/http"
	"strings"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	invitegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/gateway"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

func assertPrimaryNavLinks(t *testing.T, body string) {
	t.Helper()
	for _, href := range []string{"/app/dashboard", "/app/campaigns", "/app/notifications", "/app/settings"} {
		if !strings.Contains(body, "href=\""+href+"\"") {
			t.Fatalf("body missing nav href %q", href)
		}
	}
	if !strings.Contains(body, `action="/logout"`) {
		t.Fatalf("body missing logout form action %q", "/logout")
	}
}

func attachSessionCookie(t *testing.T, req *http.Request, auth *fakeWebAuthClient, userID string) {
	t.Helper()
	if req == nil {
		t.Fatalf("request is required")
	}
	if auth == nil {
		t.Fatalf("auth client is required")
	}
	if strings.TrimSpace(userID) == "" {
		t.Fatalf("user id is required")
	}
	resp, err := auth.CreateWebSession(context.Background(), &authv1.CreateWebSessionRequest{UserId: userID})
	if err != nil {
		t.Fatalf("CreateWebSession() error = %v", err)
	}
	sessionID := strings.TrimSpace(resp.GetSession().GetId())
	if sessionID == "" {
		t.Fatalf("expected non-empty session id")
	}
	req.AddCookie(&http.Cookie{Name: "web_session", Value: sessionID})
}

func newDependencyBundle(principalDeps principal.Dependencies, moduleDeps modules.Dependencies) *DependencyBundle {
	moduleDeps = completeTestModuleDependencies(moduleDeps)
	return &DependencyBundle{
		Principal: principalDeps,
		Modules:   moduleDeps,
	}
}

func newDefaultDependencyBundle(moduleDeps modules.Dependencies) *DependencyBundle {
	return newDependencyBundle(principal.Dependencies{}, moduleDeps)
}

func completeTestModuleDependencies(moduleDeps modules.Dependencies) modules.Dependencies {
	hasCampaignDependency := moduleDeps.Campaigns.CampaignClient != nil ||
		moduleDeps.Campaigns.CommunicationClient != nil ||
		moduleDeps.Campaigns.ParticipantClient != nil ||
		moduleDeps.Campaigns.CharacterClient != nil ||
		moduleDeps.Campaigns.DaggerheartContentClient != nil ||
		moduleDeps.Campaigns.DaggerheartAssetClient != nil ||
		moduleDeps.Campaigns.SessionClient != nil ||
		moduleDeps.Campaigns.InviteClient != nil ||
		moduleDeps.Campaigns.SocialClient != nil ||
		moduleDeps.Campaigns.AuthClient != nil ||
		moduleDeps.Campaigns.AuthorizationClient != nil
	if hasCampaignDependency {
		if moduleDeps.Campaigns.CampaignClient == nil {
			moduleDeps.Campaigns.CampaignClient = defaultCampaignClient()
		}
		if moduleDeps.Campaigns.CommunicationClient == nil {
			moduleDeps.Campaigns.CommunicationClient = defaultCommunicationClient()
		}
		if moduleDeps.Campaigns.ParticipantClient == nil {
			moduleDeps.Campaigns.ParticipantClient = defaultParticipantClient()
		}
		if moduleDeps.Campaigns.CharacterClient == nil {
			moduleDeps.Campaigns.CharacterClient = defaultCharacterClient()
		}
		if moduleDeps.Campaigns.DaggerheartContentClient == nil {
			moduleDeps.Campaigns.DaggerheartContentClient = defaultDaggerheartContentClient()
		}
		if moduleDeps.Campaigns.DaggerheartAssetClient == nil {
			moduleDeps.Campaigns.DaggerheartAssetClient = defaultDaggerheartAssetClient()
		}
		if moduleDeps.Campaigns.SessionClient == nil {
			moduleDeps.Campaigns.SessionClient = defaultSessionClient()
		}
		if moduleDeps.Campaigns.InviteClient == nil {
			moduleDeps.Campaigns.InviteClient = defaultInviteClient()
		}
		if moduleDeps.Campaigns.SocialClient == nil {
			moduleDeps.Campaigns.SocialClient = defaultSocialClient()
		}
		if moduleDeps.Campaigns.AuthClient == nil {
			moduleDeps.Campaigns.AuthClient = newFakeWebAuthClient()
		}
		if moduleDeps.Campaigns.AuthorizationClient == nil {
			moduleDeps.Campaigns.AuthorizationClient = defaultAuthorizationClient()
		}
	}
	if moduleDeps.Invite.InviteClient == nil && moduleDeps.Campaigns.InviteClient != nil {
		moduleDeps.Invite.InviteClient = moduleDeps.Campaigns.InviteClient
	}
	if moduleDeps.Invite.AuthClient == nil {
		switch {
		case moduleDeps.Campaigns.AuthClient != nil:
			moduleDeps.Invite.AuthClient = moduleDeps.Campaigns.AuthClient
		case moduleDeps.PublicAuth.AuthClient != nil:
			authClient, ok := moduleDeps.PublicAuth.AuthClient.(invitegateway.AuthClient)
			if ok {
				moduleDeps.Invite.AuthClient = authClient
			}
		}
	}
	if moduleDeps.Invite.UserHubControlClient == nil && moduleDeps.DashboardSync.UserHubControlClient != nil {
		moduleDeps.Invite.UserHubControlClient = moduleDeps.DashboardSync.UserHubControlClient
	}
	if moduleDeps.Invite.GameEventClient == nil && moduleDeps.DashboardSync.GameEventClient != nil {
		moduleDeps.Invite.GameEventClient = moduleDeps.DashboardSync.GameEventClient
	}
	hasInviteDependency := moduleDeps.Invite.InviteClient != nil ||
		moduleDeps.Invite.AuthClient != nil ||
		moduleDeps.Invite.UserHubControlClient != nil ||
		moduleDeps.Invite.GameEventClient != nil
	if hasInviteDependency {
		if moduleDeps.Invite.InviteClient == nil {
			moduleDeps.Invite.InviteClient = defaultInviteClient()
		}
		if moduleDeps.Invite.AuthClient == nil {
			moduleDeps.Invite.AuthClient = newFakeWebAuthClient()
		}
	}
	return moduleDeps
}
