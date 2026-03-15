package render

import (
	"fmt"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/shared/notificationpayload"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func TestRenderOnboardingWelcomeInAppLocalized(t *testing.T) {
	t.Parallel()

	loc := fakeLocalizer{values: map[string]string{
		"notification.generic.title":                    "Notification",
		"notification.generic.body":                     "You have a new notification.",
		"notification.signup_method.passkey":            "passkey",
		"notification.signup_method.magic_link":         "magic link",
		"notification.signup_method.unknown":            "email",
		"notification.onboarding_welcome.title":         "Welcome to Fracturing Space",
		"notification.onboarding_welcome.body":          "Your account is ready. Sign-in method: %s.",
		"notification.onboarding_welcome.email_subject": "Welcome to Fracturing Space",
		"notification.onboarding_welcome.email_body":    "Your account is ready. Sign-in method: %s.",
	}}

	out := Render(loc, Input{
		MessageType: "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":"passkey"}`,
		Channel:     ChannelInApp,
	})

	if out.Title != "Welcome to Fracturing Space" {
		t.Fatalf("title = %q, want %q", out.Title, "Welcome to Fracturing Space")
	}
	if out.BodyText != "Your account is ready. Sign-in method: passkey." {
		t.Fatalf("body = %q, want rendered onboarding body", out.BodyText)
	}
}

func TestRenderOnboardingWelcomeEmailLocalized(t *testing.T) {
	t.Parallel()

	loc := fakeLocalizer{values: map[string]string{
		"notification.generic.title":                    "Notificacao",
		"notification.generic.body":                     "Voce tem uma notificacao.",
		"notification.signup_method.passkey":            "chave de acesso",
		"notification.signup_method.magic_link":         "link magico",
		"notification.signup_method.unknown":            "email",
		"notification.onboarding_welcome.title":         "Boas-vindas ao Fracturing Space",
		"notification.onboarding_welcome.body":          "Sua conta esta pronta. Metodo de entrada: %s.",
		"notification.onboarding_welcome.email_subject": "Boas-vindas ao Fracturing Space",
		"notification.onboarding_welcome.email_body":    "Sua conta esta pronta. Metodo de entrada: %s.",
	}}

	out := Render(loc, Input{
		MessageType: "auth.onboarding.welcome.v1",
		PayloadJSON: `{"signup_method":"magic_link"}`,
		Channel:     ChannelEmail,
	})

	if out.EmailSubject != "Boas-vindas ao Fracturing Space" {
		t.Fatalf("email subject = %q, want %q", out.EmailSubject, "Boas-vindas ao Fracturing Space")
	}
	if out.BodyText != "Sua conta esta pronta. Metodo de entrada: link magico." {
		t.Fatalf("body = %q, want rendered onboarding email body", out.BodyText)
	}
}

func TestRenderOnboardingWelcomeMalformedPayloadFallsBack(t *testing.T) {
	t.Parallel()

	loc := fakeLocalizer{values: map[string]string{
		"notification.generic.title": "Notification",
		"notification.generic.body":  "You have a new notification.",
	}}

	out := Render(loc, Input{
		MessageType: "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":`,
		Channel:     ChannelInApp,
	})

	if out.Title != "Notification" {
		t.Fatalf("title = %q, want %q", out.Title, "Notification")
	}
	if out.BodyText != "You have a new notification." {
		t.Fatalf("body = %q, want %q", out.BodyText, "You have a new notification.")
	}
}

func TestRenderOnboardingWelcomeUnknownSignupMethodUsesSafeFallbackLabel(t *testing.T) {
	t.Parallel()

	loc := fakeLocalizer{values: map[string]string{
		"notification.generic.title":                    "Notification",
		"notification.generic.body":                     "You have a new notification.",
		"notification.onboarding_welcome.title":         "Welcome to Fracturing Space",
		"notification.onboarding_welcome.body":          "Your account is ready. Sign-in method: %s.",
		"notification.onboarding_welcome.email_subject": "Welcome to Fracturing Space",
		"notification.onboarding_welcome.email_body":    "Your account is ready. Sign-in method: %s.",
	}}

	out := Render(loc, Input{
		MessageType: "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":"oauth"}`,
		Channel:     ChannelInApp,
	})

	if out.BodyText != "Your account is ready. Sign-in method: another method." {
		t.Fatalf("body = %q, want safe unknown-signup fallback label", out.BodyText)
	}
}

func TestRenderWithNilLocalizerReturnsHumanReadableDefaults(t *testing.T) {
	t.Parallel()

	out := Render(nil, Input{
		MessageType: "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":"passkey"}`,
		Channel:     ChannelInApp,
	})

	if out.Title != "Notification" {
		t.Fatalf("title = %q, want %q", out.Title, "Notification")
	}
	if out.BodyText != "You have a new notification." {
		t.Fatalf("body = %q, want %q", out.BodyText, "You have a new notification.")
	}
	if out.EmailSubject != "Fracturing Space notification" {
		t.Fatalf("email subject = %q, want %q", out.EmailSubject, "Fracturing Space notification")
	}
}

