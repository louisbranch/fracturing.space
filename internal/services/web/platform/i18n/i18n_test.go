package i18n

import (
	"net/http/httptest"
	"testing"
)

func TestResolveLanguageDefaultsToEnglish(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	if got := ResolveLanguage(req); got != "en" {
		t.Fatalf("ResolveLanguage() = %q, want %q", got, "en")
	}
}

func TestResolveLanguagePrefersAcceptLanguage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "fr-CA,fr;q=0.8,en;q=0.5")
	if got := ResolveLanguage(req); got != "fr" {
		t.Fatalf("ResolveLanguage() = %q, want %q", got, "fr")
	}
}

func TestResolveLanguageTrimsAndNormalizesCase(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", " PT-BR ;q=0.9")
	if got := ResolveLanguage(req); got != "pt" {
		t.Fatalf("ResolveLanguage() = %q, want %q", got, "pt")
	}
}

func TestResolveLanguageFallsBackForMalformedHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", ";q=0.8")
	if got := ResolveLanguage(req); got != "en" {
		t.Fatalf("ResolveLanguage() = %q, want %q", got, "en")
	}
}

func TestResolveLanguageFallsBackForSingleCharacterTag(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "e")
	if got := ResolveLanguage(req); got != "en" {
		t.Fatalf("ResolveLanguage() = %q, want %q", got, "en")
	}
}
