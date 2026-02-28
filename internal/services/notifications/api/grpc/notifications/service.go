package notifications

import (
	"context"
	"errors"
	"strings"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type domainService interface {
	CreateIntent(ctx context.Context, input domain.CreateIntentInput) (domain.Notification, error)
	ListInbox(ctx context.Context, input domain.ListInboxInput) (domain.NotificationPage, error)
	GetNotification(ctx context.Context, input domain.GetNotificationInput) (domain.Notification, error)
	GetUnreadStatus(ctx context.Context, input domain.GetUnreadStatusInput) (domain.UnreadStatus, error)
	MarkRead(ctx context.Context, input domain.MarkReadInput) (domain.Notification, error)
}

const sourceSystem = "system"

// Service exposes notifications.v1 gRPC operations.
type Service struct {
	notificationsv1.UnimplementedNotificationServiceServer
	domain domainService
}

// NewService creates a notifications gRPC service.
func NewService(domainSvc domainService) *Service {
	return &Service{domain: domainSvc}
}

// CreateNotificationIntent appends one user-targeted notification.
func (s *Service) CreateNotificationIntent(ctx context.Context, in *notificationsv1.CreateNotificationIntentRequest) (*notificationsv1.CreateNotificationIntentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create notification intent request is required")
	}
	if s == nil || s.domain == nil {
		return nil, status.Error(codes.Internal, "notifications domain service is not configured")
	}

	created, err := s.domain.CreateIntent(ctx, domain.CreateIntentInput{
		RecipientUserID: in.GetRecipientUserId(),
		MessageType:     in.GetMessageType(),
		PayloadJSON:     in.GetPayloadJson(),
		DedupeKey:       in.GetDedupeKey(),
		Source:          sourceFromProto(in.GetSource()),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &notificationsv1.CreateNotificationIntentResponse{
		Notification: notificationToProto(created),
	}, nil
}

// ListNotifications lists caller-visible inbox notifications.
func (s *Service) ListNotifications(ctx context.Context, in *notificationsv1.ListNotificationsRequest) (*notificationsv1.ListNotificationsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list notifications request is required")
	}
	if s == nil || s.domain == nil {
		return nil, status.Error(codes.Internal, "notifications domain service is not configured")
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := s.domain.ListInbox(ctx, domain.ListInboxInput{
		RecipientUserID: userID,
		PageSize:        int(in.GetPageSize()),
		PageToken:       in.GetPageToken(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	resp := &notificationsv1.ListNotificationsResponse{
		Notifications: make([]*notificationsv1.Notification, 0, len(page.Notifications)),
		NextPageToken: page.NextPageToken,
	}
	for _, notification := range page.Notifications {
		resp.Notifications = append(resp.Notifications, notificationToProto(notification))
	}
	return resp, nil
}

// GetNotification fetches one notification visible to the caller.
func (s *Service) GetNotification(ctx context.Context, in *notificationsv1.GetNotificationRequest) (*notificationsv1.GetNotificationResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get notification request is required")
	}
	if s == nil || s.domain == nil {
		return nil, status.Error(codes.Internal, "notifications domain service is not configured")
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	notification, err := s.domain.GetNotification(ctx, domain.GetNotificationInput{
		RecipientUserID: userID,
		NotificationID:  in.GetNotificationId(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	if strings.TrimSpace(notification.ID) == "" {
		return nil, status.Error(codes.NotFound, "notification not found")
	}
	return &notificationsv1.GetNotificationResponse{
		Notification: notificationToProto(notification),
	}, nil
}

// GetUnreadNotificationStatus returns unread-inbox status for the caller.
func (s *Service) GetUnreadNotificationStatus(ctx context.Context, in *notificationsv1.GetUnreadNotificationStatusRequest) (*notificationsv1.GetUnreadNotificationStatusResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get unread notification status request is required")
	}
	if s == nil || s.domain == nil {
		return nil, status.Error(codes.Internal, "notifications domain service is not configured")
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	unreadStatus, err := s.domain.GetUnreadStatus(ctx, domain.GetUnreadStatusInput{
		RecipientUserID: userID,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &notificationsv1.GetUnreadNotificationStatusResponse{
		HasUnread:   unreadStatus.HasUnread,
		UnreadCount: int32(unreadStatus.UnreadCount),
	}, nil
}

// MarkNotificationRead marks one caller-owned notification as read.
func (s *Service) MarkNotificationRead(ctx context.Context, in *notificationsv1.MarkNotificationReadRequest) (*notificationsv1.MarkNotificationReadResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "mark notification read request is required")
	}
	if s == nil || s.domain == nil {
		return nil, status.Error(codes.Internal, "notifications domain service is not configured")
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	read, err := s.domain.MarkRead(ctx, domain.MarkReadInput{
		RecipientUserID: userID,
		NotificationID:  in.GetNotificationId(),
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &notificationsv1.MarkNotificationReadResponse{
		Notification: notificationToProto(read),
	}, nil
}

func notificationToProto(notification domain.Notification) *notificationsv1.Notification {
	result := &notificationsv1.Notification{
		Id:              notification.ID,
		RecipientUserId: notification.RecipientUserID,
		MessageType:     notification.MessageType,
		PayloadJson:     notification.PayloadJSON,
		DedupeKey:       notification.DedupeKey,
		Source:          sourceToProto(notification.Source),
		CreatedAt:       timestamppb.New(notification.CreatedAt),
		UpdatedAt:       timestamppb.New(notification.UpdatedAt),
	}
	if notification.ReadAt != nil {
		result.ReadAt = timestamppb.New(*notification.ReadAt)
	}
	return result
}

func sourceFromProto(source notificationsv1.NotificationSource) string {
	switch source {
	case notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM:
		return sourceSystem
	default:
		return sourceSystem
	}
}

func sourceToProto(source string) notificationsv1.NotificationSource {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "", sourceSystem:
		return notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM
	default:
		return notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM
	}
}

func mapDomainError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "notification not found")
	case errors.Is(err, domain.ErrRecipientUserIDRequired):
		return status.Error(codes.InvalidArgument, domain.ErrRecipientUserIDRequired.Error())
	case errors.Is(err, domain.ErrMessageTypeRequired):
		return status.Error(codes.InvalidArgument, domain.ErrMessageTypeRequired.Error())
	case errors.Is(err, domain.ErrNotificationIDRequired):
		return status.Error(codes.InvalidArgument, domain.ErrNotificationIDRequired.Error())
	case errors.Is(err, domain.ErrConflict):
		return status.Error(codes.AlreadyExists, domain.ErrConflict.Error())
	case errors.Is(err, domain.ErrStoreNotConfigured):
		return status.Error(codes.Internal, domain.ErrStoreNotConfigured.Error())
	case errors.Is(err, domain.ErrIDGeneratorNotConfigured):
		return status.Error(codes.Internal, domain.ErrIDGeneratorNotConfigured.Error())
	default:
		return status.Errorf(codes.Internal, "notifications domain: %v", err)
	}
}
