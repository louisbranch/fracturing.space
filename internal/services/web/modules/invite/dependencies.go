package invite

import (
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpc "google.golang.org/grpc"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	invitegateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/invite/gateway"
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

// BindGameDependency wires game-backed clients into the invite dependency set.
func BindGameDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.InviteClient = statev1.NewInviteServiceClient(conn)
}
