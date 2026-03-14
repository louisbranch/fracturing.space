package discovery

import (
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	grpc "google.golang.org/grpc"

	discoverygateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/gateway"
)

// Dependencies contains discovery feature clients.
type Dependencies struct {
	DiscoveryClient discoverygateway.DiscoveryClient
}

// BindDependency wires discovery-backed clients into the discovery dependency
// set.
func BindDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.DiscoveryClient = discoveryv1.NewDiscoveryServiceClient(conn)
}
