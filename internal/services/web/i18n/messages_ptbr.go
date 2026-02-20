package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func init() {
	lang := language.MustParse("pt-BR")

	// Landing page
	message.SetString(lang, "title.landing", "%s | Motor de IA para RPG de código aberto")
	message.SetString(lang, "landing.tagline", "Motor de código aberto, autoritativo no servidor, para campanhas de RPG de mesa determinísticas e mestres de jogo com IA.")
	message.SetString(lang, "landing.signed_in_as", "Conectado como")
	message.SetString(lang, "landing.sign_out", "Sair")
	message.SetString(lang, "landing.sign_in", "Entrar")
	message.SetString(lang, "landing.docs", "Docs")
	message.SetString(lang, "landing.github", "GitHub")
	message.SetString(lang, "meta.description", "Motor de código aberto, autoritativo no servidor, para campanhas de RPG de mesa determinísticas e mestres de jogo com IA.")

	// Login page
	message.SetString(lang, "title.login", "%s | Entrar")
	message.SetString(lang, "login.heading", "Faça login para continuar")
	message.SetString(lang, "login.requesting_access", "%s (%s) está solicitando acesso à sua conta.")
	message.SetString(lang, "login.card_title", "Acesso à Conta")
	message.SetString(lang, "login.card_subtitle", "Crie uma conta ou entre com uma chave de acesso.")
	message.SetString(lang, "login.email", "Email principal")
	message.SetString(lang, "login.create_passkey", "Criar Conta Com Chave de Acesso")
	message.SetString(lang, "login.divider", "já tem conta?")
	message.SetString(lang, "login.sign_in_passkey", "Entrar Com Chave de Acesso")

	// Login JS strings (via data attributes)
	message.SetString(lang, "login.js.missing_session", "Sessão de login ausente.")
	message.SetString(lang, "login.js.passkey_failed", "Falha no login com chave de acesso.")
	message.SetString(lang, "login.js.email_required", "Email principal é obrigatório.")
	message.SetString(lang, "login.js.passkey_created", "Chave de acesso criada. Agora você pode entrar.")
	message.SetString(lang, "login.js.register_failed", "Falha no registro da chave de acesso.")
	message.SetString(lang, "login.js.login_start_error", "Não foi possível iniciar o login com chave de acesso.")
	message.SetString(lang, "login.js.login_finish_error", "Não foi possível concluir o login com chave de acesso.")
	message.SetString(lang, "login.js.register_start_error", "Ocorreu um erro ao criar sua conta. Se você já tem uma conta, use Entrar Com Chave de Acesso abaixo.")
	message.SetString(lang, "login.js.register_finish_error", "Não foi possível concluir o registro da chave de acesso.")

	// Magic page
	message.SetString(lang, "magic.unavailable.title", "Link mágico indisponível")
	message.SetString(lang, "magic.unavailable.message", "Não foi possível conectar ao serviço de autenticação.")
	message.SetString(lang, "magic.unavailable.detail", "Por favor, tente novamente em instantes.")
	message.SetString(lang, "magic.missing.title", "Link mágico ausente")
	message.SetString(lang, "magic.missing.message", "Este link está sem o token.")
	message.SetString(lang, "magic.missing.detail", "Por favor, solicite um novo link mágico e tente novamente.")
	message.SetString(lang, "magic.invalid.title", "Link mágico inválido")
	message.SetString(lang, "magic.invalid.message", "Não foi possível validar este link mágico.")
	message.SetString(lang, "magic.invalid.detail", "Ele pode ter expirado ou já ter sido usado.")
	message.SetString(lang, "magic.verified.title", "Link mágico verificado")
	message.SetString(lang, "magic.verified.message", "Seu link é válido e seu email foi confirmado.")
	message.SetString(lang, "magic.verified.detail", "Você pode voltar ao aplicativo e continuar a entrar.")
	message.SetString(lang, "magic.verified.link", "Voltar ao aplicativo")

	// Language nav
	message.SetString(lang, "nav.lang_en", "EN")
	message.SetString(lang, "nav.lang_pt_br", "PT-BR")
}
