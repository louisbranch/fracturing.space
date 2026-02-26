package i18n

import "golang.org/x/text/language"

// AuthCopy holds translatable copy for web2 public auth pages.
type AuthCopy struct {
	MetaDescription     string
	LandingTitle        string
	LandingTagline      string
	LandingSignIn       string
	LandingDocs         string
	LandingGitHub       string
	LoginTitle          string
	LoginHeading        string
	LoginCardTitle      string
	LoginCardSubtitle   string
	LoginEmail          string
	LoginCreatePasskey  string
	LoginDivider        string
	LoginSignInPasskey  string
	JSLoginStartError   string
	JSLoginFinishError  string
	JSRegisterStartErr  string
	JSRegisterFinishErr string
	JSPasskeyFailed     string
	JSEmailRequired     string
	JSPasskeyCreated    string
	JSRegisterFailed    string
}

var authCopyEN = AuthCopy{
	MetaDescription:     "Open-source, server-authoritative engine for deterministic tabletop RPG campaigns and AI game masters.",
	LandingTitle:        "Open source AI GM engine | Fracturing.Space",
	LandingTagline:      "Open-source, server-authoritative engine for deterministic tabletop RPG campaigns and AI game masters.",
	LandingSignIn:       "Sign in",
	LandingDocs:         "Docs",
	LandingGitHub:       "GitHub",
	LoginTitle:          "Sign In | Fracturing.Space",
	LoginHeading:        "Sign in to Fracturing.Space",
	LoginCardTitle:      "Account Access",
	LoginCardSubtitle:   "Create an account or sign in with a passkey.",
	LoginEmail:          "Email",
	LoginCreatePasskey:  "Create Account With Passkey",
	LoginDivider:        "returning?",
	LoginSignInPasskey:  "Sign In With Passkey",
	JSLoginStartError:   "failed to start passkey login",
	JSLoginFinishError:  "failed to finish passkey login",
	JSRegisterStartErr:  "failed to start passkey registration",
	JSRegisterFinishErr: "failed to finish passkey registration",
	JSPasskeyFailed:     "failed to sign in with passkey",
	JSEmailRequired:     "email is required",
	JSPasskeyCreated:    "passkey created; signing you in",
	JSRegisterFailed:    "failed to create passkey",
}

var authCopyPTBR = AuthCopy{
	MetaDescription:     "Motor de código aberto, autoritativo no servidor, para campanhas de RPG de mesa determinísticas e mestres de jogo com IA.",
	LandingTitle:        "Motor de IA para RPG de código aberto | Fracturing.Space",
	LandingTagline:      "Motor de código aberto, autoritativo no servidor, para campanhas de RPG de mesa determinísticas e mestres de jogo com IA.",
	LandingSignIn:       "Entrar",
	LandingDocs:         "Docs",
	LandingGitHub:       "GitHub",
	LoginTitle:          "Entrar | Fracturing.Space",
	LoginHeading:        "Faça login em Fracturing.Space",
	LoginCardTitle:      "Acesso à Conta",
	LoginCardSubtitle:   "Crie uma conta ou entre com uma chave de acesso.",
	LoginEmail:          "Email principal",
	LoginCreatePasskey:  "Criar Conta Com Chave de Acesso",
	LoginDivider:        "já tem conta?",
	LoginSignInPasskey:  "Entrar Com Chave de Acesso",
	JSLoginStartError:   "Não foi possível iniciar o login com chave de acesso.",
	JSLoginFinishError:  "Não foi possível concluir o login com chave de acesso.",
	JSRegisterStartErr:  "Não foi possível iniciar o registro da chave de acesso.",
	JSRegisterFinishErr: "Não foi possível concluir o registro da chave de acesso.",
	JSPasskeyFailed:     "Falha no login com chave de acesso.",
	JSEmailRequired:     "Email principal é obrigatório.",
	JSPasskeyCreated:    "Chave de acesso criada; fazendo login.",
	JSRegisterFailed:    "Falha no registro da chave de acesso.",
}

// Auth returns localized auth copy for the provided language tag.
func Auth(tag language.Tag) AuthCopy {
	if tag == language.MustParse("pt-BR") {
		return authCopyPTBR
	}
	base, _ := tag.Base()
	portugueseBase, _ := language.Portuguese.Base()
	if base == portugueseBase {
		return authCopyPTBR
	}
	return authCopyEN
}
