package render

import (
	"encoding/json"
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	_ "github.com/louisbranch/fracturing.space/internal/platform/i18n/catalog"
	notificationsdomain "github.com/louisbranch/fracturing.space/internal/services/notifications/domain"
	"github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
	"golang.org/x/text/message"
)

const (
	// MessageTypeOnboardingWelcome aliases the canonical onboarding welcome template id.
	MessageTypeOnboardingWelcome = notificationsdomain.MessageTypeOnboardingWelcome
	// MessageTypeOnboardingWelcomeV1 is the versioned onboarding welcome template id.
	MessageTypeOnboardingWelcomeV1 = notificationsdomain.MessageTypeOnboardingWelcomeV1

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
	MessageType string
	PayloadJSON string
	Channel     Channel
}

// Output is localized, channel-aware copy derived from one notification artifact.
type Output struct {
	Title        string
	BodyText     string
	Facts        []OutputFact
	Actions      []OutputAction
	EmailSubject string
}

// OutputFact is localized detail metadata ready for one notification channel.
type OutputFact struct {
	Label string
	Value string
}

// OutputAction is one localized notification CTA ready for one channel.
type OutputAction struct {
	Label    string
	Kind     string
	TargetID string
	Method   string
	Style    string
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
	if input.Channel == ChannelInApp {
		if payload, ok := notificationpayload.ParseInAppPayload(input.PayloadJSON); ok {
			return outputFromInAppPayload(loc, payload)
		}
	}
	switch normalizeToken(input.MessageType) {
	case MessageTypeOnboardingWelcome, MessageTypeOnboardingWelcomeV1:
		return renderOnboardingWelcome(loc, input)
	default:
		return genericOutput(loc)
	}
}

func outputFromInAppPayload(loc Localizer, payload notificationpayload.InAppPayload) Output {
	facts := make([]OutputFact, 0, len(payload.Facts))
	for _, fact := range payload.Facts {
		facts = append(facts, OutputFact{
			Label: platformi18n.ResolveCopy(loc, fact.Label),
			Value: fact.Value,
		})
	}
	actions := make([]OutputAction, 0, len(payload.Actions))
	for _, action := range payload.Actions {
		actions = append(actions, OutputAction{
			Label:    platformi18n.ResolveCopy(loc, action.Label),
			Kind:     action.Kind,
			TargetID: action.TargetID,
			Method:   action.Method,
			Style:    action.Style,
		})
	}
	return Output{
		Title:    platformi18n.ResolveCopy(loc, payload.Title),
		BodyText: platformi18n.ResolveCopy(loc, payload.Body),
		Facts:    facts,
		Actions:  actions,
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
