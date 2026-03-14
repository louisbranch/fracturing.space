package publicauth

import (
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	grpc "google.golang.org/grpc"

	publicauthgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/publicauth/gateway"
)

// Dependencies contains public-auth feature clients and runtime config.
type Dependencies struct {
	AuthClient  publicauthgateway.AuthClient
	AuthBaseURL string
}

// BindAuthDependency wires auth-backed clients into the public-auth dependency
// set.
func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.AuthClient = authv1.NewAuthServiceClient(conn)
}
