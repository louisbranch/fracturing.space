package render

import (
	"encoding/json"
	"strings"

	_ "github.com/louisbranch/fracturing.space/internal/platform/i18n/catalog"
	"golang.org/x/text/message"
)

const (
	// MessageTypeOnboardingWelcome is the canonical onboarding welcome copy id.
	MessageTypeOnboardingWelcome = "auth.onboarding.welcome"
	// MessageTypeOnboardingWelcomeV1 is the versioned onboarding welcome copy id.
	MessageTypeOnboardingWelcomeV1 = "auth.onboarding.welcome.v1"

	defaultGenericTitle        = "Notification"
	defaultGenericBody         = "You have a new notification."
	defaultUnknownSignupMethod = "another method"
)

// Input is one in-app notification copy render request.
type Input struct {
	MessageType string
	PayloadJSON string
}

// Output is localized in-app copy derived from one notification artifact.
type Output struct {
	Title    string
	BodyText string
}

// Localizer is the minimal message-printer contract required by the renderer.
type Localizer interface {
	Sprintf(key message.Reference, args ...any) string
}

// onboardingPayload mirrors the only structured payload the web inbox
// currently needs to interpret for localized copy.
type onboardingPayload struct {
	SignupMethod string `json:"signup_method"`
}

// RenderInApp returns localized title/body copy for one in-app notification.
func RenderInApp(loc Localizer, input Input) Output {
	switch normalizeToken(input.MessageType) {
	case MessageTypeOnboardingWelcome, MessageTypeOnboardingWelcomeV1:
		return renderOnboardingWelcome(loc, input)
	default:
		return genericOutput(loc)
	}
}

// renderOnboardingWelcome keeps onboarding-specific payload parsing and
// localization isolated from generic notification fallback behavior.
func renderOnboardingWelcome(loc Localizer, input Input) Output {
	payload := onboardingPayload{}
	if raw := strings.TrimSpace(input.PayloadJSON); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return genericOutput(loc)
		}
	}

	signupMethod := localizedSignupMethod(loc, payload.SignupMethod)
	title := localize(loc, "notification.onboarding_welcome.title")
	body := localize(loc, "notification.onboarding_welcome.body", signupMethod)
	if title == "notification.onboarding_welcome.title" || body == "notification.onboarding_welcome.body" {
		return genericOutput(loc)
	}

	return Output{
		Title:    title,
		BodyText: body,
	}
}

// genericOutput provides stable human-readable fallback copy when message-type
// specific rendering is unavailable or localization is missing.
func genericOutput(loc Localizer) Output {
	return Output{
		Title:    localizeWithFallback(loc, "notification.generic.title", defaultGenericTitle),
		BodyText: localizeWithFallback(loc, "notification.generic.body", defaultGenericBody),
	}
}

// localizedSignupMethod translates producer signup-method tokens into
// user-facing labels without leaking raw internal identifiers.
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

// localize centralizes nil-safe access to the message printer so callers can
// fall back cleanly when no request localizer is available.
func localize(loc Localizer, key message.Reference, args ...any) string {
	if loc == nil {
		if asString, ok := key.(string); ok {
			return asString
		}
		return ""
	}
	return loc.Sprintf(key, args...)
}

// localizeWithFallback preserves reader-friendly copy when the localized key is
// missing or resolves to the untranslated lookup token.
func localizeWithFallback(loc Localizer, key string, fallback string) string {
	value := strings.TrimSpace(localize(loc, key))
	if value == "" || value == key {
		return fallback
	}
	return value
}

// normalizeToken keeps message-type and payload token matching case-insensitive
// and whitespace-safe at the render boundary.
func normalizeToken(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
