package app

import (
	"context"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type gatewayStub struct {
	list []NotificationSummary
	item NotificationSummary
	err  error
}

func (g gatewayStub) ListNotifications(context.Context, string) ([]NotificationSummary, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.list, nil
}
func (g gatewayStub) GetNotification(context.Context, string, string) (NotificationSummary, error) {
	if g.err != nil {
		return NotificationSummary{}, g.err
	}
	return g.item, nil
}
func (g gatewayStub) OpenNotification(context.Context, string, string) (NotificationSummary, error) {
	if g.err != nil {
		return NotificationSummary{}, g.err
	}
	return g.item, nil
}

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)
	_, err := svc.ListNotifications(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestServiceRequiresUserID(t *testing.T) {
	t.Parallel()

	svc := NewService(gatewayStub{})
	_, err := svc.ListNotifications(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected user-id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusUnauthorized {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusUnauthorized)
	}
}

func TestGetAndOpenValidateNotificationID(t *testing.T) {
	t.Parallel()

	svc := NewService(gatewayStub{})
	_, err := svc.GetNotification(context.Background(), "user-1", "   ")
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
	_, err = svc.OpenNotification(context.Background(), "user-1", "   ")
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestGetAndOpenRequireGatewayReturnedID(t *testing.T) {
	t.Parallel()

	svc := NewService(gatewayStub{item: NotificationSummary{}})
	_, err := svc.GetNotification(context.Background(), "user-1", "n1")
	if err == nil {
		t.Fatalf("expected not-found error when gateway id is empty")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}

	_, err = svc.OpenNotification(context.Background(), "user-1", "n1")
	if err == nil {
		t.Fatalf("expected not-found error when gateway id is empty")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestListNotificationsReturnsEmptySliceForNilGatewayData(t *testing.T) {
	t.Parallel()

	svc := NewService(gatewayStub{list: nil})
	items, err := svc.ListNotifications(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListNotifications() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}
