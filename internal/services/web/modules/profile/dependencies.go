package profile

import (
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	grpc "google.golang.org/grpc"

	profilegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/profile/gateway"
)

// Dependencies contains public profile feature clients.
type Dependencies struct {
	AuthClient   profilegateway.AuthClient
	SocialClient profilegateway.SocialClient
}

// BindAuthDependency wires auth-backed clients into the profile dependency set.
func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.AuthClient = authv1.NewAuthServiceClient(conn)
}

// BindSocialDependency wires social-backed clients into the profile dependency
// set.
func BindSocialDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.SocialClient = socialv1.NewSocialServiceClient(conn)
}
