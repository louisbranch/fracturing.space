package declarative

import (
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
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

func parseGmMode(value string) (gamev1.GmMode, error) {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return gamev1.GmMode_AI, nil
	}
	if enumValue, ok := gamev1.GmMode_value[candidate]; ok {
		return gamev1.GmMode(enumValue), nil
	}
	return gamev1.GmMode_GM_MODE_UNSPECIFIED, fmt.Errorf("unsupported gm_mode %q", value)
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

func parseDifficultyTier(value string) discoveryv1.DiscoveryDifficultyTier {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER
	}
	if enumValue, ok := discoveryv1.DiscoveryDifficultyTier_value[candidate]; ok {
		return discoveryv1.DiscoveryDifficultyTier(enumValue)
	}
	return discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER
}

func parseDiscoveryGmMode(value string) discoveryv1.DiscoveryGmMode {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_UNSPECIFIED
	}
	if !strings.HasPrefix(candidate, "DISCOVERY_GM_MODE_") {
		candidate = "DISCOVERY_GM_MODE_" + candidate
	}
	if enumValue, ok := discoveryv1.DiscoveryGmMode_value[candidate]; ok {
		return discoveryv1.DiscoveryGmMode(enumValue)
	}
	return discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_UNSPECIFIED
}

func parseDiscoveryIntent(value string) discoveryv1.DiscoveryIntent {
	candidate := normalizeEnumLabel(value)
	if candidate == "" {
		return discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_UNSPECIFIED
	}
	if !strings.HasPrefix(candidate, "DISCOVERY_INTENT_") {
		candidate = "DISCOVERY_INTENT_" + candidate
	}
	if enumValue, ok := discoveryv1.DiscoveryIntent_value[candidate]; ok {
		return discoveryv1.DiscoveryIntent(enumValue)
	}
	return discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_UNSPECIFIED
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
