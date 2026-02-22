package render

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func init() {
	lang := language.English

	message.SetString(lang, "notification.generic.title", defaultGenericTitle)
	message.SetString(lang, "notification.generic.body", defaultGenericBody)
	message.SetString(lang, "notification.generic.email_subject", defaultGenericEmailSubject)
	message.SetString(lang, "notification.signup_method.passkey", "passkey")
	message.SetString(lang, "notification.signup_method.magic_link", "magic link")
	message.SetString(lang, "notification.signup_method.unknown", defaultUnknownSignupMethod)
	message.SetString(lang, "notification.onboarding_welcome.title", "Welcome to Fracturing Space")
	message.SetString(lang, "notification.onboarding_welcome.body", "Your account is ready. Sign-in method: %s.")
	message.SetString(lang, "notification.onboarding_welcome.email_subject", "Welcome to Fracturing Space")
	message.SetString(lang, "notification.onboarding_welcome.email_body", "Your account is ready. Sign-in method: %s.")
}
