package game

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	gamei18n "github.com/louisbranch/fracturing.space/internal/services/game/i18n"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"golang.org/x/text/message"
)

func defaultUnknownParticipantName(locale commonv1.Locale) string {
	return localizeByLocale(locale, gamei18n.ParticipantDefaultUnknownNameKey, gamei18n.ParticipantDefaultUnknownNameFallback)
}

func defaultAIParticipantName(locale commonv1.Locale) string {
	return localizeByLocale(locale, gamei18n.ParticipantDefaultAINameKey, gamei18n.ParticipantDefaultAINameFallback)
}

func defaultUnknownParticipantPronouns() string {
	return sharedpronouns.PronounTheyThem
}

func defaultAIParticipantPronouns() string {
	return sharedpronouns.PronounItIts
}

func localizeByLocale(locale commonv1.Locale, key, fallback string) string {
	value := message.NewPrinter(platformi18n.TagForLocale(locale)).Sprintf(key)
	value = strings.TrimSpace(value)
	if value == "" || value == key {
		return fallback
	}
	return value
}
