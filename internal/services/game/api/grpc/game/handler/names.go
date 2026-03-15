package handler

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	gamei18n "github.com/louisbranch/fracturing.space/internal/services/game/i18n"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"golang.org/x/text/message"
)

// DefaultUnknownParticipantName returns the localized default name for an
// unknown participant.
func DefaultUnknownParticipantName(locale commonv1.Locale) string {
	return LocalizeByLocale(locale, gamei18n.ParticipantDefaultUnknownNameKey, gamei18n.ParticipantDefaultUnknownNameFallback)
}

// DefaultAIParticipantName returns the localized default name for an AI
// participant.
func DefaultAIParticipantName(locale commonv1.Locale) string {
	return LocalizeByLocale(locale, gamei18n.ParticipantDefaultAINameKey, gamei18n.ParticipantDefaultAINameFallback)
}

// DefaultUnknownParticipantPronouns returns the default pronouns for an
// unknown participant.
func DefaultUnknownParticipantPronouns() string {
	return sharedpronouns.PronounTheyThem
}

// DefaultAIParticipantPronouns returns the default pronouns for an AI
// participant.
func DefaultAIParticipantPronouns() string {
	return sharedpronouns.PronounItIts
}

// LocalizeByLocale returns a localized value for the given key and locale,
// falling back to the provided default when the key is not found.
func LocalizeByLocale(locale commonv1.Locale, key, fallback string) string {
	value := message.NewPrinter(platformi18n.TagForLocale(locale)).Sprintf(key)
	value = strings.TrimSpace(value)
	if value == "" || value == key {
		return fallback
	}
	return value
}
