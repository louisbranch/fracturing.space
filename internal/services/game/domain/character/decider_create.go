package character

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideCreate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCharacterAlreadyExists,
			Message: "character already exists",
		})
	}
	var payload CreatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	characterID := strings.TrimSpace(payload.CharacterID.String())
	if characterID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCharacterIDRequired,
			Message: "character id is required",
		})
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCharacterNameEmpty,
			Message: "character name is required",
		})
	}
	kind, ok := normalizeCharacterKindLabel(payload.Kind)
	if !ok {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCharacterKindInvalid,
			Message: "character kind is invalid",
		})
	}
	ownerParticipantID := strings.TrimSpace(payload.OwnerParticipantID.String())
	if ownerParticipantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCharacterOwnerParticipantID,
			Message: "owner participant id is required",
		})
	}
	notes := strings.TrimSpace(payload.Notes)
	pronouns := strings.TrimSpace(payload.Pronouns)
	aliases := normalizeAliases(payload.Aliases)
	avatarSetID, avatarAssetID, err := resolveCharacterAvatarSelection(
		characterID,
		payload.AvatarSetID,
		payload.AvatarAssetID,
	)
	if err != nil {
		return command.Reject(characterAvatarRejection(err))
	}

	normalizedPayload := CreatePayload{
		CharacterID:        ids.CharacterID(characterID),
		OwnerParticipantID: ids.ParticipantID(ownerParticipantID),
		Name:               name,
		Kind:               kind,
		Notes:              notes,
		AvatarSetID:        avatarSetID,
		AvatarAssetID:      avatarAssetID,
		Pronouns:           pronouns,
		Aliases:            aliases,
	}
	return acceptCharacterEvent(cmd, now, EventTypeCreated, characterID, normalizedPayload)
}
