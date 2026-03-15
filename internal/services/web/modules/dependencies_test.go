package modules

import (
	"testing"

	grpc "google.golang.org/grpc"
)

func TestNewDependenciesSetsSharedRuntimeConfig(t *testing.T) {
	t.Parallel()

	deps := NewDependencies("https://cdn.example.com/assets")
	if deps.AssetBaseURL != "https://cdn.example.com/assets" {
		t.Fatalf("AssetBaseURL = %q, want %q", deps.AssetBaseURL, "https://cdn.example.com/assets")
	}
}

func TestBindDependenciesWireOwnedModuleClients(t *testing.T) {
	t.Parallel()

	conn := &grpc.ClientConn{}
	deps := Dependencies{}

	BindAuthDependency(&deps, conn)
	if deps.PublicAuth.AuthClient == nil {
		t.Fatal("PublicAuth.AuthClient = nil, want client")
	}
	if deps.Campaigns.AuthClient == nil {
		t.Fatal("Campaigns.AuthClient = nil, want client")
	}
	if deps.Invite.AuthClient == nil {
		t.Fatal("Invite.AuthClient = nil, want client")
	}
	if deps.Profile.AuthClient == nil {
		t.Fatal("Profile.AuthClient = nil, want client")
	}
	if deps.Settings.AccountClient == nil {
		t.Fatal("Settings.AccountClient = nil, want client")
	}
	if deps.Settings.PasskeyClient == nil {
		t.Fatal("Settings.PasskeyClient = nil, want client")
	}

	BindSocialDependency(&deps, conn)
	if deps.Campaigns.SocialClient == nil {
		t.Fatal("Campaigns.SocialClient = nil, want client")
	}
	if deps.Profile.SocialClient == nil {
		t.Fatal("Profile.SocialClient = nil, want client")
	}
	if deps.Settings.SocialClient == nil {
		t.Fatal("Settings.SocialClient = nil, want client")
	}

	BindGameDependency(&deps, conn)
	if deps.Campaigns.CampaignClient == nil {
		t.Fatal("Campaigns.CampaignClient = nil, want client")
	}
	if deps.Campaigns.InteractionClient == nil {
		t.Fatal("Campaigns.InteractionClient = nil, want client")
	}
	if deps.Campaigns.ParticipantClient == nil {
		t.Fatal("Campaigns.ParticipantClient = nil, want client")
	}
	if deps.Campaigns.CharacterClient == nil {
		t.Fatal("Campaigns.CharacterClient = nil, want client")
	}
	if deps.Campaigns.DaggerheartContentClient == nil {
		t.Fatal("Campaigns.DaggerheartContentClient = nil, want client")
	}
	if deps.Campaigns.DaggerheartAssetClient == nil {
		t.Fatal("Campaigns.DaggerheartAssetClient = nil, want client")
	}
	if deps.Campaigns.SessionClient == nil {
		t.Fatal("Campaigns.SessionClient = nil, want client")
	}
	if deps.Campaigns.InviteClient == nil {
		t.Fatal("Campaigns.InviteClient = nil, want client")
	}
	if deps.Invite.InviteClient == nil {
		t.Fatal("Invite.InviteClient = nil, want client")
	}
	if deps.Campaigns.AuthorizationClient == nil {
		t.Fatal("Campaigns.AuthorizationClient = nil, want client")
	}
	if deps.DashboardSync.GameEventClient == nil {
		t.Fatal("DashboardSync.GameEventClient = nil, want client")
	}
	if deps.Invite.GameEventClient == nil {
		t.Fatal("Invite.GameEventClient = nil, want client")
	}

	BindAIDependency(&deps, conn)
	if deps.Settings.CredentialClient == nil {
		t.Fatal("Settings.CredentialClient = nil, want client")
	}
	if deps.Settings.AgentClient == nil {
		t.Fatal("Settings.AgentClient = nil, want client")
	}
	if deps.Campaigns.AgentClient == nil {
		t.Fatal("Campaigns.AgentClient = nil, want client")
	}

	BindDiscoveryDependency(&deps, conn)
	if deps.Discovery.DiscoveryClient == nil {
		t.Fatal("Discovery.DiscoveryClient = nil, want client")
	}
	if deps.Campaigns.DiscoveryClient == nil {
		t.Fatal("Campaigns.DiscoveryClient = nil, want client")
	}

	BindUserHubDependency(&deps, conn)
	if deps.Dashboard.UserHubClient == nil {
		t.Fatal("Dashboard.UserHubClient = nil, want client")
	}
	if deps.DashboardSync.UserHubControlClient == nil {
		t.Fatal("DashboardSync.UserHubControlClient = nil, want client")
	}
	if deps.Invite.UserHubControlClient == nil {
		t.Fatal("Invite.UserHubControlClient = nil, want client")
	}

	BindNotificationsDependency(&deps, conn)
	if deps.Notifications.NotificationClient == nil {
		t.Fatal("Notifications.NotificationClient = nil, want client")
	}

	BindStatusDependency(&deps, conn)
	if deps.Dashboard.StatusClient == nil {
		t.Fatal("Dashboard.StatusClient = nil, want client")
	}
}

func TestBindDependenciesIgnoreNilInputs(t *testing.T) {
	t.Parallel()

	conn := &grpc.ClientConn{}
	deps := Dependencies{}

	BindAuthDependency(nil, conn)
	BindAuthDependency(&deps, nil)
	BindSocialDependency(nil, conn)
	BindSocialDependency(&deps, nil)
	BindGameDependency(nil, conn)
	BindGameDependency(&deps, nil)
	BindAIDependency(nil, conn)
	BindAIDependency(&deps, nil)
	BindDiscoveryDependency(nil, conn)
	BindDiscoveryDependency(&deps, nil)
	BindUserHubDependency(nil, conn)
	BindUserHubDependency(&deps, nil)
	BindNotificationsDependency(nil, conn)
	BindNotificationsDependency(&deps, nil)
	BindStatusDependency(nil, conn)
	BindStatusDependency(&deps, nil)

	if deps != (Dependencies{}) {
		t.Fatalf("Dependencies mutated with nil inputs: %+v", deps)
	}
}
