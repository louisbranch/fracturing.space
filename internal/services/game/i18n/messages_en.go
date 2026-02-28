package i18n

import (
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func init() {
	lang := language.English

	message.SetString(lang, ParticipantDefaultUnknownNameKey, "Mysterious Person")
	message.SetString(lang, ParticipantDefaultAINameKey, "Oracle")
}
