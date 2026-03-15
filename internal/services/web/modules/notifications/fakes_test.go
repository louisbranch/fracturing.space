package notifications

import (
	"context"

	notificationsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/notifications/app"
)

// fakeGateway implements notificationsapp.Gateway for tests with configurable
// return values and error injection.
// values and error injection.
type fakeGateway struct {
	listItems []notificationsapp.NotificationSummary
	listErr   error
	getItem   notificationsapp.NotificationSummary
	getErr    error
	openItem  notificationsapp.NotificationSummary
	openErr   error
}

var _ notificationsapp.Gateway = fakeGateway{}

func (f fakeGateway) ListNotifications(context.Context, string) ([]notificationsapp.NotificationSummary, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listItems == nil {
		return []notificationsapp.NotificationSummary{testNotificationSummary("n1", false)}, nil
	}
	return f.listItems, nil
}

func (f fakeGateway) GetNotification(context.Context, string, string) (notificationsapp.NotificationSummary, error) {
	if f.getErr != nil {
		return notificationsapp.NotificationSummary{}, f.getErr
	}
	if f.getItem != (notificationsapp.NotificationSummary{}) {
		return f.getItem, nil
	}
	if len(f.listItems) > 0 {
		return f.listItems[0], nil
	}
	return testNotificationSummary("n1", false), nil
}

func (f fakeGateway) OpenNotification(context.Context, string, string) (notificationsapp.NotificationSummary, error) {
	if f.openErr != nil {
		return notificationsapp.NotificationSummary{}, f.openErr
	}
	if f.openItem != (notificationsapp.NotificationSummary{}) {
		return f.openItem, nil
	}
	return testNotificationSummary("n1", true), nil
}

func testNotificationSummary(id string, read bool) notificationsapp.NotificationSummary {
	return notificationsapp.NotificationSummary{
		ID:          id,
		MessageType: "system.message.v1",
		PayloadJSON: `{"title":"Welcome to Fracturing Space","body":"Your account is ready. Sign-in method: passkey."}`,
		Read:        read,
	}
}
