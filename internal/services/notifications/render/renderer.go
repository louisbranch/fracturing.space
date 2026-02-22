package render

import (
	"encoding/json"
	"strings"

	"golang.org/x/text/message"
)

const (
	// TopicOnboardingWelcome aliases the canonical onboarding welcome template id.
	TopicOnboardingWelcome = "auth.onboarding.welcome"
	// TopicOnboardingWelcomeV1 is the versioned onboarding welcome template id.
	TopicOnboardingWelcomeV1 = "auth.onboarding.welcome.v1"

	defaultGenericTitle        = "Notification"
	defaultGenericBody         = "You have a new notification."
	defaultGenericEmailSubject = "Fracturing Space notification"
	defaultUnknownSignupMethod = "another method"
)

// Channel identifies where one notification artifact is rendered.
type Channel string

const (
	// ChannelInApp renders copy for the web inbox/detail view.
	ChannelInApp Channel = "in_app"
	// ChannelEmail renders copy for email delivery.
	ChannelEmail Channel = "email"
)

// Input is one channel render request for a stored notification artifact.
type Input struct {
	Topic       string
	PayloadJSON string
	Channel     Channel
}

// Output is localized, channel-aware copy derived from one notification artifact.
type Output struct {
	Title        string
	BodyText     string
	EmailSubject string
}

// Localizer is the minimal message-printer contract required by the renderer.
type Localizer interface {
	Sprintf(key message.Reference, args ...any) string
}

type onboardingPayload struct {
	SignupMethod string `json:"signup_method"`
}

// Render returns localized copy for one notification artifact.
func Render(loc Localizer, input Input) Output {
	switch normalizeToken(input.Topic) {
	case TopicOnboardingWelcome, TopicOnboardingWelcomeV1:
		return renderOnboardingWelcome(loc, input)
	default:
		return genericOutput(loc)
	}
}

func renderOnboardingWelcome(loc Localizer, input Input) Output {
	payload := onboardingPayload{}
	if raw := strings.TrimSpace(input.PayloadJSON); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return genericOutput(loc)
		}
	}

	signupMethod := localizedSignupMethod(loc, payload.SignupMethod)
	title := localize(loc, "notification.onboarding_welcome.title")
	subject := localize(loc, "notification.onboarding_welcome.email_subject")
	if subject == "notification.onboarding_welcome.email_subject" {
		subject = title
	}

	bodyKey := "notification.onboarding_welcome.body"
	if input.Channel == ChannelEmail {
		bodyKey = "notification.onboarding_welcome.email_body"
	}
	body := localize(loc, bodyKey, signupMethod)

	if title == "notification.onboarding_welcome.title" || body == bodyKey {
		return genericOutput(loc)
	}

	return Output{
		Title:        title,
		BodyText:     body,
		EmailSubject: subject,
	}
}

func genericOutput(loc Localizer) Output {
	title := localizeWithFallback(loc, "notification.generic.title", defaultGenericTitle)
	body := localizeWithFallback(loc, "notification.generic.body", defaultGenericBody)
	subject := localizeWithFallback(loc, "notification.generic.email_subject", defaultGenericEmailSubject)
	if subject == "notification.generic.email_subject" {
		subject = title
	}

	return Output{
		Title:        title,
		BodyText:     body,
		EmailSubject: subject,
	}
}

func localizedSignupMethod(loc Localizer, raw string) string {
	key := "notification.signup_method.unknown"
	fallback := defaultUnknownSignupMethod
	switch normalizeToken(raw) {
	case "passkey":
		key = "notification.signup_method.passkey"
		fallback = "passkey"
	case "magic-link", "magic_link", "magiclink":
		key = "notification.signup_method.magic_link"
		fallback = "magic link"
	}

	return localizeWithFallback(loc, key, fallback)
}

func localize(loc Localizer, key message.Reference, args ...any) string {
	if loc == nil {
		if asString, ok := key.(string); ok {
			return asString
		}
		return ""
	}
	return loc.Sprintf(key, args...)
}

func localizeWithFallback(loc Localizer, key string, fallback string) string {
	value := strings.TrimSpace(localize(loc, key))
	if value == "" || value == key {
		return fallback
	}
	return value
}

func normalizeToken(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
