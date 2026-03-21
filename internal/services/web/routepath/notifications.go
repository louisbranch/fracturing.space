package routepath

const (
	AppNotifications           = "/app/notifications"
	Notifications              = "/app/notifications/"
	AppNotificationPattern     = Notifications + "{notificationID}"
	AppNotificationOpenPattern = Notifications + "{notificationID}/open"
	AppNotificationRestPattern = Notifications + "{notificationID}/{rest...}"
)

// AppNotification returns the notification detail route.
func AppNotification(notificationID string) string {
	return Notifications + escapeSegment(notificationID)
}

// AppNotificationOpen returns the notification open-and-acknowledge route.
func AppNotificationOpen(notificationID string) string {
	return AppNotification(notificationID) + "/open"
}
