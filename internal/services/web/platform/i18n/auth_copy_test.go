package i18n

import (
	"testing"

	"golang.org/x/text/language"
)

func TestAuthReturnsPortugueseCopyForPTBR(t *testing.T) {
	t.Parallel()

	copy := Auth(language.MustParse("pt-BR"))
	if copy.LoginHeading != "Faça login em Fracturing.Space" {
		t.Fatalf("LoginHeading = %q", copy.LoginHeading)
	}
	if copy.JSEmailRequired != "Email principal é obrigatório." {
		t.Fatalf("JSEmailRequired = %q", copy.JSEmailRequired)
	}
}

func TestAuthReturnsPortugueseCopyForPortugueseBaseLanguage(t *testing.T) {
	t.Parallel()

	copy := Auth(language.MustParse("pt-PT"))
	if copy.LandingTagline != "Motor de código aberto, autoritativo no servidor, para campanhas de RPG de mesa determinísticas e mestres de jogo com IA." {
		t.Fatalf("LandingTagline = %q", copy.LandingTagline)
	}
}

func TestAuthFallsBackToEnglishForNonPortugueseLanguage(t *testing.T) {
	t.Parallel()

	copy := Auth(language.MustParse("en-US"))
	if copy.LoginTitle != "Sign In | Fracturing.Space" {
		t.Fatalf("LoginTitle = %q", copy.LoginTitle)
	}
	if copy.JSRegisterFailed != "Passkey registration failed." {
		t.Fatalf("JSRegisterFailed = %q", copy.JSRegisterFailed)
	}
}
