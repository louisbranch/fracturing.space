package domain

import "strings"

const (
	// MessageTypeOnboardingWelcome is the canonical onboarding welcome notification type.
	MessageTypeOnboardingWelcome = "auth.onboarding.welcome"
	// MessageTypeOnboardingWelcomeV1 is the versioned onboarding welcome notification type.
	MessageTypeOnboardingWelcomeV1 = "auth.onboarding.welcome.v1"
)

// DeliveryPolicy defines the service-owned effective channels for one message type.
type DeliveryPolicy struct {
	InApp bool
	Email bool
}

// NormalizeMessageType normalizes a producer-provided message type token.
func NormalizeMessageType(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// ResolveDeliveryPolicy returns the effective channel policy for one message type.
//
// TODO(notifications-preferences): add user-specific message-type overrides for
// configurable message types after preferences storage/UI are implemented.
func ResolveDeliveryPolicy(messageType string) DeliveryPolicy {
	switch NormalizeMessageType(messageType) {
	case MessageTypeOnboardingWelcome, MessageTypeOnboardingWelcomeV1:
		// Onboarding welcome stays email-only and non-configurable.
		return DeliveryPolicy{InApp: false, Email: true}
	default:
		// General notifications are in-app by default until preferences ship.
		return DeliveryPolicy{InApp: true, Email: false}
	}
}
