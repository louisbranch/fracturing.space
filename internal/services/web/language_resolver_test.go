package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

func TestLanguageResolverNilClientReturnsFallback(t *testing.T) {
	t.Parallel()

	r := newLanguageResolver(nil, func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	lang := r.resolveRequestLanguage(req)

	if lang != "en-US" {
		t.Fatalf("resolveRequestLanguage = %q, want %q", lang, "en-US")
	}
}

func TestLanguageResolverAnonymousReturnsFallback(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{
			Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR},
		},
	}
	r := newLanguageResolver(account, func(*http.Request) string { return "" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	lang := r.resolveRequestLanguage(req)

	if lang != "en-US" {
		t.Fatalf("resolveRequestLanguage = %q, want %q for anonymous user", lang, "en-US")
	}
}

func TestLanguageResolverUsesAccountLocale(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{
			Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR},
		},
	}
	r := newLanguageResolver(account, func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	lang := r.resolveRequestLanguage(req)

	if lang != "pt-BR" {
		t.Fatalf("resolveRequestLanguage = %q, want %q", lang, "pt-BR")
	}
}

func TestLanguageResolverUnspecifiedLocaleReturnsFallback(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{
			Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_UNSPECIFIED},
		},
	}
	r := newLanguageResolver(account, func(*http.Request) string { return "user-1" })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	lang := r.resolveRequestLanguage(req)

	if lang != "en-US" {
		t.Fatalf("resolveRequestLanguage = %q, want %q", lang, "en-US")
	}
}
