package declarative

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func parseLocale(value string) commonv1.Locale {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return commonv1.Locale_LOCALE_EN_US
	}
	if !strings.HasPrefix(candidate, "LOCALE_") {
		candidate = "LOCALE_" + candidate
	}
	if enumValue, ok := commonv1.Locale_value[candidate]; ok {
		return commonv1.Locale(enumValue)
	}
	return commonv1.Locale_LOCALE_EN_US
}

func parseGameSystem(value string) commonv1.GameSystem {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	}
	if !strings.HasPrefix(candidate, "GAME_SYSTEM_") {
		candidate = "GAME_SYSTEM_" + candidate
	}
	if enumValue, ok := commonv1.GameSystem_value[candidate]; ok {
		return commonv1.GameSystem(enumValue)
	}
	return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
}

func parseGmMode(value string) gamev1.GmMode {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return gamev1.GmMode_HUMAN
	}
	if enumValue, ok := gamev1.GmMode_value[candidate]; ok {
		return gamev1.GmMode(enumValue)
	}
	return gamev1.GmMode_HUMAN
}

func parseCampaignIntent(value string) gamev1.CampaignIntent {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return gamev1.CampaignIntent_STANDARD
	}
	if enumValue, ok := gamev1.CampaignIntent_value[candidate]; ok {
		return gamev1.CampaignIntent(enumValue)
	}
	return gamev1.CampaignIntent_STANDARD
}

func parseAccessPolicy(value string) gamev1.CampaignAccessPolicy {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return gamev1.CampaignAccessPolicy_PRIVATE
	}
	if enumValue, ok := gamev1.CampaignAccessPolicy_value[candidate]; ok {
		return gamev1.CampaignAccessPolicy(enumValue)
	}
	return gamev1.CampaignAccessPolicy_PRIVATE
}

func parseParticipantRole(value string) gamev1.ParticipantRole {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return gamev1.ParticipantRole_PLAYER
	}
	if enumValue, ok := gamev1.ParticipantRole_value[candidate]; ok {
		return gamev1.ParticipantRole(enumValue)
	}
	return gamev1.ParticipantRole_PLAYER
}

func parseParticipantController(value string) gamev1.Controller {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return gamev1.Controller_CONTROLLER_HUMAN
	}
	if enumValue, ok := gamev1.Controller_value[candidate]; ok {
		return gamev1.Controller(enumValue)
	}
	return gamev1.Controller_CONTROLLER_HUMAN
}

func parseCharacterKind(value string) gamev1.CharacterKind {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return gamev1.CharacterKind_PC
	}
	if enumValue, ok := gamev1.CharacterKind_value[candidate]; ok {
		return gamev1.CharacterKind(enumValue)
	}
	return gamev1.CharacterKind_PC
}

func parseSessionStatus(value string) gamev1.SessionStatus {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return gamev1.SessionStatus_SESSION_ACTIVE
	}
	if enumValue, ok := gamev1.SessionStatus_value[candidate]; ok {
		return gamev1.SessionStatus(enumValue)
	}
	return gamev1.SessionStatus_SESSION_ACTIVE
}

func parseDifficultyTier(value string) listingv1.CampaignDifficultyTier {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER
	}
	if enumValue, ok := listingv1.CampaignDifficultyTier_value[candidate]; ok {
		return listingv1.CampaignDifficultyTier(enumValue)
	}
	return listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER
}

func normalizeEnumLabel(value string) string {
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return ""
	}
	candidate = strings.ReplaceAll(candidate, "-", "_")
	candidate = strings.ReplaceAll(candidate, " ", "_")
	candidate = strings.ToUpper(candidate)
	return candidate
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func stringValue(value string) *wrapperspb.StringValue {
	return wrapperspb.String(strings.TrimSpace(value))
}
