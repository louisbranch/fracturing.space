package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/web/principal"
)

func TestLanguageResolverNilClientReturnsFallback(t *testing.T) {
	t.Parallel()

	r := principal.New(principal.Dependencies{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	lang := r.ResolveLanguage(req)

	if lang != "en-US" {
		t.Fatalf("ResolveLanguage = %q, want %q", lang, "en-US")
	}
}

func TestLanguageResolverAnonymousReturnsFallback(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{
			Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR},
		},
	}
	r := principal.New(principal.Dependencies{AccountClient: account})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	lang := r.ResolveLanguage(req)

	if lang != "en-US" {
		t.Fatalf("ResolveLanguage = %q, want %q for anonymous user", lang, "en-US")
	}
}

func TestLanguageResolverUsesAccountLocale(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{
			Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_PT_BR},
		},
	}
	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{SessionClient: auth, AccountClient: account})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	lang := r.ResolveLanguage(req)

	if lang != "pt-BR" {
		t.Fatalf("ResolveLanguage = %q, want %q", lang, "pt-BR")
	}
}

func TestLanguageResolverUnspecifiedLocaleReturnsFallback(t *testing.T) {
	t.Parallel()

	account := &fakeAccountClient{
		getProfileResp: &authv1.GetProfileResponse{
			Profile: &authv1.AccountProfile{Locale: commonv1.Locale_LOCALE_UNSPECIFIED},
		},
	}
	auth := newFakeWebAuthClient()
	r := principal.New(principal.Dependencies{SessionClient: auth, AccountClient: account})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	attachSessionCookie(t, req, auth, "user-1")
	lang := r.ResolveLanguage(req)

	if lang != "en-US" {
		t.Fatalf("ResolveLanguage = %q, want %q", lang, "en-US")
	}
}
