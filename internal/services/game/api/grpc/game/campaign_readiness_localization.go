package game

import (
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"golang.org/x/text/message"
)

func resolveReadinessLocale(requested commonv1.Locale, campaignLocale commonv1.Locale) commonv1.Locale {
	if requested != commonv1.Locale_LOCALE_UNSPECIFIED {
		return platformi18n.NormalizeLocale(requested)
	}
	if campaignLocale != commonv1.Locale_LOCALE_UNSPECIFIED {
		return platformi18n.NormalizeLocale(campaignLocale)
	}
	return commonv1.Locale_LOCALE_EN_US
}

func readinessBlockerToProto(locale commonv1.Locale, blocker readiness.Blocker) *campaignv1.CampaignSessionReadinessBlocker {
	metadata := make(map[string]string, len(blocker.Metadata))
	for key, value := range blocker.Metadata {
		metadata[key] = value
	}
	return &campaignv1.CampaignSessionReadinessBlocker{
		Code:     strings.TrimSpace(blocker.Code),
		Message:  localizeReadinessBlockerMessage(locale, blocker),
		Metadata: metadata,
	}
}

func localizeReadinessBlockerMessage(locale commonv1.Locale, blocker readiness.Blocker) string {
	printer := message.NewPrinter(platformi18n.TagForLocale(locale))
	switch blocker.Code {
	case readiness.RejectionCodeSessionReadinessCampaignStatusDisallowsStart:
		return printer.Sprintf("game.session_readiness.campaign_status_disallows_start", readinessBlockerMetadataValue(blocker.Metadata, "status"))
	case readiness.RejectionCodeSessionReadinessActiveSessionExists:
		return printer.Sprintf("game.session_readiness.active_session_exists")
	case readiness.RejectionCodeSessionReadinessAIAgentRequired:
		return printer.Sprintf("game.session_readiness.ai_agent_required")
	case readiness.RejectionCodeSessionReadinessAIGMParticipantRequired:
		return printer.Sprintf("game.session_readiness.ai_gm_participant_required")
	case readiness.RejectionCodeSessionReadinessGMRequired:
		return printer.Sprintf("game.session_readiness.gm_required")
	case readiness.RejectionCodeSessionReadinessPlayerRequired:
		return printer.Sprintf("game.session_readiness.player_required")
	case readiness.RejectionCodeSessionReadinessCharacterControllerRequired:
		return printer.Sprintf("game.session_readiness.character_controller_required", readinessBlockerMetadataValue(blocker.Metadata, "character_id"))
	case readiness.RejectionCodeSessionReadinessPlayerCharacterRequired:
		return printer.Sprintf("game.session_readiness.player_character_required", readinessPlayerParticipantLabel(blocker.Metadata))
	case readiness.RejectionCodeSessionReadinessCharacterSystemRequired:
		reason := readinessBlockerOptionalMetadataValue(blocker.Metadata, "reason")
		if reason == "" {
			return printer.Sprintf("game.session_readiness.character_system_required", readinessBlockerMetadataValue(blocker.Metadata, "character_id"))
		}
		return printer.Sprintf("game.session_readiness.character_system_required_with_reason", readinessBlockerMetadataValue(blocker.Metadata, "character_id"), reason)
	default:
		return strings.TrimSpace(blocker.Message)
	}
}

func readinessBlockerOptionalMetadataValue(metadata map[string]string, key string) string {
	return readinessBlockerMetadataValueOrDefault(metadata, key, "")
}

func readinessPlayerParticipantLabel(metadata map[string]string) string {
	name := readinessBlockerOptionalMetadataValue(metadata, "participant_name")
	if name != "" {
		return name
	}
	return readinessBlockerMetadataValue(metadata, "participant_id")
}

func readinessBlockerMetadataValue(metadata map[string]string, key string) string {
	return readinessBlockerMetadataValueOrDefault(metadata, key, "unspecified")
}

func readinessBlockerMetadataValueOrDefault(metadata map[string]string, key, fallback string) string {
	value := strings.TrimSpace(metadata[key])
	if value != "" {
		return value
	}
	return fallback
}
