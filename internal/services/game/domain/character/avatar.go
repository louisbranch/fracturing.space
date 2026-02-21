package character

import (
	"errors"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

var characterAvatarManifest = assetcatalog.AvatarManifest()

func resolveCharacterAvatarSelection(characterID, setID, assetID string) (string, string, error) {
	return characterAvatarManifest.ResolveSelection(assetcatalog.SelectionInput{
		EntityType: "character",
		EntityID:   characterID,
		SetID:      setID,
		AssetID:    assetID,
	})
}

func characterAvatarRejection(err error) command.Rejection {
	switch {
	case errors.Is(err, assetcatalog.ErrSetNotFound):
		return command.Rejection{
			Code:    rejectionCodeCharacterAvatarSetInvalid,
			Message: "character avatar set is invalid",
		}
	case errors.Is(err, assetcatalog.ErrAssetInvalid):
		return command.Rejection{
			Code:    rejectionCodeCharacterAvatarAssetInvalid,
			Message: "character avatar asset is invalid",
		}
	default:
		return command.Rejection{
			Code:    rejectionCodeCharacterAvatarAssetInvalid,
			Message: "character avatar is invalid",
		}
	}
}
