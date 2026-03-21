package invite

import (
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	invitegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/gateway"
	grpc "google.golang.org/grpc"
)

// Dependencies contains public invite reads/mutations.
type Dependencies struct {
	InviteClient invitegateway.InviteClient
	AuthClient   invitegateway.AuthClient
}

// BindAuthDependency wires auth-backed clients into the invite dependency set.
func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.AuthClient = authv1.NewAuthServiceClient(conn)
}

// BindInviteDependency wires invite-service clients into the invite dependency set.
func BindInviteDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.InviteClient = invitev1.NewInviteServiceClient(conn)
}
