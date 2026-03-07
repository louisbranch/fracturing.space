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

	lastUserID       string
	lastNotification string
	getCalls         int
	openCalls        int
	listCalls        int
}

func (g *gatewayStub) ListNotifications(_ context.Context, userID string) ([]NotificationSummary, error) {
	g.listCalls++
	g.lastUserID = userID
	if g.err != nil {
		return nil, g.err
	}
	return g.list, nil
}
func (g *gatewayStub) GetNotification(_ context.Context, userID string, notificationID string) (NotificationSummary, error) {
	g.getCalls++
	g.lastUserID = userID
	g.lastNotification = notificationID
	if g.err != nil {
		return NotificationSummary{}, g.err
	}
	return g.item, nil
}
func (g *gatewayStub) OpenNotification(_ context.Context, userID string, notificationID string) (NotificationSummary, error) {
	g.openCalls++
	g.lastUserID = userID
	g.lastNotification = notificationID
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

func TestUnavailableGatewayFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := NewUnavailableGateway()
	if IsGatewayHealthy(nil) {
		t.Fatalf("IsGatewayHealthy(nil) = true, want false")
	}
	if IsGatewayHealthy(gateway) {
		t.Fatalf("IsGatewayHealthy(unavailable) = true, want false")
	}
	if !IsGatewayHealthy(&gatewayStub{}) {
		t.Fatalf("IsGatewayHealthy(stub) = false, want true")
	}

	ctx := context.Background()
	if list, err := gateway.ListNotifications(ctx, "user-1"); err == nil {
		t.Fatalf("ListNotifications() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("ListNotifications() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if list != nil {
		t.Fatalf("ListNotifications() list = %+v, want nil", list)
	}
	if item, err := gateway.GetNotification(ctx, "user-1", "n1"); err == nil {
		t.Fatalf("GetNotification() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("GetNotification() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if item != (NotificationSummary{}) {
		t.Fatalf("GetNotification() item = %+v, want zero value", item)
	}
	if item, err := gateway.OpenNotification(ctx, "user-1", "n1"); err == nil {
		t.Fatalf("OpenNotification() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("OpenNotification() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if item != (NotificationSummary{}) {
		t.Fatalf("OpenNotification() item = %+v, want zero value", item)
	}
}

func TestServiceRequiresUserID(t *testing.T) {
	t.Parallel()

	svc := NewService(&gatewayStub{})
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

	svc := NewService(&gatewayStub{})
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

	svc := NewService(&gatewayStub{item: NotificationSummary{}})
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

	svc := NewService(&gatewayStub{list: nil})
	items, err := svc.ListNotifications(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListNotifications() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func TestGetAndOpenNormalizeDelegatedIDs(t *testing.T) {
	t.Parallel()

	gateway := &gatewayStub{item: NotificationSummary{ID: "n1"}}
	svc := NewService(gateway)

	if _, err := svc.GetNotification(context.Background(), " user-1 ", " n1 "); err != nil {
		t.Fatalf("GetNotification() error = %v", err)
	}
	if gateway.getCalls != 1 {
		t.Fatalf("get calls = %d, want 1", gateway.getCalls)
	}
	if gateway.lastUserID != "user-1" {
		t.Fatalf("delegated user id = %q, want %q", gateway.lastUserID, "user-1")
	}
	if gateway.lastNotification != "n1" {
		t.Fatalf("delegated notification id = %q, want %q", gateway.lastNotification, "n1")
	}

	if _, err := svc.OpenNotification(context.Background(), " user-1 ", " n1 "); err != nil {
		t.Fatalf("OpenNotification() error = %v", err)
	}
	if gateway.openCalls != 1 {
		t.Fatalf("open calls = %d, want 1", gateway.openCalls)
	}
	if gateway.lastUserID != "user-1" {
		t.Fatalf("delegated user id = %q, want %q", gateway.lastUserID, "user-1")
	}
	if gateway.lastNotification != "n1" {
		t.Fatalf("delegated notification id = %q, want %q", gateway.lastNotification, "n1")
	}
}
