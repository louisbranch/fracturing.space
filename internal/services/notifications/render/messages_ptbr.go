package render

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func init() {
	lang := language.MustParse("pt-BR")

	message.SetString(lang, "notification.generic.title", "Notificação")
	message.SetString(lang, "notification.generic.body", "Você tem uma nova notificação.")
	message.SetString(lang, "notification.generic.email_subject", "Notificação do Fracturing Space")
	message.SetString(lang, "notification.signup_method.passkey", "chave de acesso")
	message.SetString(lang, "notification.signup_method.magic_link", "link mágico")
	message.SetString(lang, "notification.signup_method.unknown", "outro método")
	message.SetString(lang, "notification.onboarding_welcome.title", "Boas-vindas ao Fracturing Space")
	message.SetString(lang, "notification.onboarding_welcome.body", "Sua conta está pronta. Método de entrada: %s.")
	message.SetString(lang, "notification.onboarding_welcome.email_subject", "Boas-vindas ao Fracturing Space")
	message.SetString(lang, "notification.onboarding_welcome.email_body", "Sua conta está pronta. Método de entrada: %s.")
}
