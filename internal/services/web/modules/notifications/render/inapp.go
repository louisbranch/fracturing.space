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
	MessageTypeOnboardingWelcomeV1    = "auth.onboarding.welcome.v1"
	MessageTypeCampaignInviteCreated  = "campaign.invite.created.v1"
	MessageTypeCampaignInviteAccepted = "campaign.invite.accepted.v1"
	MessageTypeCampaignInviteDeclined = "campaign.invite.declined.v1"

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

// invitePayload mirrors the structured invite-notification fields used in web copy.
type invitePayload struct {
	CampaignName      string `json:"campaign_name"`
	ParticipantName   string `json:"participant_name"`
	InviterUsername   string `json:"inviter_username"`
	RecipientUsername string `json:"recipient_username"`
}

// RenderInApp returns localized title/body copy for one in-app notification.
func RenderInApp(loc Localizer, input Input) Output {
	switch normalizeToken(input.MessageType) {
	case MessageTypeOnboardingWelcome, MessageTypeOnboardingWelcomeV1:
		return renderOnboardingWelcome(loc, input)
	case MessageTypeCampaignInviteCreated:
		return renderCampaignInviteCreated(loc, input)
	case MessageTypeCampaignInviteAccepted:
		return renderCampaignInviteAccepted(loc, input)
	case MessageTypeCampaignInviteDeclined:
		return renderCampaignInviteDeclined(loc, input)
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

// renderCampaignInviteCreated formats recipient-facing invite creation copy.
func renderCampaignInviteCreated(loc Localizer, input Input) Output {
	payload, ok := parseInvitePayload(input.PayloadJSON)
	if !ok {
		return genericOutput(loc)
	}
	body := "You were invited"
	if payload.CampaignName != "" && payload.ParticipantName != "" {
		body = "You were invited to " + payload.CampaignName + " as " + payload.ParticipantName + "."
	} else if payload.CampaignName != "" {
		body = "You were invited to " + payload.CampaignName + "."
	}
	if payload.InviterUsername != "" {
		body += " Invited by @" + payload.InviterUsername + "."
	}
	return Output{
		Title:    "Campaign invitation",
		BodyText: body,
	}
}

// renderCampaignInviteAccepted formats creator-facing acceptance copy.
func renderCampaignInviteAccepted(loc Localizer, input Input) Output {
	payload, ok := parseInvitePayload(input.PayloadJSON)
	if !ok {
		return genericOutput(loc)
	}
	body := "An invitation was accepted."
	if payload.RecipientUsername != "" && payload.ParticipantName != "" && payload.CampaignName != "" {
		body = "@" + payload.RecipientUsername + " accepted " + payload.ParticipantName + " in " + payload.CampaignName + "."
	}
	return Output{
		Title:    "Invitation accepted",
		BodyText: body,
	}
}

// renderCampaignInviteDeclined formats creator-facing decline copy.
func renderCampaignInviteDeclined(loc Localizer, input Input) Output {
	payload, ok := parseInvitePayload(input.PayloadJSON)
	if !ok {
		return genericOutput(loc)
	}
	body := "An invitation was declined."
	if payload.RecipientUsername != "" && payload.ParticipantName != "" && payload.CampaignName != "" {
		body = "@" + payload.RecipientUsername + " declined " + payload.ParticipantName + " in " + payload.CampaignName + "."
	}
	return Output{
		Title:    "Invitation declined",
		BodyText: body,
	}
}

// parseInvitePayload trims and validates invite notification payloads before rendering.
func parseInvitePayload(raw string) (invitePayload, bool) {
	payload := invitePayload{}
	if raw == "" {
		return invitePayload{}, false
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return invitePayload{}, false
	}
	payload.CampaignName = strings.TrimSpace(payload.CampaignName)
	payload.ParticipantName = strings.TrimSpace(payload.ParticipantName)
	payload.InviterUsername = strings.TrimSpace(payload.InviterUsername)
	payload.RecipientUsername = strings.TrimSpace(payload.RecipientUsername)
	return payload, payload.CampaignName != "" || payload.ParticipantName != "" || payload.InviterUsername != "" || payload.RecipientUsername != ""
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
