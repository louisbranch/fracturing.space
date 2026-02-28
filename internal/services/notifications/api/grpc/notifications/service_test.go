package notifications

import (
	"context"
	"errors"
	"testing"
	"time"

	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/notifications/domain"
	"google.golang.org/grpc/codes"
	grpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestCreateNotificationIntent_Success(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 21, 35, 0, 0, time.UTC)
	fake := &fakeDomainService{
		createResult: domain.Notification{
			ID:              "notif-1",
			RecipientUserID: "user-1",
			MessageType:     "campaign.invite",
			PayloadJSON:     `{"invite_id":"inv-1"}`,
			DedupeKey:       "invite:inv-1",
			Source:          "system",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}
	svc := NewService(fake)

	resp, err := svc.CreateNotificationIntent(context.Background(), &notificationsv1.CreateNotificationIntentRequest{
		RecipientUserId: "user-1",
		MessageType:     "campaign.invite",
		PayloadJson:     `{"invite_id":"inv-1"}`,
		DedupeKey:       "invite:inv-1",
		Source:          notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM,
	})
	if err != nil {
		t.Fatalf("create notification intent: %v", err)
	}
	if resp.GetNotification().GetId() != "notif-1" {
		t.Fatalf("notification.id = %q, want %q", resp.GetNotification().GetId(), "notif-1")
	}
	if fake.lastCreate.Source != "system" {
		t.Fatalf("domain source = %q, want %q", fake.lastCreate.Source, "system")
	}
}

func TestListNotifications_RequiresUserIdentity(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeDomainService{})
	_, err := svc.ListNotifications(context.Background(), &notificationsv1.ListNotificationsRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
}

func TestListNotifications_UsesCallerIdentity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 21, 21, 40, 0, 0, time.UTC)
	fake := &fakeDomainService{
		listResult: domain.NotificationPage{
			Notifications: []domain.Notification{
				{
					ID:              "notif-2",
					RecipientUserID: "user-1",
					MessageType:     "session.update",
					PayloadJSON:     `{"session_id":"sess-1"}`,
					DedupeKey:       "session:sess-1",
					Source:          "system",
					CreatedAt:       now,
					UpdatedAt:       now,
				},
			},
		},
	}
	svc := NewService(fake)

	ctx := grpcmetadata.NewIncomingContext(context.Background(), grpcmetadata.Pairs(metadata.UserIDHeader, "user-1"))
	resp, err := svc.ListNotifications(ctx, &notificationsv1.ListNotificationsRequest{
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if fake.lastListInput.RecipientUserID != "user-1" {
		t.Fatalf("recipient_user_id = %q, want %q", fake.lastListInput.RecipientUserID, "user-1")
	}
	if len(resp.GetNotifications()) != 1 {
		t.Fatalf("notifications len = %d, want 1", len(resp.GetNotifications()))
	}
}

func TestMarkNotificationRead_NotFound(t *testing.T) {
	t.Parallel()

	fake := &fakeDomainService{
		markReadErr: domain.ErrNotFound,
	}
	svc := NewService(fake)
	ctx := grpcmetadata.NewIncomingContext(context.Background(), grpcmetadata.Pairs(metadata.UserIDHeader, "user-1"))

	_, err := svc.MarkNotificationRead(ctx, &notificationsv1.MarkNotificationReadRequest{NotificationId: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestGetUnreadNotificationStatus_RequiresUserIdentity(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeDomainService{})
	_, err := svc.GetUnreadNotificationStatus(context.Background(), &notificationsv1.GetUnreadNotificationStatusRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
}

func TestGetUnreadNotificationStatus_UsesCallerIdentity(t *testing.T) {
	t.Parallel()

	fake := &fakeDomainService{
		unreadStatusResult: domain.UnreadStatus{
			HasUnread:   true,
			UnreadCount: 2,
		},
	}
	svc := NewService(fake)

	ctx := grpcmetadata.NewIncomingContext(context.Background(), grpcmetadata.Pairs(metadata.UserIDHeader, "user-1"))
	resp, err := svc.GetUnreadNotificationStatus(ctx, &notificationsv1.GetUnreadNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("get unread notification status: %v", err)
	}
	if fake.lastUnreadStatusInput.RecipientUserID != "user-1" {
		t.Fatalf("recipient_user_id = %q, want %q", fake.lastUnreadStatusInput.RecipientUserID, "user-1")
	}
	if !resp.GetHasUnread() {
		t.Fatalf("has_unread = false, want true")
	}
	if resp.GetUnreadCount() != 2 {
		t.Fatalf("unread_count = %d, want 2", resp.GetUnreadCount())
	}
}

type fakeDomainService struct {
	createResult domain.Notification
	createErr    error
	lastCreate   domain.CreateIntentInput

	listResult    domain.NotificationPage
	listErr       error
	lastListInput domain.ListInboxInput

	markReadResult domain.Notification
	markReadErr    error
	lastMarkRead   domain.MarkReadInput

	unreadStatusResult    domain.UnreadStatus
	unreadStatusErr       error
	lastUnreadStatusInput domain.GetUnreadStatusInput
}

func (f *fakeDomainService) CreateIntent(_ context.Context, input domain.CreateIntentInput) (domain.Notification, error) {
	f.lastCreate = input
	if f.createErr != nil {
		return domain.Notification{}, f.createErr
	}
	return f.createResult, nil
}

func (f *fakeDomainService) ListInbox(_ context.Context, input domain.ListInboxInput) (domain.NotificationPage, error) {
	f.lastListInput = input
	if f.listErr != nil {
		return domain.NotificationPage{}, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeDomainService) MarkRead(_ context.Context, input domain.MarkReadInput) (domain.Notification, error) {
	f.lastMarkRead = input
	if f.markReadErr != nil {
		return domain.Notification{}, f.markReadErr
	}
	return f.markReadResult, nil
}

func (f *fakeDomainService) GetUnreadStatus(_ context.Context, input domain.GetUnreadStatusInput) (domain.UnreadStatus, error) {
	f.lastUnreadStatusInput = input
	if f.unreadStatusErr != nil {
		return domain.UnreadStatus{}, f.unreadStatusErr
	}
	return f.unreadStatusResult, nil
}

var _ domainService = (*fakeDomainService)(nil)

func TestMapDomainError(t *testing.T) {
	t.Parallel()

	if got := status.Code(mapDomainError(domain.ErrNotFound)); got != codes.NotFound {
		t.Fatalf("map ErrNotFound = %v, want %v", got, codes.NotFound)
	}
	if got := status.Code(mapDomainError(domain.ErrRecipientUserIDRequired)); got != codes.InvalidArgument {
		t.Fatalf("map ErrRecipientUserIDRequired = %v, want %v", got, codes.InvalidArgument)
	}
	if got := status.Code(mapDomainError(domain.ErrConflict)); got != codes.AlreadyExists {
		t.Fatalf("map ErrConflict = %v, want %v", got, codes.AlreadyExists)
	}
	internal := errors.New("boom")
	if got := status.Code(mapDomainError(internal)); got != codes.Internal {
		t.Fatalf("map internal = %v, want %v", got, codes.Internal)
	}
}
