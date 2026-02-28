package domain

import (
	"context"
	"testing"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestOnboardingWelcomeHandler_HandleSuccess(t *testing.T) {
	now := time.Date(2026, 2, 21, 23, 20, 0, 0, time.UTC)
	notif := &fakeNotificationsClient{
		createResp: &notificationsv1.CreateNotificationIntentResponse{},
	}
	handler := NewOnboardingWelcomeHandler(notif, func() time.Time { return now })

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1","email":"user-1@example.com","signup_method":"passkey"}`,
	})
	if err != nil {
		t.Fatalf("handle onboarding welcome: %v", err)
	}
	if notif.lastCreateReq == nil {
		t.Fatal("expected create notification intent request")
	}
	if notif.lastCreateReq.GetRecipientUserId() != "user-1" {
		t.Fatalf("recipient user id = %q, want %q", notif.lastCreateReq.GetRecipientUserId(), "user-1")
	}
	if notif.lastCreateReq.GetMessageType() != "auth.onboarding.welcome" {
		t.Fatalf("message_type = %q, want %q", notif.lastCreateReq.GetMessageType(), "auth.onboarding.welcome")
	}
	if got := notif.lastCreateReq.GetSource(); got != notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM {
		t.Fatalf("source = %v, want %v", got, notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM)
	}
	if notif.lastCreateReq.GetDedupeKey() != "welcome:user:user-1:v1" {
		t.Fatalf("dedupe key = %q, want %q", notif.lastCreateReq.GetDedupeKey(), "welcome:user:user-1:v1")
	}
}

func TestOnboardingWelcomeHandler_MissingUserIDPermanent(t *testing.T) {
	handler := NewOnboardingWelcomeHandler(&fakeNotificationsClient{}, nil)

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{}`,
	})
	if err == nil {
		t.Fatal("expected error for missing user id")
	}
	if !IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestOnboardingWelcomeHandler_MissingEmailSkipsWelcomeIntent(t *testing.T) {
	notif := &fakeNotificationsClient{}
	handler := NewOnboardingWelcomeHandler(notif, nil)

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1","signup_method":"passkey"}`,
	})
	if err != nil {
		t.Fatalf("handle onboarding welcome: %v", err)
	}
	if notif.lastCreateReq != nil {
		t.Fatal("expected no notification intent when signup payload has no email")
	}
}

func TestOnboardingWelcomeHandler_InvalidArgumentFromNotificationsPermanent(t *testing.T) {
	notif := &fakeNotificationsClient{
		createErr: status.Error(codes.InvalidArgument, "bad request"),
	}
	handler := NewOnboardingWelcomeHandler(notif, nil)

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1","email":"user-1@example.com"}`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestOnboardingWelcomeHandler_UnavailableRetryable(t *testing.T) {
	notif := &fakeNotificationsClient{
		createErr: status.Error(codes.Unavailable, "downstream unavailable"),
	}
	handler := NewOnboardingWelcomeHandler(notif, nil)

	err := handler.Handle(context.Background(), &authv1.IntegrationOutboxEvent{
		Id:          "evt-1",
		EventType:   "auth.signup_completed",
		PayloadJson: `{"user_id":"user-1","email":"user-1@example.com"}`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if IsPermanent(err) {
		t.Fatalf("expected retryable error, got permanent: %v", err)
	}
}

type fakeNotificationsClient struct {
	createResp    *notificationsv1.CreateNotificationIntentResponse
	createErr     error
	lastCreateReq *notificationsv1.CreateNotificationIntentRequest
}

func (f *fakeNotificationsClient) CreateNotificationIntent(_ context.Context, req *notificationsv1.CreateNotificationIntentRequest, _ ...grpc.CallOption) (*notificationsv1.CreateNotificationIntentResponse, error) {
	f.lastCreateReq = req
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp == nil {
		return &notificationsv1.CreateNotificationIntentResponse{}, nil
	}
	return f.createResp, nil
}
