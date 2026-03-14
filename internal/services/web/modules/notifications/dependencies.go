package notifications

import (
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	grpc "google.golang.org/grpc"

	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
)

// Dependencies contains notifications feature clients.
type Dependencies struct {
	NotificationClient notificationsgateway.NotificationClient
}

// BindDependency wires notification-backed clients into the notifications
// dependency set.
func BindDependency(deps *Dependencies, conn *grpc.ClientConn) {
	if deps == nil || conn == nil {
		return
	}
	deps.NotificationClient = notificationsv1.NewNotificationServiceClient(conn)
}
