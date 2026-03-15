package web

import (
	"os"
	"strings"
	"testing"
)

func TestDependencyGraphKeepsClientBindingOutOfCommandLayer(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("dependency_graph.go")
	if err != nil {
		t.Fatalf("ReadFile(dependency_graph.go) error = %v", err)
	}
	source := string(content)

	for _, expected := range []string{
		"web.LookupStartupDependencyDescriptor",
		"web.DependencyNameAuth",
		"web.DependencyNameStatus",
	} {
		if !strings.Contains(source, expected) {
			t.Fatalf("dependency_graph.go missing %q", expected)
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
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("dependency_graph.go still contains %q", forbidden)
		}
	}
}
