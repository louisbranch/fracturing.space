package notifications

import (
	"context"
	"net/http"
	"testing"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewGRPCGatewayFailsClosedWhenClientMissing(t *testing.T) {
	t.Parallel()

	gateway := NewGRPCGateway(module.Dependencies{})
	_, err := gateway.ListNotifications(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestGRPCGatewayListNotificationsMapsFieldsAndUserMetadata(t *testing.T) {
	t.Parallel()

	createdAt := timestamppb.New(time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC))
	readAt := timestamppb.New(time.Date(2026, 2, 25, 11, 0, 0, 0, time.UTC))
	client := &notificationClientStub{
		listResponses: []*notificationsv1.ListNotificationsResponse{{
			Notifications: []*notificationsv1.Notification{{
				Id:          " note-1 ",
				MessageType: "auth.onboarding.welcome",
				PayloadJson: `{"signup_method":"passkey"}`,
				Source:      notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
				CreatedAt:   createdAt,
				UpdatedAt:   readAt,
				ReadAt:      readAt,
			}},
		}},
	}
	gateway := grpcGateway{client: client}

	items, err := gateway.ListNotifications(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListNotifications() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].ID != "note-1" {
		t.Fatalf("ID = %q, want %q", items[0].ID, "note-1")
	}
	if items[0].MessageType != "auth.onboarding.welcome" {
		t.Fatalf("MessageType = %q, want %q", items[0].MessageType, "auth.onboarding.welcome")
	}
	if !items[0].Read {
		t.Fatalf("expected read=true when read_at exists")
	}
	if items[0].Source != notificationSourceSystem {
		t.Fatalf("Source = %q, want %q", items[0].Source, notificationSourceSystem)
	}
	if client.lastListUserID != "user-1" {
		t.Fatalf("list metadata user id = %q, want %q", client.lastListUserID, "user-1")
	}
	if client.lastListReq.GetPageSize() != notificationPageSize {
		t.Fatalf("PageSize = %d, want %d", client.lastListReq.GetPageSize(), notificationPageSize)
	}
}

func TestGRPCGatewayGetNotificationScansPages(t *testing.T) {
	t.Parallel()

	client := &notificationClientStub{
		listResponses: []*notificationsv1.ListNotificationsResponse{
			{Notifications: []*notificationsv1.Notification{{Id: "note-1"}}, NextPageToken: "token-2"},
			{Notifications: []*notificationsv1.Notification{{Id: "note-2"}}},
		},
	}
	gateway := grpcGateway{client: client}

	item, err := gateway.GetNotification(context.Background(), "user-1", "note-2")
	if err != nil {
		t.Fatalf("GetNotification() error = %v", err)
	}
	if item.ID != "note-2" {
		t.Fatalf("ID = %q, want %q", item.ID, "note-2")
	}
}

func TestGRPCGatewayOpenNotificationMapsNotFound(t *testing.T) {
	t.Parallel()

	client := &notificationClientStub{markErr: status.Error(codes.NotFound, "notification not found")}
	gateway := grpcGateway{client: client}

	_, err := gateway.OpenNotification(context.Background(), "user-1", "missing")
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusNotFound {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusNotFound)
	}
}

func TestGRPCGatewayRequiresExplicitUserID(t *testing.T) {
	t.Parallel()

	client := &notificationClientStub{}
	gateway := grpcGateway{client: client}

	_, err := gateway.ListNotifications(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected user-id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusUnauthorized {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusUnauthorized)
	}
}

type notificationClientStub struct {
	listResponses  []*notificationsv1.ListNotificationsResponse
	listErr        error
	markResp       *notificationsv1.MarkNotificationReadResponse
	markErr        error
	lastListReq    *notificationsv1.ListNotificationsRequest
	lastMarkReq    *notificationsv1.MarkNotificationReadRequest
	lastListUserID string
	lastMarkUserID string
}

func (f *notificationClientStub) ListNotifications(ctx context.Context, req *notificationsv1.ListNotificationsRequest, _ ...grpc.CallOption) (*notificationsv1.ListNotificationsResponse, error) {
	f.lastListReq = req
	f.lastListUserID = outgoingUserID(ctx)
	if f.listErr != nil {
		return nil, f.listErr
	}
	if len(f.listResponses) == 0 {
		return &notificationsv1.ListNotificationsResponse{}, nil
	}
	resp := f.listResponses[0]
	f.listResponses = f.listResponses[1:]
	return resp, nil
}

func (f *notificationClientStub) GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
	return &notificationsv1.GetUnreadNotificationStatusResponse{}, nil
}

func (f *notificationClientStub) MarkNotificationRead(ctx context.Context, req *notificationsv1.MarkNotificationReadRequest, _ ...grpc.CallOption) (*notificationsv1.MarkNotificationReadResponse, error) {
	f.lastMarkReq = req
	f.lastMarkUserID = outgoingUserID(ctx)
	if f.markErr != nil {
		return nil, f.markErr
	}
	if f.markResp != nil {
		return f.markResp, nil
	}
	return &notificationsv1.MarkNotificationReadResponse{Notification: &notificationsv1.Notification{Id: req.GetNotificationId()}}, nil
}

func outgoingUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(grpcmeta.UserIDHeader)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
