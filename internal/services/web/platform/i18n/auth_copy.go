package i18n

import (
	"fmt"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// AuthCopy holds translatable copy for web public auth pages.
type AuthCopy struct {
	MetaDescription     string
	LandingTitle        string
	LandingTagline      string
	LandingSignIn       string
	LandingDocs         string
	LandingGitHub       string
	LoginTitle          string
	LoginHeading        string
	LoginUsername       string
	LoginJoinTitle      string
	LoginCreatePasskey  string
	LoginDivider        string
	LoginLogInTitle     string
	LoginSignInPasskey  string
	JSLoginStartError   string
	JSLoginFinishError  string
	JSRegisterStartErr  string
	JSRegisterFinishErr string
	JSPasskeyFailed     string
	JSLoginUsernameReq  string
	JSRegisterUserReq   string
	JSPasskeyCreated    string
	JSRegisterFailed    string
}

const authAppDisplayName = "Fracturing.Space"

// Auth returns localized auth copy for the provided language tag.
func Auth(tag language.Tag) AuthCopy {
	localizedTag := normalizeAuthTag(tag)
	loc := message.NewPrinter(localizedTag)

	landingTitle := localizeWithFallback(loc, "title.landing", "Open source AI GM engine")
	loginTitle := localizeWithFallback(loc, "title.login", "Sign In")

	return AuthCopy{
		MetaDescription:     localizeWithFallback(loc, "meta.description", "Open-source, server-authoritative engine for deterministic tabletop RPG campaigns and AI game masters."),
		LandingTitle:        withProductSuffix(landingTitle),
		LandingTagline:      localizeWithFallback(loc, "landing.tagline", "Open-source, server-authoritative engine for deterministic tabletop RPG campaigns and AI game masters."),
		LandingSignIn:       localizeWithFallback(loc, "landing.sign_in", "Sign in"),
		LandingDocs:         localizeWithFallback(loc, "landing.docs", "Docs"),
		LandingGitHub:       localizeWithFallback(loc, "landing.github", "GitHub"),
		LoginTitle:          withProductSuffix(loginTitle),
		LoginHeading:        localizeWithFallback(loc, "login.heading", "Welcome to %s", authAppDisplayName),
		LoginUsername:       localizeWithFallback(loc, "login.username", "Username"),
		LoginJoinTitle:      localizeWithFallback(loc, "login.join_title", "Join"),
		LoginCreatePasskey:  localizeWithFallback(loc, "login.create_passkey", "Create Account With Passkey"),
		LoginDivider:        localizeWithFallback(loc, "login.divider", "Or"),
		LoginLogInTitle:     localizeWithFallback(loc, "login.log_in_title", "Log in"),
		LoginSignInPasskey:  localizeWithFallback(loc, "login.sign_in_passkey", "Log In With Passkey"),
		JSLoginStartError:   localizeWithFallback(loc, "login.js.login_start_error", "Unable to start passkey login."),
		JSLoginFinishError:  localizeWithFallback(loc, "login.js.login_finish_error", "Unable to finish passkey login."),
		JSRegisterStartErr:  localizeWithFallback(loc, "login.js.register_start_error", "Unable to start passkey registration."),
		JSRegisterFinishErr: localizeWithFallback(loc, "login.js.register_finish_error", "Unable to finish passkey registration."),
		JSPasskeyFailed:     localizeWithFallback(loc, "login.js.passkey_failed", "Passkey login failed."),
		JSLoginUsernameReq:  localizeWithFallback(loc, "login.js.login_username_required", "Username is required to log in."),
		JSRegisterUserReq:   localizeWithFallback(loc, "login.js.register_username_required", "Username is required to create an account."),
		JSPasskeyCreated:    localizeWithFallback(loc, "login.js.passkey_created", "Passkey created; signing you in."),
		JSRegisterFailed:    localizeWithFallback(loc, "login.js.register_failed", "Passkey registration failed."),
	}
}

// normalizeAuthTag centralizes this web behavior in one helper seam.
func normalizeAuthTag(tag language.Tag) language.Tag {
	if tag == language.MustParse("pt-BR") {
		return language.MustParse("pt-BR")
	}
	base, _ := tag.Base()
	portugueseBase, _ := language.Portuguese.Base()
	if base == portugueseBase {
		return language.MustParse("pt-BR")
	}
	return language.MustParse("en-US")
}

// withProductSuffix centralizes this web behavior in one helper seam.
func withProductSuffix(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return authAppDisplayName
	}
	return fmt.Sprintf("%s | %s", trimmed, authAppDisplayName)
}

// localizeWithFallback centralizes this web behavior in one helper seam.
func localizeWithFallback(loc *message.Printer, key string, fallback string, args ...any) string {
	if loc != nil {
		value := strings.TrimSpace(loc.Sprintf(key, args...))
		if value != "" && value != key {
			return value
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(fallback, args...)
	}
	return fallback
}
