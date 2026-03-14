package web

import (
	"github.com/louisbranch/fracturing.space/internal/services/web/modules"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
	grpc "google.golang.org/grpc"
)

// DependencyBundle is a single source of startup dependencies used by web service
// composition.
type DependencyBundle struct {
	// Principal carries the clients required for request-scoped principal resolution.
	Principal principal.Dependencies
	// Modules carries feature module dependencies and shared runtime config.
	Modules modules.Dependencies
}

// NewDependencyBundle returns a dependency bundle with shared runtime config
// pre-applied to both principal and module dependency sets.
func NewDependencyBundle(assetBaseURL string) DependencyBundle {
	return DependencyBundle{
		Principal: principal.NewDependencies(assetBaseURL),
		Modules:   modules.NewDependencies(assetBaseURL),
	}
}

// BindAuthDependency wires auth-backed clients into the web dependency bundle.
func BindAuthDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	principal.BindAuthDependency(&bundle.Principal, conn)
	modules.BindAuthDependency(&bundle.Modules, conn)
}

// BindSocialDependency wires social-backed clients into the web dependency bundle.
func BindSocialDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	principal.BindSocialDependency(&bundle.Principal, conn)
	modules.BindSocialDependency(&bundle.Modules, conn)
}

// BindGameDependency wires game-backed clients into the web dependency bundle.
func BindGameDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	modules.BindGameDependency(&bundle.Modules, conn)
}

// BindAIDependency wires AI-backed clients into the web dependency bundle.
func BindAIDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	modules.BindAIDependency(&bundle.Modules, conn)
}

// BindDiscoveryDependency wires discovery-backed clients into the web dependency bundle.
func BindDiscoveryDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	modules.BindDiscoveryDependency(&bundle.Modules, conn)
}

// BindUserHubDependency wires userhub-backed clients into the web dependency bundle.
func BindUserHubDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	modules.BindUserHubDependency(&bundle.Modules, conn)
}

// BindNotificationsDependency wires notification-backed clients into the web dependency bundle.
func BindNotificationsDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	principal.BindNotificationsDependency(&bundle.Principal, conn)
	modules.BindNotificationsDependency(&bundle.Modules, conn)
}

// BindStatusDependency wires the status client into the dashboard dependency set.
func BindStatusDependency(bundle *DependencyBundle, conn *grpc.ClientConn) {
	if bundle == nil || conn == nil {
		return
	}
	modules.BindStatusDependency(&bundle.Modules, conn)
}
