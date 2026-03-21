package web

import (
	"errors"
	"strings"
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

	BindInviteDependency(&bundle, conn)
	if bundle.Modules.Campaigns.InviteClient == nil {
		t.Fatal("Modules.Campaigns.InviteClient = nil, want client")
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
	BindInviteDependency(nil, conn)
	BindInviteDependency(&bundle, nil)
	BindStatusDependency(nil, conn)
	BindStatusDependency(&bundle, nil)
}

func TestStartupDependencyDescriptorsExposeStableBindings(t *testing.T) {
	t.Parallel()

	descriptors := StartupDependencyDescriptors()
	if len(descriptors) != 9 {
		t.Fatalf("descriptor count = %d, want 9", len(descriptors))
	}

	for _, name := range []string{
		DependencyNameAuth,
		DependencyNameSocial,
		DependencyNameGame,
		DependencyNameInvite,
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
		if descriptor.DefaultGRPCService == "" {
			t.Fatalf("descriptor %q has empty default gRPC service", name)
		}
		if descriptor.Capability == "" {
			t.Fatalf("descriptor %q has empty capability", name)
		}
		if len(descriptor.Surfaces) == 0 {
			t.Fatalf("descriptor %q has no owned surfaces", name)
		}
	}
}

func TestStartupDependencyDescriptorsReturnCopy(t *testing.T) {
	t.Parallel()

	descriptors := StartupDependencyDescriptors()
	descriptors[0].Name = "mutated"
	descriptors[0].Surfaces[0] = "mutated"

	lookup, ok := LookupStartupDependencyDescriptor(DependencyNameAuth)
	if !ok {
		t.Fatalf("LookupStartupDependencyDescriptor(%q) = false, want true", DependencyNameAuth)
	}
	if lookup.Name != DependencyNameAuth {
		t.Fatalf("LookupStartupDependencyDescriptor(%q).Name = %q, want %q", DependencyNameAuth, lookup.Name, DependencyNameAuth)
	}
	if lookup.Surfaces[0] != "principal" {
		t.Fatalf("LookupStartupDependencyDescriptor(%q).Surfaces[0] = %q, want %q", DependencyNameAuth, lookup.Surfaces[0], "principal")
	}
}

func TestLookupStartupDependencyDescriptorUnknown(t *testing.T) {
	t.Parallel()

	if descriptor, ok := LookupStartupDependencyDescriptor("missing"); ok {
		t.Fatalf("LookupStartupDependencyDescriptor(missing) = (%+v, true), want false", descriptor)
	}
}

func TestValidateRequiredDependencyBundleRequiresDependencies(t *testing.T) {
	t.Parallel()

	if err := validateRequiredDependencyBundle(nil); err == nil {
		t.Fatal("expected missing dependencies error")
	}
}

func TestValidateRequiredDependencyBundleAcceptsBootstrappedRequiredDependencies(t *testing.T) {
	t.Parallel()

	conn := &grpc.ClientConn{}
	bundle := NewDependencyBundle("")
	BindAuthDependency(&bundle, conn)
	BindSocialDependency(&bundle, conn)
	BindGameDependency(&bundle, conn)
	BindInviteDependency(&bundle, conn)

	if err := validateRequiredDependencyBundle(&bundle); err != nil {
		t.Fatalf("validateRequiredDependencyBundle() error = %v", err)
	}
}

func TestValidateRequiredDependencyBundleRejectsIncompleteRequiredDependency(t *testing.T) {
	t.Parallel()

	bundle := NewDependencyBundle("")
	BindAuthDependency(&bundle, &grpc.ClientConn{})
	BindGameDependency(&bundle, &grpc.ClientConn{})

	err := validateRequiredDependencyBundle(&bundle)
	if err == nil {
		t.Fatal("expected incomplete dependency error")
	}
	var validationErr StartupDependencyValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("validateRequiredDependencyBundle() error type = %T, want StartupDependencyValidationError", err)
	}
	if got := err.Error(); got == "" || !containsAll(got, []string{DependencyNameSocial, "principal.social"}) {
		t.Fatalf("validateRequiredDependencyBundle() error = %q, want social completeness detail", got)
	}
}

func TestValidateRequiredDependencyBundleReportsAllMissingDependencies(t *testing.T) {
	t.Parallel()

	bundle := NewDependencyBundle("")
	BindAuthDependency(&bundle, &grpc.ClientConn{})

	err := validateRequiredDependencyBundle(&bundle)
	if err == nil {
		t.Fatal("expected incomplete dependency error")
	}
	var validationErr StartupDependencyValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("validateRequiredDependencyBundle() error type = %T, want StartupDependencyValidationError", err)
	}
	if len(validationErr.Issues) != 3 {
		t.Fatalf("validation issue count = %d, want 3", len(validationErr.Issues))
	}
	gotSocial := false
	gotGame := false
	gotInvite := false
	for _, issue := range validationErr.Issues {
		switch issue.Name {
		case DependencyNameSocial:
			gotSocial = true
			if !containsAll(strings.Join(issue.Missing, ","), []string{"principal.social", "modules.profile.social"}) {
				t.Fatalf("social issue missing = %v, want principal.social and modules.profile.social", issue.Missing)
			}
		case DependencyNameGame:
			gotGame = true
			if !containsAll(strings.Join(issue.Missing, ","), []string{"modules.campaigns.campaign", "modules.campaigns.participant"}) {
				t.Fatalf("game issue missing = %v, want campaign and participant client", issue.Missing)
			}
		case DependencyNameInvite:
			gotInvite = true
			if !containsAll(strings.Join(issue.Missing, ","), []string{"modules.campaigns.invite", "modules.invite.invite"}) {
				t.Fatalf("invite issue missing = %v, want modules.campaigns.invite and modules.invite.invite", issue.Missing)
			}
		}
	}
	if !gotSocial {
		t.Fatalf("validation issues = %#v, want social issue", validationErr.Issues)
	}
	if !gotGame {
		t.Fatalf("validation issues = %#v, want game issue", validationErr.Issues)
	}
	if !gotInvite {
		t.Fatalf("validation issues = %#v, want invite issue", validationErr.Issues)
	}
}

func containsAll(value string, fragments []string) bool {
	for _, fragment := range fragments {
		if fragment == "" {
			continue
		}
		if !strings.Contains(value, fragment) {
			return false
		}
	}
	return true
}
