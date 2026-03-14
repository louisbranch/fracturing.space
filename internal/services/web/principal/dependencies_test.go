package principal

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

func TestBindDependenciesWireOwnedPrincipalClients(t *testing.T) {
	t.Parallel()

	conn := &grpc.ClientConn{}
	deps := Dependencies{}

	BindAuthDependency(&deps, conn)
	if deps.SessionClient == nil {
		t.Fatal("SessionClient = nil, want client")
	}
	if deps.AccountClient == nil {
		t.Fatal("AccountClient = nil, want client")
	}

	BindSocialDependency(&deps, conn)
	if deps.SocialClient == nil {
		t.Fatal("SocialClient = nil, want client")
	}

	BindNotificationsDependency(&deps, conn)
	if deps.NotificationClient == nil {
		t.Fatal("NotificationClient = nil, want client")
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
	BindNotificationsDependency(nil, conn)
	BindNotificationsDependency(&deps, nil)

	if deps != (Dependencies{}) {
		t.Fatalf("Dependencies mutated with nil inputs: %+v", deps)
	}
}
