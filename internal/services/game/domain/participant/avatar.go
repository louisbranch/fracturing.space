package participant

import (
	"errors"
	"strings"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

var participantAvatarManifest = assetcatalog.AvatarManifest()

func resolveParticipantAvatarSelection(participantID, userID, setID, assetID string) (string, string, error) {
	entityType := "participant"
	entityID := participantID
	if trimmedUserID := strings.TrimSpace(userID); trimmedUserID != "" {
		entityType = "user"
		entityID = trimmedUserID
	}
	return participantAvatarManifest.ResolveSelection(assetcatalog.SelectionInput{
		EntityType: entityType,
		EntityID:   entityID,
		SetID:      setID,
		AssetID:    assetID,
	})
}

func participantAvatarRejection(err error) command.Rejection {
	switch {
	case errors.Is(err, assetcatalog.ErrSetNotFound):
		return command.Rejection{
			Code:    rejectionCodeParticipantAvatarSetInvalid,
			Message: "participant avatar set is invalid",
		}
	case errors.Is(err, assetcatalog.ErrAssetInvalid):
		return command.Rejection{
			Code:    rejectionCodeParticipantAvatarAssetInvalid,
			Message: "participant avatar asset is invalid",
		}
	default:
		return command.Rejection{
			Code:    rejectionCodeParticipantAvatarAssetInvalid,
			Message: "participant avatar is invalid",
		}
	}
}
