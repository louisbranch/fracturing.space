package i18n

import (
	"net/http"
	"strings"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	sharedi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	_ "github.com/louisbranch/fracturing.space/internal/services/shared/i18nmessages"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web2/platform/errors"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Localizer exposes translated formatting used by web2 templates and handlers.
type Localizer interface {
	Sprintf(key message.Reference, args ...any) string
}

// ResolveTag resolves request language with authenticated/private preference first.
func ResolveTag(r *http.Request, resolveLanguage func(*http.Request) string) language.Tag {
	if resolveLanguage != nil {
		if tag, ok := platformi18n.ParseTag(strings.TrimSpace(resolveLanguage(r))); ok {
			return tag
		}
	}
	tag, _ := sharedi18n.ResolveTag(r)
	return tag
}

// EnsureLanguageCookie syncs the language cookie to the resolved tag.
func EnsureLanguageCookie(w http.ResponseWriter, r *http.Request, tag language.Tag) {
	if w == nil {
		return
	}
	expected := strings.TrimSpace(tag.String())
	if expected == "" {
		return
	}
	if r != nil {
		if cookie, err := r.Cookie(sharedi18n.LangCookieName); err == nil {
			if strings.TrimSpace(cookie.Value) == expected {
				return
			}
		}
	}
	sharedi18n.SetLanguageCookie(w, tag)
}

// ResolveLocalizer resolves a localized printer and language string for a request.
func ResolveLocalizer(w http.ResponseWriter, r *http.Request, resolveLanguage func(*http.Request) string) (*message.Printer, string) {
	tag := ResolveTag(r, resolveLanguage)
	EnsureLanguageCookie(w, r, tag)
	return sharedi18n.Printer(tag), tag.String()
}

// LocalizeError resolves a translated error string when a mapping is available.
func LocalizeError(loc Localizer, err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	if loc == nil {
		return msg
	}
	if key := apperrors.LocalizationKey(err); key != "" {
		return loc.Sprintf(key)
	}
	return msg
}
