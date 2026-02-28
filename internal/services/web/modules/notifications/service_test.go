package notifications

import (
	"context"
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := newService(nil)
	_, err := svc.listNotifications(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected unavailable error for listNotifications")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}

	_, err = svc.getNotification(context.Background(), "user-1", "n-1")
	if err == nil {
		t.Fatalf("expected unavailable error for getNotification")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}

	_, err = svc.openNotification(context.Background(), "user-1", "n-1")
	if err == nil {
		t.Fatalf("expected unavailable error for openNotification")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestServiceRequiresExplicitUserID(t *testing.T) {
	t.Parallel()

	svc := newService(notificationGatewayStub{})
	_, err := svc.listNotifications(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected user-id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusUnauthorized {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusUnauthorized)
	}
}

func TestListNotificationsReturnsEmptySliceForEmptyList(t *testing.T) {
	t.Parallel()

	svc := newService(notificationGatewayStub{items: []NotificationSummary{}})
	items, err := svc.listNotifications(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("listNotifications() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func TestListNotificationsPropagatesGatewayError(t *testing.T) {
	t.Parallel()

	svc := newService(notificationGatewayStub{listErr: errors.New("boom")})
	_, err := svc.listNotifications(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected list error")
	}
	if err.Error() != "boom" {
		t.Fatalf("err = %q, want %q", err.Error(), "boom")
	}
}

func TestGetNotificationReturnsNotFoundWhenIDMissing(t *testing.T) {
	t.Parallel()

	svc := newService(notificationGatewayStub{})
	_, err := svc.getNotification(context.Background(), "user-1", "   ")
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestOpenNotificationReturnsNotFoundWhenIDMissing(t *testing.T) {
	t.Parallel()

	svc := newService(notificationGatewayStub{})
	_, err := svc.openNotification(context.Background(), "user-1", "   ")
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestOpenNotificationPropagatesGatewayError(t *testing.T) {
	t.Parallel()

	svc := newService(notificationGatewayStub{openErr: errors.New("unavailable")})
	_, err := svc.openNotification(context.Background(), "user-1", "n1")
	if err == nil {
		t.Fatalf("expected open error")
	}
	if err.Error() != "unavailable" {
		t.Fatalf("err = %q, want %q", err.Error(), "unavailable")
	}
}

type notificationGatewayStub struct {
	items    []NotificationSummary
	listErr  error
	getItem  NotificationSummary
	getErr   error
	openItem NotificationSummary
	openErr  error
}

func (f notificationGatewayStub) ListNotifications(context.Context, string) ([]NotificationSummary, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.items == nil {
		return []NotificationSummary{{ID: "n-1", MessageType: "auth.onboarding.welcome"}}, nil
	}
	return f.items, nil
}

func (f notificationGatewayStub) GetNotification(context.Context, string, string) (NotificationSummary, error) {
	if f.getErr != nil {
		return NotificationSummary{}, f.getErr
	}
	if f.getItem == (NotificationSummary{}) {
		return NotificationSummary{ID: "n-1", MessageType: "auth.onboarding.welcome"}, nil
	}
	return f.getItem, nil
}

func (f notificationGatewayStub) OpenNotification(context.Context, string, string) (NotificationSummary, error) {
	if f.openErr != nil {
		return NotificationSummary{}, f.openErr
	}
	if f.openItem == (NotificationSummary{}) {
		return NotificationSummary{ID: "n-1", MessageType: "auth.onboarding.welcome", Read: true}, nil
	}
	return f.openItem, nil
}
