package render

import (
	"fmt"
	"testing"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func TestRenderInAppOnboardingWelcomeLocalized(t *testing.T) {
	t.Parallel()

	loc := fakeLocalizer{values: map[string]string{
		"notification.generic.title":            "Notification",
		"notification.generic.body":             "You have a new notification.",
		"notification.signup_method.passkey":    "passkey",
		"notification.signup_method.magic_link": "magic link",
		"notification.signup_method.unknown":    "email",
		"notification.onboarding_welcome.title": "Welcome to Fracturing Space",
		"notification.onboarding_welcome.body":  "Your account is ready. Sign-in method: %s.",
	}}

	out := RenderInApp(loc, Input{
		MessageType: "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":"passkey"}`,
	})

	if out.Title != "Welcome to Fracturing Space" {
		t.Fatalf("title = %q, want %q", out.Title, "Welcome to Fracturing Space")
	}
	if out.BodyText != "Your account is ready. Sign-in method: passkey." {
		t.Fatalf("body = %q, want rendered onboarding body", out.BodyText)
	}
}

func TestRenderInAppMalformedPayloadFallsBack(t *testing.T) {
	t.Parallel()

	loc := fakeLocalizer{values: map[string]string{
		"notification.generic.title": "Notification",
		"notification.generic.body":  "You have a new notification.",
	}}

	out := RenderInApp(loc, Input{
		MessageType: "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":`,
	})

	if out.Title != "Notification" {
		t.Fatalf("title = %q, want %q", out.Title, "Notification")
	}
	if out.BodyText != "You have a new notification." {
		t.Fatalf("body = %q, want %q", out.BodyText, "You have a new notification.")
	}
}

func TestRenderInAppUnknownSignupMethodUsesSafeFallbackLabel(t *testing.T) {
	t.Parallel()

	loc := fakeLocalizer{values: map[string]string{
		"notification.generic.title":            "Notification",
		"notification.generic.body":             "You have a new notification.",
		"notification.onboarding_welcome.title": "Welcome to Fracturing Space",
		"notification.onboarding_welcome.body":  "Your account is ready. Sign-in method: %s.",
	}}

	out := RenderInApp(loc, Input{
		MessageType: "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":"oauth"}`,
	})

	if out.BodyText != "Your account is ready. Sign-in method: another method." {
		t.Fatalf("body = %q, want safe unknown-signup fallback label", out.BodyText)
	}
}

func TestRenderInAppWithNilLocalizerReturnsHumanReadableDefaults(t *testing.T) {
	t.Parallel()

	out := RenderInApp(nil, Input{
		MessageType: "auth.onboarding.welcome",
		PayloadJSON: `{"signup_method":"passkey"}`,
	})

	if out.Title != "Notification" {
		t.Fatalf("title = %q, want %q", out.Title, "Notification")
	}
	if out.BodyText != "You have a new notification." {
		t.Fatalf("body = %q, want %q", out.BodyText, "You have a new notification.")
	}
}

func TestRenderInAppUnknownTopicFallsBack(t *testing.T) {
	t.Parallel()

	loc := fakeLocalizer{values: map[string]string{
		"notification.generic.title": "Notification",
		"notification.generic.body":  "You have a new notification.",
	}}

	out := RenderInApp(loc, Input{
		MessageType: "unknown.topic",
		PayloadJSON: `{}`,
	})

	if out.Title != "Notification" {
		t.Fatalf("title = %q, want %q", out.Title, "Notification")
	}
	if out.BodyText != "You have a new notification." {
		t.Fatalf("body = %q, want %q", out.BodyText, "You have a new notification.")
	}
}

func TestRenderInAppWithRealPrinterUsesRegisteredCatalog(t *testing.T) {
	t.Parallel()

	printer := message.NewPrinter(language.AmericanEnglish)
	out := RenderInApp(printer, Input{
		MessageType: MessageTypeOnboardingWelcome,
		PayloadJSON: `{"signup_method":"passkey"}`,
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
