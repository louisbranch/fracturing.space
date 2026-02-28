package notifications

import "context"

// fakeGateway implements NotificationGateway for tests with configurable return
// values and error injection.
type fakeGateway struct {
	listItems []NotificationSummary
	listErr   error
	getItem   NotificationSummary
	getErr    error
	openItem  NotificationSummary
	openErr   error
}

var _ NotificationGateway = fakeGateway{}

func (f fakeGateway) ListNotifications(context.Context, string) ([]NotificationSummary, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listItems == nil {
		return []NotificationSummary{{ID: "n1", MessageType: "auth.onboarding.welcome", Read: false}}, nil
	}
	return f.listItems, nil
}

func (f fakeGateway) GetNotification(context.Context, string, string) (NotificationSummary, error) {
	if f.getErr != nil {
		return NotificationSummary{}, f.getErr
	}
	if f.getItem != (NotificationSummary{}) {
		return f.getItem, nil
	}
	if len(f.listItems) > 0 {
		return f.listItems[0], nil
	}
	return NotificationSummary{ID: "n1", MessageType: "auth.onboarding.welcome", Read: false}, nil
}

func (f fakeGateway) OpenNotification(context.Context, string, string) (NotificationSummary, error) {
	if f.openErr != nil {
		return NotificationSummary{}, f.openErr
	}
	if f.openItem != (NotificationSummary{}) {
		return f.openItem, nil
	}
	return NotificationSummary{ID: "n1", MessageType: "auth.onboarding.welcome", Read: true}, nil
}
