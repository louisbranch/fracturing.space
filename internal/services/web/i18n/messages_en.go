package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func init() {
	lang := language.English

	// Landing page
	message.SetString(lang, "title.landing", "%s | Open source AI GM engine")
	message.SetString(lang, "landing.tagline", "Open-source, server-authoritative engine for deterministic tabletop RPG campaigns and AI game masters.")
	message.SetString(lang, "landing.signed_in_as", "Signed in as")
	message.SetString(lang, "landing.sign_out", "Sign out")
	message.SetString(lang, "landing.sign_in", "Sign in")
	message.SetString(lang, "landing.docs", "Docs")
	message.SetString(lang, "landing.github", "GitHub")
	message.SetString(lang, "meta.description", "Open-source, server-authoritative engine for deterministic tabletop RPG campaigns and AI game masters.")

	// Login page
	message.SetString(lang, "title.login", "%s | Sign In")
	message.SetString(lang, "login.heading", "Sign in to continue")
	message.SetString(lang, "login.requesting_access", "%s (%s) is requesting access to your account.")
	message.SetString(lang, "login.card_title", "Account Access")
	message.SetString(lang, "login.card_subtitle", "Create an account or sign in with a passkey.")
	message.SetString(lang, "login.email", "Primary email")
	message.SetString(lang, "login.create_passkey", "Create Account With Passkey")
	message.SetString(lang, "login.divider", "returning?")
	message.SetString(lang, "login.sign_in_passkey", "Sign In With Passkey")

	// Login JS strings (via data attributes)
	message.SetString(lang, "login.js.missing_session", "Missing login session.")
	message.SetString(lang, "login.js.passkey_failed", "Passkey login failed.")
	message.SetString(lang, "login.js.email_required", "Primary email is required.")
	message.SetString(lang, "login.js.passkey_created", "Passkey created. You can now sign in.")
	message.SetString(lang, "login.js.register_failed", "Passkey registration failed.")
	message.SetString(lang, "login.js.login_start_error", "Unable to start passkey login.")
	message.SetString(lang, "login.js.login_finish_error", "Unable to finish passkey login.")
	message.SetString(lang, "login.js.register_start_error", "An error occurred creating your account. If you already have an account, use Sign In With Passkey below.")
	message.SetString(lang, "login.js.register_finish_error", "Unable to finish passkey registration.")

	// Magic page
	message.SetString(lang, "magic.unavailable.title", "Magic link unavailable")
	message.SetString(lang, "magic.unavailable.message", "We could not reach the authentication service.")
	message.SetString(lang, "magic.unavailable.detail", "Please try again in a moment.")
	message.SetString(lang, "magic.missing.title", "Magic link missing")
	message.SetString(lang, "magic.missing.message", "This link is missing its token.")
	message.SetString(lang, "magic.missing.detail", "Please request a new magic link and try again.")
	message.SetString(lang, "magic.invalid.title", "Magic link invalid")
	message.SetString(lang, "magic.invalid.message", "We could not validate this magic link.")
	message.SetString(lang, "magic.invalid.detail", "It may have expired or already been used.")
	message.SetString(lang, "magic.verified.title", "Magic link verified")
	message.SetString(lang, "magic.verified.message", "Your link is valid and your email has been confirmed.")
	message.SetString(lang, "magic.verified.detail", "You can return to the app and continue sign in.")
	message.SetString(lang, "magic.verified.link", "Return to the app")

	// Language nav
	message.SetString(lang, "nav.lang_en", "EN")
	message.SetString(lang, "nav.lang_pt_br", "PT-BR")
}
