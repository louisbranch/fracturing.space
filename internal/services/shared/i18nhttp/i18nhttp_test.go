package i18nhttp

import (
	"net/http/httptest"
	"testing"

	"golang.org/x/text/language"
)

func TestResolveTag(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "http://example.com/?lang=pt-BR", nil)
	tag, persist := ResolveTag(req)
	if tag != language.BrazilianPortuguese {
		t.Fatalf("tag = %v, want %v", tag, language.BrazilianPortuguese)
	}
	if !persist {
		t.Fatal("persist = false, want true")
	}
}

func TestBuildLanguageOptions(t *testing.T) {
	t.Parallel()

	options := BuildLanguageOptions(
		[]language.Tag{language.AmericanEnglish, language.BrazilianPortuguese},
		"pt-BR",
		func(tag language.Tag) string { return tag.String() + "-label" },
	)
	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(options))
	}
	if !options[1].Active {
		t.Fatalf("options[1].Active = false, want true")
	}
}

func TestLanguageURL(t *testing.T) {
	t.Parallel()

	got := LanguageURL("/app/campaigns", "page=2", "en-US")
	if got == "" {
		t.Fatal("LanguageURL returned empty string")
	}
	if got != "/app/campaigns?lang=en-US&page=2" && got != "/app/campaigns?page=2&lang=en-US" {
		t.Fatalf("LanguageURL = %q", got)
	}
}
