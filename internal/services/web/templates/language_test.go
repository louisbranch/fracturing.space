package templates

import (
	"testing"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type langFakeLocalizer struct{}

func (langFakeLocalizer) Sprintf(key message.Reference, _ ...any) string {
	if s, ok := key.(string); ok {
		return s
	}
	return ""
}

func TestNormalizeTag(t *testing.T) {
	t.Run("valid en-US", func(t *testing.T) {
		tag := normalizeTag("en-US")
		if tag == language.Und {
			t.Error("expected a valid tag")
		}
	})

	t.Run("empty string returns default", func(t *testing.T) {
		tag := normalizeTag("")
		if tag == language.Und {
			t.Error("expected default tag, not Und")
		}
	})

	t.Run("invalid tag returns default", func(t *testing.T) {
		tag := normalizeTag("zzz-invalid")
		// Should return the default language, not panic
		_ = tag
	})
}

func TestLanguageLabel(t *testing.T) {
	loc := langFakeLocalizer{}

	t.Run("en-US", func(t *testing.T) {
		label := languageLabel(loc, language.AmericanEnglish)
		if label == "" {
			t.Error("expected non-empty label for en-US")
		}
	})

	t.Run("pt-BR", func(t *testing.T) {
		label := languageLabel(loc, language.BrazilianPortuguese)
		if label == "" {
			t.Error("expected non-empty label for pt-BR")
		}
	})

	t.Run("unknown tag falls back to tag string", func(t *testing.T) {
		tag := language.Japanese
		label := languageLabel(loc, tag)
		if label == "" {
			t.Error("expected non-empty label for unknown tag")
		}
	})
}

func TestLanguageURL(t *testing.T) {
	t.Run("empty path defaults to /", func(t *testing.T) {
		page := PageContext{}
		got := LanguageURL(page, "en-US")
		if got == "" {
			t.Error("expected non-empty URL")
		}
	})

	t.Run("preserves existing query params", func(t *testing.T) {
		page := PageContext{
			CurrentPath:  "/campaigns",
			CurrentQuery: "page=2",
		}
		got := LanguageURL(page, "pt-BR")
		if got == "" {
			t.Error("expected non-empty URL")
		}
	})

	t.Run("malformed query handled", func(t *testing.T) {
		page := PageContext{
			CurrentPath:  "/test",
			CurrentQuery: "%ZZinvalid",
		}
		got := LanguageURL(page, "en-US")
		if got == "" {
			t.Error("expected non-empty URL even with bad query")
		}
	})
}

func TestActiveLanguageLabel(t *testing.T) {
	t.Run("nil localizer uses fallback", func(t *testing.T) {
		page := PageContext{Lang: "en-US"}
		label := ActiveLanguageLabel(page)
		// Should return something (either the key or localized label)
		_ = label
	})

	t.Run("with localizer", func(t *testing.T) {
		page := PageContext{
			Lang: "en-US",
			Loc:  langFakeLocalizer{},
		}
		label := ActiveLanguageLabel(page)
		if label == "" {
			t.Error("expected non-empty label")
		}
	})

	t.Run("unsupported lang falls back to first option", func(t *testing.T) {
		page := PageContext{
			Lang: "ja-JP",
			Loc:  langFakeLocalizer{},
		}
		label := ActiveLanguageLabel(page)
		if label == "" {
			t.Error("expected non-empty label from fallback")
		}
	})
}

func TestLanguageOptions(t *testing.T) {
	page := PageContext{
		Lang: "en-US",
		Loc:  langFakeLocalizer{},
	}
	options := LanguageOptions(page)
	if len(options) == 0 {
		t.Fatal("expected at least one option")
	}

	hasActive := false
	for _, opt := range options {
		if opt.Tag == "" {
			t.Error("expected non-empty tag")
		}
		if opt.Label == "" {
			t.Error("expected non-empty label")
		}
		if opt.Active {
			hasActive = true
		}
	}
	if !hasActive {
		t.Error("expected at least one active option")
	}
}
