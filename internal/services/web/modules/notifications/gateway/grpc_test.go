package gateway

import (
	"context"
	"net/http"
	"testing"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type clientStub struct {
	lastReq    *notificationsv1.ListNotificationsRequest
	lastUserID string
}

func (c *clientStub) ListNotifications(ctx context.Context, req *notificationsv1.ListNotificationsRequest, _ ...grpc.CallOption) (*notificationsv1.ListNotificationsResponse, error) {
	c.lastReq = req
	if md, ok := metadata.FromOutgoingContext(ctx); ok {
		ids := md.Get(grpcmeta.UserIDHeader)
		if len(ids) > 0 {
			c.lastUserID = ids[0]
		}
	}
	return &notificationsv1.ListNotificationsResponse{Notifications: []*notificationsv1.Notification{{
		Id:        "n1",
		Source:    notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
		CreatedAt: timestamppb.New(time.Now().UTC()),
	}}}, nil
}
func (*clientStub) GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
	return &notificationsv1.GetUnreadNotificationStatusResponse{}, nil
}
func (*clientStub) GetNotification(context.Context, *notificationsv1.GetNotificationRequest, ...grpc.CallOption) (*notificationsv1.GetNotificationResponse, error) {
	return &notificationsv1.GetNotificationResponse{Notification: &notificationsv1.Notification{Id: "n1"}}, nil
}
func (*clientStub) MarkNotificationRead(context.Context, *notificationsv1.MarkNotificationReadRequest, ...grpc.CallOption) (*notificationsv1.MarkNotificationReadResponse, error) {
	return &notificationsv1.MarkNotificationReadResponse{Notification: &notificationsv1.Notification{Id: "n1"}}, nil
}

func TestNewGRPCGatewayWithoutClientFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(nil)
	_, err := gateway.ListNotifications(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestGRPCGatewayListNotificationsMapsMetadataAndConstants(t *testing.T) {
	t.Parallel()

	client := &clientStub{}
	gateway := GRPCGateway{Client: client}
	items, err := gateway.ListNotifications(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListNotifications() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Source != NotificationSourceSystem {
		t.Fatalf("source = %q, want %q", items[0].Source, NotificationSourceSystem)
	}
	if client.lastReq.GetPageSize() != NotificationPageSize {
		t.Fatalf("PageSize = %d, want %d", client.lastReq.GetPageSize(), NotificationPageSize)
	}
	if client.lastUserID != "user-1" {
		t.Fatalf("user id = %q, want %q", client.lastUserID, "user-1")
	}
}

func TestGRPCGatewayGetAndOpenNotification(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{Client: &clientStub{}}
	item, err := gateway.GetNotification(context.Background(), "user-1", "n1")
	if err != nil {
		t.Fatalf("GetNotification() error = %v", err)
	}
	if item.ID != "n1" {
		t.Fatalf("GetNotification id = %q, want %q", item.ID, "n1")
	}
	item, err = gateway.OpenNotification(context.Background(), "user-1", "n1")
	if err != nil {
		t.Fatalf("OpenNotification() error = %v", err)
	}
	if item.ID != "n1" {
		t.Fatalf("OpenNotification id = %q, want %q", item.ID, "n1")
	}
}
