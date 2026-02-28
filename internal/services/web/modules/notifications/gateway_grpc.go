package notifications

import (
	"context"
	"strings"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/grpcpaging"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NotificationClient exposes notification inbox listing and acknowledgement operations.
type NotificationClient interface {
	ListNotifications(context.Context, *notificationsv1.ListNotificationsRequest, ...grpc.CallOption) (*notificationsv1.ListNotificationsResponse, error)
	GetUnreadNotificationStatus(context.Context, *notificationsv1.GetUnreadNotificationStatusRequest, ...grpc.CallOption) (*notificationsv1.GetUnreadNotificationStatusResponse, error)
	GetNotification(context.Context, *notificationsv1.GetNotificationRequest, ...grpc.CallOption) (*notificationsv1.GetNotificationResponse, error)
	MarkNotificationRead(context.Context, *notificationsv1.MarkNotificationReadRequest, ...grpc.CallOption) (*notificationsv1.MarkNotificationReadResponse, error)
}

const (
	notificationSourceSystem  = "system"
	notificationSourceUnknown = "unknown"
	notificationPageSize      = int32(200)
	notificationMaxPages      = 20
)

// NewGRPCGateway builds the production notifications gateway from the notification client.
func NewGRPCGateway(client NotificationClient) NotificationGateway {
	if client == nil {
		return unavailableGateway{}
	}
	return grpcGateway{client: client}
}

type grpcGateway struct {
	client NotificationClient
}

func (g grpcGateway) ListNotifications(ctx context.Context, userID string) ([]NotificationSummary, error) {
	return g.listAllNotifications(ctx, userID)
}

func (g grpcGateway) GetNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	resp, err := g.client.GetNotification(
		grpcauthctx.WithUserID(ctx, userID),
		&notificationsv1.GetNotificationRequest{NotificationId: notificationID},
	)
	if err != nil {
		return NotificationSummary{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackKey:     "error.web.message.failed_to_get_notification",
			FallbackMessage: "failed to get notification",
		})
	}
	if resp == nil {
		return NotificationSummary{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.failed_to_get_notification", "failed to get notification")
	}
	return mapNotification(resp.GetNotification()), nil
}

func (g grpcGateway) OpenNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	resp, err := g.client.MarkNotificationRead(
		grpcauthctx.WithUserID(ctx, userID),
		&notificationsv1.MarkNotificationReadRequest{NotificationId: notificationID},
	)
	if err != nil {
		return NotificationSummary{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackKey:     "error.web.message.failed_to_mark_notification_read",
			FallbackMessage: "failed to mark notification read",
		})
	}
	return mapNotification(resp.GetNotification()), nil
}

func (g grpcGateway) listAllNotifications(ctx context.Context, userID string) ([]NotificationSummary, error) {
	authCtx := grpcauthctx.WithUserID(ctx, userID)
	return grpcpaging.CollectPagesMax[NotificationSummary, *notificationsv1.Notification](
		authCtx, notificationPageSize, notificationMaxPages,
		func(ctx context.Context, pageToken string) ([]*notificationsv1.Notification, string, error) {
			resp, err := g.client.ListNotifications(
				ctx,
				&notificationsv1.ListNotificationsRequest{PageSize: notificationPageSize, PageToken: pageToken},
			)
			if err != nil {
				return nil, "", apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
					FallbackKind:    apperrors.KindUnavailable,
					FallbackKey:     "error.web.message.failed_to_list_notifications",
					FallbackMessage: "failed to list notifications",
				})
			}
			if resp == nil {
				return nil, "", nil
			}
			return resp.GetNotifications(), resp.GetNextPageToken(), nil
		},
		func(notification *notificationsv1.Notification) (NotificationSummary, bool) {
			if notification == nil {
				return NotificationSummary{}, false
			}
			return mapNotification(notification), true
		},
	)
}

func mapNotification(notification *notificationsv1.Notification) NotificationSummary {
	if notification == nil {
		return NotificationSummary{}
	}
	readAt := protoTimestamp(notification.GetReadAt())
	return NotificationSummary{
		ID:          strings.TrimSpace(notification.GetId()),
		MessageType: strings.TrimSpace(notification.GetMessageType()),
		PayloadJSON: strings.TrimSpace(notification.GetPayloadJson()),
		Source:      sourceFromProto(notification.GetSource()),
		Read:        readAt != nil,
		CreatedAt:   protoTimestampValue(notification.GetCreatedAt()),
		UpdatedAt:   protoTimestampValue(notification.GetUpdatedAt()),
		ReadAt:      readAt,
	}
}

func sourceFromProto(source notificationsv1.NotificationSource) string {
	switch source {
	case notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM:
		return notificationSourceSystem
	default:
		return notificationSourceUnknown
	}
}

func protoTimestampValue(value *timestamppb.Timestamp) time.Time {
	timestamp := protoTimestamp(value)
	if timestamp == nil {
		return time.Time{}
	}
	return *timestamp
}

func protoTimestamp(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}
	if err := value.CheckValid(); err != nil {
		return nil
	}
	timestamp := value.AsTime().UTC()
	return &timestamp
}
