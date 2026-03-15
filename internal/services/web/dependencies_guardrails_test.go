package web

import (
	"os"
	"strings"
	"testing"
)

func TestDependenciesDelegateClientBindingToOwningPackages(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("dependencies.go")
	if err != nil {
		t.Fatalf("ReadFile(dependencies.go) error = %v", err)
	}
	source := string(content)

	for _, expected := range []string{
		"principal.NewDependencies(",
		"modules.NewDependencies(",
		"principal.BindAuthDependency(",
		"principal.BindSocialDependency(",
		"principal.BindNotificationsDependency(",
		"modules.BindAuthDependency(",
		"modules.BindSocialDependency(",
		"modules.BindGameDependency(",
		"modules.BindAIDependency(",
		"modules.BindDiscoveryDependency(",
		"modules.BindUserHubDependency(",
		"modules.BindNotificationsDependency(",
		"modules.BindStatusDependency(",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("dependencies.go missing %q", expected)
		}
	}

	for _, forbidden := range []string{
		"authv1.NewAuthServiceClient",
		"socialv1.NewSocialServiceClient",
		"statev1.NewCampaignServiceClient",
		"aiv1.NewAgentServiceClient",
		"discoveryv1.NewDiscoveryServiceClient",
		"userhubv1.NewUserHubServiceClient",
		"notificationsv1.NewNotificationServiceClient",
		"statusv1.NewStatusServiceClient",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("dependencies.go still contains %q", forbidden)
		}
	}
}

func TestStartupDependencyDescriptorsStayServiceOwned(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("startup_dependencies.go")
	if err != nil {
		t.Fatalf("ReadFile(startup_dependencies.go) error = %v", err)
	}
	source := string(content)

	for _, expected := range []string{
		"BindAuthDependency",
		"BindSocialDependency",
		"BindGameDependency",
		"BindAIDependency",
		"BindDiscoveryDependency",
		"BindUserHubDependency",
		"BindNotificationsDependency",
		"BindStatusDependency",
		"func StartupDependencyDescriptors()",
		"func LookupStartupDependencyDescriptor(",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("startup_dependencies.go missing %q", expected)
		}
	}

	for _, forbidden := range []string{
		"authv1.NewAuthServiceClient",
		"socialv1.NewSocialServiceClient",
		"statev1.NewCampaignServiceClient",
		"aiv1.NewAgentServiceClient",
		"discoveryv1.NewDiscoveryServiceClient",
		"userhubv1.NewUserHubServiceClient",
		"notificationsv1.NewNotificationServiceClient",
		"statusv1.NewStatusServiceClient",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("startup_dependencies.go still contains %q", forbidden)
		}
	}
}
