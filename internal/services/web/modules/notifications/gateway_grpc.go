package notifications

import (
	"context"
	"strings"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	notificationSourceSystem  = "system"
	notificationSourceUnknown = "unknown"
	notificationPageSize      = int32(200)
	notificationMaxPages      = 20
)

// NewGRPCGateway builds the production notifications gateway from shared dependencies.
func NewGRPCGateway(deps module.Dependencies) NotificationGateway {
	if deps.NotificationClient == nil {
		return unavailableGateway{}
	}
	return grpcGateway{client: deps.NotificationClient}
}

type grpcGateway struct {
	client module.NotificationClient
}

func (g grpcGateway) ListNotifications(ctx context.Context, userID string) ([]NotificationSummary, error) {
	if g.client == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.notification_service_client_is_not_configured", "notification service client is not configured")
	}
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return nil, err
	}
	items, err := g.listAllNotifications(ctx, resolvedUserID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (g grpcGateway) GetNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return NotificationSummary{}, err
	}
	resolvedNotificationID := strings.TrimSpace(notificationID)
	if resolvedNotificationID == "" {
		return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	items, err := g.listAllNotifications(ctx, resolvedUserID)
	if err != nil {
		return NotificationSummary{}, err
	}
	for _, item := range items {
		if strings.TrimSpace(item.ID) == resolvedNotificationID {
			return item, nil
		}
	}
	return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
}

func (g grpcGateway) OpenNotification(ctx context.Context, userID string, notificationID string) (NotificationSummary, error) {
	if g.client == nil {
		return NotificationSummary{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.notification_service_client_is_not_configured", "notification service client is not configured")
	}
	resolvedUserID, err := requireUserID(userID)
	if err != nil {
		return NotificationSummary{}, err
	}
	resolvedNotificationID := strings.TrimSpace(notificationID)
	if resolvedNotificationID == "" {
		return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	resp, err := g.client.MarkNotificationRead(
		grpcauthctx.WithUserID(ctx, resolvedUserID),
		&notificationsv1.MarkNotificationReadRequest{NotificationId: resolvedNotificationID},
	)
	if err != nil {
		return NotificationSummary{}, mapOpenNotificationError(err)
	}
	item := mapNotification(resp.GetNotification())
	if strings.TrimSpace(item.ID) == "" {
		return NotificationSummary{}, apperrors.E(apperrors.KindNotFound, "notification not found")
	}
	return item, nil
}

func (g grpcGateway) listAllNotifications(ctx context.Context, userID string) ([]NotificationSummary, error) {
	if g.client == nil {
		return nil, apperrors.EK(apperrors.KindUnavailable, "error.web.message.notification_service_client_is_not_configured", "notification service client is not configured")
	}
	pageToken := ""
	items := make([]NotificationSummary, 0, notificationPageSize)
	for page := 0; page < notificationMaxPages; page++ {
		resp, err := g.client.ListNotifications(
			grpcauthctx.WithUserID(ctx, userID),
			&notificationsv1.ListNotificationsRequest{PageSize: notificationPageSize, PageToken: pageToken},
		)
		if err != nil {
			return nil, mapListNotificationsError(err)
		}
		if resp == nil {
			break
		}
		for _, notification := range resp.GetNotifications() {
			if notification == nil {
				continue
			}
			items = append(items, mapNotification(notification))
		}
		nextPageToken := strings.TrimSpace(resp.GetNextPageToken())
		if nextPageToken == "" || nextPageToken == pageToken {
			break
		}
		pageToken = nextPageToken
	}
	return items, nil
}

func mapNotification(notification *notificationsv1.Notification) NotificationSummary {
	if notification == nil {
		return NotificationSummary{}
	}
	readAt := protoTimestamp(notification.GetReadAt())
	return NotificationSummary{
		ID:          strings.TrimSpace(notification.GetId()),
		Topic:       strings.TrimSpace(notification.GetTopic()),
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

func mapListNotificationsError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.Unauthenticated:
		return apperrors.E(apperrors.KindUnauthorized, "authentication required")
	case codes.PermissionDenied:
		return apperrors.E(apperrors.KindForbidden, "access denied")
	case codes.InvalidArgument:
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_list_notifications", "failed to list notifications")
	default:
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.failed_to_list_notifications", "failed to list notifications")
	}
}

func mapOpenNotificationError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.NotFound:
		return apperrors.E(apperrors.KindNotFound, "notification not found")
	case codes.Unauthenticated:
		return apperrors.E(apperrors.KindUnauthorized, "authentication required")
	case codes.PermissionDenied:
		return apperrors.E(apperrors.KindForbidden, "access denied")
	case codes.InvalidArgument:
		return apperrors.EK(apperrors.KindInvalidInput, "error.web.message.failed_to_mark_notification_read", "failed to mark notification read")
	default:
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.failed_to_mark_notification_read", "failed to mark notification read")
	}
}
