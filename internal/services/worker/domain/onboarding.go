package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// OnboardingWelcomeTopic is the canonical topic for signup welcome notifications.
	OnboardingWelcomeTopic = "auth.onboarding.welcome"
	// OnboardingWelcomeSource is the internal catch-all notification source.
	OnboardingWelcomeSource = notificationsv1.NotificationSource_NOTIFICATION_SOURCE_SYSTEM
)

type notificationIntentClient interface {
	CreateNotificationIntent(ctx context.Context, in *notificationsv1.CreateNotificationIntentRequest, opts ...grpc.CallOption) (*notificationsv1.CreateNotificationIntentResponse, error)
}

// OnboardingWelcomeHandler handles auth signup completion events.
type OnboardingWelcomeHandler struct {
	notifications notificationIntentClient
	clock         func() time.Time
}

// NewOnboardingWelcomeHandler creates an onboarding welcome event handler.
func NewOnboardingWelcomeHandler(notifications notificationIntentClient, clock func() time.Time) *OnboardingWelcomeHandler {
	if clock == nil {
		clock = time.Now
	}
	return &OnboardingWelcomeHandler{
		notifications: notifications,
		clock:         clock,
	}
}

// Handle converts signup completed events into notification intents.
func (h *OnboardingWelcomeHandler) Handle(ctx context.Context, event *authv1.IntegrationOutboxEvent) error {
	if h == nil || h.notifications == nil {
		return Permanent(fmt.Errorf("notifications client is not configured"))
	}
	payload, err := decodeSignupCompletedPayload(event)
	if err != nil {
		return Permanent(err)
	}
	userID := payload.UserID

	now := h.clock().UTC()
	notificationPayload, err := json.Marshal(map[string]string{
		"user_id":        userID,
		"event_id":       strings.TrimSpace(event.GetId()),
		"event_type":     strings.TrimSpace(event.GetEventType()),
		"signup_method":  strings.TrimSpace(payload.SignupMethod),
		"notified_at":    now.Format(time.RFC3339Nano),
		"notification_v": "v1",
	})
	if err != nil {
		return Permanent(fmt.Errorf("encode notification payload: %w", err))
	}

	_, err = h.notifications.CreateNotificationIntent(ctx, &notificationsv1.CreateNotificationIntentRequest{
		RecipientUserId: userID,
		Topic:           OnboardingWelcomeTopic,
		PayloadJson:     string(notificationPayload),
		DedupeKey:       onboardingWelcomeDedupeKey(userID),
		Source:          OnboardingWelcomeSource,
	})
	if err == nil {
		return nil
	}
	if isPermanentNotificationError(err) {
		return Permanent(err)
	}
	return err
}

func onboardingWelcomeDedupeKey(userID string) string {
	return "welcome:user:" + userID + ":v1"
}

func isPermanentNotificationError(err error) bool {
	code := status.Code(err)
	switch code {
	case codes.InvalidArgument, codes.PermissionDenied, codes.NotFound, codes.FailedPrecondition:
		return true
	default:
		return false
	}
}
