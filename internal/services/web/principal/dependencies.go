package principal

import (
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	grpc "google.golang.org/grpc"
)

// NewDependencies returns principal dependency defaults with shared runtime
// configuration applied.
func NewDependencies(assetBaseURL string) Dependencies {
	return Dependencies{AssetBaseURL: assetBaseURL}
}

// BindAuthDependency wires auth-backed clients into the principal dependency
// set.
func BindAuthDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.SessionClient = authv1.NewAuthServiceClient(conn)
	deps.AccountClient = authv1.NewAccountServiceClient(conn)
}

// BindSocialDependency wires social-backed clients into the principal
// dependency set.
func BindSocialDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.SocialClient = socialv1.NewSocialServiceClient(conn)
}

// BindNotificationsDependency wires notification-backed clients into the
// principal dependency set.
func BindNotificationsDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.NotificationClient = notificationsv1.NewNotificationServiceClient(conn)
}
