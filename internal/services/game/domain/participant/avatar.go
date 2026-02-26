package participant

import (
	"errors"
	"strings"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

var participantAvatarManifest = assetcatalog.AvatarManifest()

func resolveParticipantAvatarSelection(participantID, userID, setID, assetID string) (string, string, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return participantAvatarManifest.ResolveSelection(assetcatalog.SelectionInput{
			EntityType: assetcatalog.AvatarRoleParticipant,
			EntityID:   strings.TrimSpace(participantID),
			SetID:      assetcatalog.AvatarSetBlankV1,
			AssetID:    "",
		})
	}
	return participantAvatarManifest.ResolveSelection(assetcatalog.SelectionInput{
		EntityType: assetcatalog.AvatarRoleUser,
		EntityID:   trimmedUserID,
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
