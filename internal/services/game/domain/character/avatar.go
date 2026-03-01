package character

import (
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/internal/avatarreject"
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
	return avatarreject.FromSelectionError(
		err,
		command.Rejection{
			Code:    rejectionCodeCharacterAvatarSetInvalid,
			Message: "character avatar set is invalid",
		},
		command.Rejection{
			Code:    rejectionCodeCharacterAvatarAssetInvalid,
			Message: "character avatar asset is invalid",
		},
		command.Rejection{
			Code:    rejectionCodeCharacterAvatarAssetInvalid,
			Message: "character avatar is invalid",
		},
	)
}
