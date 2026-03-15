package campaign

import (
	"encoding/json"
	"strings"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

func decideCreate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignAlreadyExists,
			Message: "campaign already exists",
		})
	}
	var payload CreatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: commandDecodeMessage(cmd, err),
		})
	}
	normalizedPayload, rejection := normalizeCreatePayload(payload, string(cmd.CampaignID))
	if rejection != nil {
		return command.Reject(*rejection)
	}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeCreated, "campaign", string(cmd.CampaignID), payloadJSON, command.NowFunc(now)().UTC())
	return command.Accept(evt)
}

func normalizeCreatePayload(payload CreatePayload, campaignID string) (CreatePayload, *command.Rejection) {
	normalizedName := strings.TrimSpace(payload.Name)
	if normalizedName == "" {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignNameEmpty,
			Message: "campaign name is required",
		}
	}
	normalizedGameSystem, ok := normalizeGameSystemLabel(payload.GameSystem)
	if !ok {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignGameSystemInvalid,
			Message: "game system is required",
		}
	}
	normalizedGmMode, ok := normalizeGmModeLabel(payload.GmMode)
	if !ok {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignGmModeInvalid,
			Message: "gm mode is required",
		}
	}

	coverAssetID := strings.TrimSpace(payload.CoverAssetID)
	if coverAssetID == "" {
		coverAssetID = defaultCampaignCoverAssetID(campaignID)
	}
	normalizedCoverAssetID, ok := normalizeCampaignCoverAssetID(coverAssetID)
	if !ok {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignCoverAssetInvalid,
			Message: "campaign cover asset is invalid",
		}
	}
	coverSetID := strings.TrimSpace(payload.CoverSetID)
	if coverSetID == "" {
		coverSetID = defaultCampaignCoverSetID
	}
	normalizedCoverSetID, ok := normalizeCampaignCoverSetID(coverSetID)
	if !ok {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignCoverSetInvalid,
			Message: "campaign cover set is invalid",
		}
	}

	return CreatePayload{
		Name:         normalizedName,
		Locale:       normalizeCampaignLocale(payload.Locale),
		GameSystem:   normalizedGameSystem,
		GmMode:       normalizedGmMode,
		Intent:       strings.TrimSpace(payload.Intent),
		AccessPolicy: strings.TrimSpace(payload.AccessPolicy),
		ThemePrompt:  payload.ThemePrompt,
		CoverAssetID: normalizedCoverAssetID,
		CoverSetID:   normalizedCoverSetID,
	}, nil
}

// normalizeCampaignLocale canonicalizes known locale labels/tags and falls
// back to the platform default so create-event payloads remain replay-safe.
func normalizeCampaignLocale(value string) string {
	locale, ok := platformi18n.ParseLocale(value)
	if !ok {
		return platformi18n.LocaleString(platformi18n.DefaultLocale())
	}
	return platformi18n.LocaleString(locale)
}

// normalizeGameSystemLabel returns a canonical label for stable payload hashes.
//
// Campaign behavior depends on a stable system identity, not caller-specific casing
// or enum prefix variants.
func normalizeGameSystemLabel(value string) (string, bool) {
	system, ok := NormalizeGameSystem(value)
	if !ok || system == GameSystemUnspecified {
		return "", false
	}
	return system.String(), true
}

// normalizeGmModeLabel returns a canonical label for stable payload hashes.
//
// The canonical GM mode value is the shared contract used later by permission and
// command-policy checks.
func normalizeGmModeLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	switch strings.ToUpper(trimmed) {
	case "HUMAN", "GM_MODE_HUMAN":
		return "human", true
	case "AI", "GM_MODE_AI":
		return "ai", true
	case "HYBRID", "GM_MODE_HYBRID":
		return "hybrid", true
	default:
		return "", false
	}
}
