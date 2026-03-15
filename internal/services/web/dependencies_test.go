package web

import (
	"testing"

	grpc "google.golang.org/grpc"
)

func TestNewDependencyBundleAppliesSharedRuntimeConfig(t *testing.T) {
	t.Parallel()

	bundle := NewDependencyBundle("https://cdn.example.com/assets")
	if bundle.Principal.AssetBaseURL != "https://cdn.example.com/assets" {
		t.Fatalf("Principal.AssetBaseURL = %q, want %q", bundle.Principal.AssetBaseURL, "https://cdn.example.com/assets")
	}
	if bundle.Modules.AssetBaseURL != "https://cdn.example.com/assets" {
		t.Fatalf("Modules.AssetBaseURL = %q, want %q", bundle.Modules.AssetBaseURL, "https://cdn.example.com/assets")
	}
}

func TestDependencyBindersWireRootBundle(t *testing.T) {
	t.Parallel()

	conn := &grpc.ClientConn{}
	bundle := NewDependencyBundle("")

	BindAuthDependency(&bundle, conn)
	if bundle.Principal.SessionClient == nil {
		t.Fatal("Principal.SessionClient = nil, want client")
	}
	if bundle.Modules.PublicAuth.AuthClient == nil {
		t.Fatal("Modules.PublicAuth.AuthClient = nil, want client")
	}

	BindSocialDependency(&bundle, conn)
	if bundle.Principal.SocialClient == nil {
		t.Fatal("Principal.SocialClient = nil, want client")
	}
	if bundle.Modules.Profile.SocialClient == nil {
		t.Fatal("Modules.Profile.SocialClient = nil, want client")
	}

	BindGameDependency(&bundle, conn)
	if bundle.Modules.Campaigns.CampaignClient == nil {
		t.Fatal("Modules.Campaigns.CampaignClient = nil, want client")
	}
	if bundle.Modules.Invite.InviteClient == nil {
		t.Fatal("Modules.Invite.InviteClient = nil, want client")
	}

	BindAIDependency(&bundle, conn)
	if bundle.Modules.Settings.AgentClient == nil {
		t.Fatal("Modules.Settings.AgentClient = nil, want client")
	}
	if bundle.Modules.Campaigns.AgentClient == nil {
		t.Fatal("Modules.Campaigns.AgentClient = nil, want client")
	}

	BindDiscoveryDependency(&bundle, conn)
	if bundle.Modules.Discovery.DiscoveryClient == nil {
		t.Fatal("Modules.Discovery.DiscoveryClient = nil, want client")
	}

	BindUserHubDependency(&bundle, conn)
	if bundle.Modules.Dashboard.UserHubClient == nil {
		t.Fatal("Modules.Dashboard.UserHubClient = nil, want client")
	}

	BindNotificationsDependency(&bundle, conn)
	if bundle.Principal.NotificationClient == nil {
		t.Fatal("Principal.NotificationClient = nil, want client")
	}
	if bundle.Modules.Notifications.NotificationClient == nil {
		t.Fatal("Modules.Notifications.NotificationClient = nil, want client")
	}

	BindStatusDependency(&bundle, conn)
	if bundle.Modules.Dashboard.StatusClient == nil {
		t.Fatal("Modules.Dashboard.StatusClient = nil, want client")
	}
}

func TestDependencyBindersIgnoreNilInputs(t *testing.T) {
	t.Parallel()

	conn := &grpc.ClientConn{}
	bundle := NewDependencyBundle("")

	BindAuthDependency(nil, conn)
	BindAuthDependency(&bundle, nil)
	BindSocialDependency(nil, conn)
	BindSocialDependency(&bundle, nil)
	BindGameDependency(nil, conn)
	BindGameDependency(&bundle, nil)
	BindAIDependency(nil, conn)
	BindAIDependency(&bundle, nil)
	BindDiscoveryDependency(nil, conn)
	BindDiscoveryDependency(&bundle, nil)
	BindUserHubDependency(nil, conn)
	BindUserHubDependency(&bundle, nil)
	BindNotificationsDependency(nil, conn)
	BindNotificationsDependency(&bundle, nil)
	BindStatusDependency(nil, conn)
	BindStatusDependency(&bundle, nil)
}

func TestStartupDependencyDescriptorsExposeStableBindings(t *testing.T) {
	t.Parallel()

	descriptors := StartupDependencyDescriptors()
	if len(descriptors) != 8 {
		t.Fatalf("descriptor count = %d, want 8", len(descriptors))
	}

	for _, name := range []string{
		DependencyNameAuth,
		DependencyNameSocial,
		DependencyNameGame,
		DependencyNameAI,
		DependencyNameDiscovery,
		DependencyNameUserHub,
		DependencyNameNotifications,
		DependencyNameStatus,
	} {
		descriptor, ok := LookupStartupDependencyDescriptor(name)
		if !ok {
			t.Fatalf("LookupStartupDependencyDescriptor(%q) = false, want true", name)
		}
		if descriptor.Bind == nil {
			t.Fatalf("descriptor %q has nil binder", name)
		}
	}
}

func TestStartupDependencyDescriptorsReturnCopy(t *testing.T) {
	t.Parallel()

	descriptors := StartupDependencyDescriptors()
	descriptors[0].Name = "mutated"

	lookup, ok := LookupStartupDependencyDescriptor(DependencyNameAuth)
	if !ok {
		t.Fatalf("LookupStartupDependencyDescriptor(%q) = false, want true", DependencyNameAuth)
	}
	if lookup.Name != DependencyNameAuth {
		t.Fatalf("LookupStartupDependencyDescriptor(%q).Name = %q, want %q", DependencyNameAuth, lookup.Name, DependencyNameAuth)
	}
}

func TestLookupStartupDependencyDescriptorUnknown(t *testing.T) {
	t.Parallel()

	if descriptor, ok := LookupStartupDependencyDescriptor("missing"); ok {
		t.Fatalf("LookupStartupDependencyDescriptor(missing) = (%+v, true), want false", descriptor)
	}
}
