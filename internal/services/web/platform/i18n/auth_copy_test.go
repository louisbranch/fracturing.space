package i18n

import (
	"testing"

	"golang.org/x/text/language"
)

func TestAuthReturnsPortugueseCopyForPTBR(t *testing.T) {
	t.Parallel()

	copy := Auth(language.MustParse("pt-BR"))
	if copy.LoginHeading != authCopyPTBR.LoginHeading {
		t.Fatalf("LoginHeading = %q, want %q", copy.LoginHeading, authCopyPTBR.LoginHeading)
	}
	if copy.JSEmailRequired != authCopyPTBR.JSEmailRequired {
		t.Fatalf("JSEmailRequired = %q, want %q", copy.JSEmailRequired, authCopyPTBR.JSEmailRequired)
	}
}

func TestAuthReturnsPortugueseCopyForPortugueseBaseLanguage(t *testing.T) {
	t.Parallel()

	copy := Auth(language.MustParse("pt-PT"))
	if copy.LandingTagline != authCopyPTBR.LandingTagline {
		t.Fatalf("LandingTagline = %q, want %q", copy.LandingTagline, authCopyPTBR.LandingTagline)
	}
}

func TestAuthFallsBackToEnglishForNonPortugueseLanguage(t *testing.T) {
	t.Parallel()

	copy := Auth(language.MustParse("en-US"))
	if copy.LoginTitle != authCopyEN.LoginTitle {
		t.Fatalf("LoginTitle = %q, want %q", copy.LoginTitle, authCopyEN.LoginTitle)
	}
	if copy.JSRegisterFailed != authCopyEN.JSRegisterFailed {
		t.Fatalf("JSRegisterFailed = %q, want %q", copy.JSRegisterFailed, authCopyEN.JSRegisterFailed)
	}
}
