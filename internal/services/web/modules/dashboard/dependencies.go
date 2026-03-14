package dashboard

import (
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	userhubv1 "github.com/louisbranch/fracturing.space/api/gen/go/userhub/v1"
	grpc "google.golang.org/grpc"

	dashboardgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/dashboard/gateway"
)

// Dependencies contains dashboard feature clients.
type Dependencies struct {
	UserHubClient dashboardgateway.UserHubClient
	StatusClient  statusv1.StatusServiceClient
}

// BindUserHubDependency wires userhub-backed clients into the dashboard
// dependency set.
func BindUserHubDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.UserHubClient = userhubv1.NewUserHubServiceClient(conn)
}

// BindStatusDependency wires the status client into the dashboard dependency
// set.
func BindStatusDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.StatusClient = statusv1.NewStatusServiceClient(conn)
}