func TestRenderUnknownTopicFallsBack(t *testing.T) {
	t.Parallel()

	loc := fakeLocalizer{values: map[string]string{
		"notification.generic.title": "Notification",
		"notification.generic.body":  "You have a new notification.",
	}}

	out := Render(loc, Input{
		MessageType: "unknown.topic",
		PayloadJSON: `{}`,
		Channel:     ChannelInApp,
	})

	if out.Title != "Notification" {
		t.Fatalf("title = %q, want %q", out.Title, "Notification")
	}
	if out.BodyText != "You have a new notification." {
		t.Fatalf("body = %q, want %q", out.BodyText, "You have a new notification.")
	}
}

func TestRenderInAppUsesCanonicalPayload(t *testing.T) {
	t.Parallel()

	out := Render(nil, Input{
		MessageType: "campaign.invite.created.v1",
		PayloadJSON: `{"title":"Campaign invitation","body":"This invitation is addressed to you. You can accept or decline it now.","facts":[{"label":"Campaign","value":"Skyfall"},{"label":"Seat","value":"Scout"}],"actions":[{"label":"View invitation","kind":"public_invite_view","target_id":"inv-1","method":"GET","style":"primary"}]}`,
		Channel:     ChannelInApp,
	})

	if out.Title != "Campaign invitation" {
		t.Fatalf("title = %q, want %q", out.Title, "Campaign invitation")
	}
	if out.BodyText != "This invitation is addressed to you. You can accept or decline it now." {
		t.Fatalf("body = %q, want canonical payload body", out.BodyText)
	}
	if len(out.Facts) != 2 || out.Facts[0].Value != "Skyfall" {
		t.Fatalf("facts = %+v, want canonical payload facts", out.Facts)
	}
	if len(out.Actions) != 1 || out.Actions[0].Kind != notificationpayload.ActionKindPublicInviteView || out.Actions[0].TargetID != "inv-1" || out.Actions[0].Method != notificationpayload.ActionMethodGet {
		t.Fatalf("actions = %+v, want canonical payload action", out.Actions)
	}
}

func TestRenderInAppMalformedCanonicalPayloadFallsBack(t *testing.T) {
	t.Parallel()

	out := Render(nil, Input{
		MessageType: "campaign.invite.created.v1",
		PayloadJSON: `{"title":"Campaign invitation","actions":[{"label":"Accept invitation"}`,
		Channel:     ChannelInApp,
	})

	if out.Title != "Notification" || out.BodyText != "You have a new notification." {
		t.Fatalf("fallback = %+v, want generic copy", out)
	}
}

func TestRenderOnboardingWelcomeWithRealPrinterUsesRegisteredCatalog(t *testing.T) {
	t.Parallel()

	printer := message.NewPrinter(language.AmericanEnglish)
	out := Render(printer, Input{
		MessageType: MessageTypeOnboardingWelcome,
		PayloadJSON: `{"signup_method":"passkey"}`,
		Channel:     ChannelInApp,
	})

	if out.Title != "Welcome to Fracturing Space" {
		t.Fatalf("title = %q, want %q", out.Title, "Welcome to Fracturing Space")
	}
	if out.BodyText != "Your account is ready. Sign-in method: passkey." {
		t.Fatalf("body = %q, want %q", out.BodyText, "Your account is ready. Sign-in method: passkey.")
	}
}

type fakeLocalizer struct {
	values map[string]string
}

func (f fakeLocalizer) Sprintf(key message.Reference, args ...any) string {
	asString, ok := key.(string)
	if !ok {
		return ""
	}
	template := f.values[asString]
	if template == "" {
		return asString
	}
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}
