package i18n

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sharedi18n "github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type localizerStub struct {
	value string
}

func (s *localizerStub) Sprintf(message.Reference, ...any) string {
	return s.value
}

func TestResolveTagPrefersPrivateResolver(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "fr")

	got := ResolveTag(req, func(*http.Request) string { return " pt-BR " })
	if got.String() != "pt-BR" {
		t.Fatalf("ResolveTag() = %q, want %q", got.String(), "pt-BR")
	}
}

func TestResolveTagFallsBackWhenPrivateResolverInvalid(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "pt-BR")

	got := ResolveTag(req, func(*http.Request) string { return "not-a-tag" })
	if got.String() != "pt-BR" {
		t.Fatalf("ResolveTag() = %q, want %q", got.String(), "pt-BR")
	}
}

func TestEnsureLanguageCookieSkipsWhenAlreadyCurrent(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sharedi18n.LangCookieName, Value: "pt-BR"})
	rr := httptest.NewRecorder()

	EnsureLanguageCookie(rr, req, language.MustParse("pt-BR"))
	if got := rr.Header().Get("Set-Cookie"); got != "" {
		t.Fatalf("Set-Cookie = %q, want empty", got)
	}
}

func TestEnsureLanguageCookieWritesExpectedValue(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	EnsureLanguageCookie(rr, req, language.MustParse("en-US"))
	setCookie := rr.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, sharedi18n.LangCookieName+"=en-US") {
		t.Fatalf("Set-Cookie = %q, want language cookie", setCookie)
	}
}

func TestResolveLocalizerReturnsLanguageAndSetsCookie(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "pt-BR")
	rr := httptest.NewRecorder()

	printer, lang := ResolveLocalizer(rr, req, nil)
	if printer == nil {
		t.Fatalf("ResolveLocalizer() printer = nil, want non-nil")
	}
	if lang != "pt-BR" {
		t.Fatalf("ResolveLocalizer() language = %q, want %q", lang, "pt-BR")
	}
	setCookie := rr.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, sharedi18n.LangCookieName+"=pt-BR") {
		t.Fatalf("Set-Cookie = %q, want language cookie", setCookie)
	}
}

func TestLocalizeErrorUsesLocalizationKeyWhenAvailable(t *testing.T) {
	t.Parallel()

	if got := LocalizeError(nil, nil); got != "" {
		t.Fatalf("LocalizeError(nil,nil) = %q, want empty", got)
	}

	err := apperrors.EK(apperrors.KindUnavailable, "errors.settings.unavailable", "settings service is not configured")
	loc := &localizerStub{value: "Settings service unavailable"}
	if got := LocalizeError(loc, err); got != "Settings service unavailable" {
		t.Fatalf("LocalizeError(localized) = %q, want %q", got, "Settings service unavailable")
	}

	if got := LocalizeError(nil, errors.New("boom")); got != "boom" {
		t.Fatalf("LocalizeError(no-localizer) = %q, want %q", got, "boom")
	}
	if got := LocalizeError(loc, apperrors.E(apperrors.KindUnavailable, "fallback")); got != "fallback" {
		t.Fatalf("LocalizeError(no-key) = %q, want %q", got, "fallback")
	}
}
