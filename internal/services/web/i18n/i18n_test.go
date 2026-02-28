package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/louisbranch/fracturing.space/internal/services/notifications/render"
	"golang.org/x/text/language"
)

func TestResolveTagPrecedence(t *testing.T) {
	t.Run("query param wins", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/?lang=pt-BR", nil)
		req.Header.Set("Accept-Language", "en")
		req.AddCookie(&http.Cookie{Name: LangCookieName, Value: "en"})

		tag, persist := ResolveTag(req)
		if tag.String() != "pt-BR" {
			t.Fatalf("expected pt-BR, got %s", tag.String())
		}
		if !persist {
			t.Fatalf("expected persist to be true")
		}
	})

	t.Run("cookie wins over accept-language", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		req.Header.Set("Accept-Language", "pt-BR")
		req.AddCookie(&http.Cookie{Name: LangCookieName, Value: "en"})

		tag, persist := ResolveTag(req)
		if tag.String() != "en-US" {
			t.Fatalf("expected en-US, got %s", tag.String())
		}
		if persist {
			t.Fatalf("expected persist to be false")
		}
	})

	t.Run("accept-language fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		req.Header.Set("Accept-Language", "pt-BR, en;q=0.9")

		tag, persist := ResolveTag(req)
		if tag.String() != "pt-BR" {
			t.Fatalf("expected pt-BR, got %s", tag.String())
		}
		if persist {
			t.Fatalf("expected persist to be false")
		}
	})
}

func TestResolveTagInvalidValues(t *testing.T) {
	t.Run("invalid query param falls back", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/?lang=not-a-lang", nil)
		req.Header.Set("Accept-Language", "pt-BR")

		tag, _ := ResolveTag(req)
		if tag.String() != "pt-BR" {
			t.Fatalf("expected pt-BR, got %s", tag.String())
		}
	})

	t.Run("unsupported cookie falls back", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		req.AddCookie(&http.Cookie{Name: LangCookieName, Value: "fr"})

		tag, _ := ResolveTag(req)
		if tag.String() != Default().String() {
			t.Fatalf("expected default, got %s", tag.String())
		}
	})
}

func TestSetLanguageCookieNilSafe(t *testing.T) {
	// Should not panic when called with nil ResponseWriter.
	SetLanguageCookie(nil, Default())
}

func TestSetLanguageCookie(t *testing.T) {
	recorder := httptest.NewRecorder()
	SetLanguageCookie(recorder, Default())
	response := recorder.Result()

	cookies := response.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != LangCookieName {
		t.Fatalf("expected cookie name %s, got %s", LangCookieName, cookie.Name)
	}
	if cookie.Value != Default().String() {
		t.Fatalf("expected cookie value %s, got %s", Default().String(), cookie.Value)
	}
	if cookie.Path != "/" {
		t.Fatalf("expected path /, got %s", cookie.Path)
	}
	if cookie.MaxAge <= 0 {
		t.Fatalf("expected MaxAge to be set")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
}

func TestNotificationRendererMessageKeys(t *testing.T) {
	t.Parallel()

	type keyCheck struct {
		key  string
		args []any
	}
	checks := []keyCheck{
		{key: "notification.generic.title"},
		{key: "notification.generic.body"},
		{key: "notification.generic.email_subject"},
		{key: "notification.signup_method.passkey"},
		{key: "notification.signup_method.magic_link"},
		{key: "notification.signup_method.unknown"},
		{key: "notification.onboarding_welcome.title"},
		{key: "notification.onboarding_welcome.body", args: []any{"passkey"}},
		{key: "notification.onboarding_welcome.email_subject"},
		{key: "notification.onboarding_welcome.email_body", args: []any{"passkey"}},
	}

	for _, tag := range []language.Tag{language.AmericanEnglish, language.MustParse("pt-BR")} {
		printer := Printer(tag)
		for _, check := range checks {
			got := printer.Sprintf(check.key, check.args...)
			if got == "" {
				t.Fatalf("tag %s key %q returned empty string", tag, check.key)
			}
			if got == check.key {
				t.Fatalf("tag %s key %q is missing translation", tag, check.key)
			}
		}
	}
}

func TestParticipantRoleLabelPTBR(t *testing.T) {
	t.Parallel()

	printer := Printer(language.MustParse("pt-BR"))
	got := printer.Sprintf("game.participants.value.gm")
	if got != "MJ" {
		t.Fatalf("game.participants.value.gm = %q, want %q", got, "MJ")
	}
}
