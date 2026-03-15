package campaign

import (
	"encoding/json"
	"strings"
	"time"

	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

type updateFieldNormalizer func(currentStatus Status, value string) (string, *command.Rejection)

var campaignUpdateFieldNormalizers = map[string]updateFieldNormalizer{
	"status": func(currentStatus Status, value string) (string, *command.Rejection) {
		normalizedStatus, ok := normalizeStatusLabel(value)
		if !ok {
			return "", &command.Rejection{Code: rejectionCodeCampaignStatusInvalid, Message: "campaign status is invalid"}
		}
		if !isStatusTransitionAllowed(currentStatus, normalizedStatus) {
			return "", &command.Rejection{Code: rejectionCodeCampaignStatusTransition, Message: "campaign status transition is not allowed"}
		}
		return string(normalizedStatus), nil
	},
	"name": func(_ Status, value string) (string, *command.Rejection) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return "", &command.Rejection{Code: rejectionCodeCampaignNameEmpty, Message: "campaign name is required"}
		}
		return trimmed, nil
	},
	"theme_prompt": func(_ Status, value string) (string, *command.Rejection) {
		return strings.TrimSpace(value), nil
	},
	"locale": func(_ Status, value string) (string, *command.Rejection) {
		locale, ok := platformi18n.ParseLocale(value)
		if !ok {
			return "", &command.Rejection{Code: rejectionCodeCampaignLocaleInvalid, Message: "campaign locale is invalid"}
		}
		return platformi18n.LocaleString(locale), nil
	},
	"cover_asset_id": func(_ Status, value string) (string, *command.Rejection) {
		normalizedCoverAssetID, ok := normalizeCampaignCoverAssetID(value)
		if !ok {
			return "", &command.Rejection{Code: rejectionCodeCampaignCoverAssetInvalid, Message: "campaign cover asset is invalid"}
		}
		return normalizedCoverAssetID, nil
	},
	"cover_set_id": func(_ Status, value string) (string, *command.Rejection) {
		normalizedCoverSetID, ok := normalizeCampaignCoverSetID(value)
		if !ok {
			return "", &command.Rejection{Code: rejectionCodeCampaignCoverSetInvalid, Message: "campaign cover set is invalid"}
		}
		return normalizedCoverSetID, nil
	},
}

func decideUpdate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}
	var payload UpdatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: commandDecodeMessage(cmd, err),
		})
	}
	if len(payload.Fields) == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignUpdateEmpty,
			Message: "campaign update requires fields",
		})
	}

	normalizedFields := make(map[string]string, len(payload.Fields))
	currentStatus := state.Status
	if currentStatus == "" {
		currentStatus = StatusDraft
	}
	for key, value := range payload.Fields {
		normalizeField, ok := campaignUpdateFieldNormalizers[key]
		if !ok {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignUpdateFieldInvalid,
				Message: "campaign update field is invalid",
			})
		}
		normalized, rejection := normalizeField(currentStatus, value)
		if rejection != nil {
			return command.Reject(*rejection)
		}
		normalizedFields[key] = normalized
	}

	normalizedPayload := UpdatePayload{Fields: normalizedFields}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeUpdated, "campaign", string(cmd.CampaignID), payloadJSON, command.NowFunc(now)().UTC())
	return command.Accept(evt)
}
