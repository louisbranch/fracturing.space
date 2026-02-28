package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func init() {
	lang := language.MustParse("pt-BR")

	message.SetString(lang, ParticipantDefaultUnknownNameKey, "Pessoa Misteriosa")
	message.SetString(lang, ParticipantDefaultAINameKey, "Oráculo")
}
