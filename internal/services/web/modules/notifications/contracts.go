package notifications

import (
	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
	notificationsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/gateway"
)

// NotificationSummary is the transport-facing alias for notifications app data.
type NotificationSummary = notificationsapp.NotificationSummary

// NotificationGateway is the transport-facing alias for notifications app gateway contract.
type NotificationGateway = notificationsapp.Gateway

// NotificationClient aliases keep root constructor/test seams stable.
type NotificationClient = notificationsgateway.NotificationClient

const (
	notificationSourceSystem  = notificationsgateway.NotificationSourceSystem
	notificationSourceUnknown = notificationsgateway.NotificationSourceUnknown
	notificationPageSize      = notificationsgateway.NotificationPageSize
	notificationMaxPages      = notificationsgateway.NotificationMaxPages
)
