package account

import (
	"errors"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
)

var (
	ErrAvatarSetInvalid   = errors.New("avatar set is invalid")
	ErrAvatarAssetInvalid = errors.New("avatar asset is invalid")
)

var profileAvatarManifest = assetcatalog.AvatarManifest()

func resolveProfileAvatarSelection(userID, setID, assetID string) (string, string, error) {
	resolvedSetID, resolvedAssetID, err := profileAvatarManifest.ResolveSelection(assetcatalog.SelectionInput{
		EntityType: "user",
		EntityID:   userID,
		SetID:      setID,
		AssetID:    assetID,
	})
	if err == nil {
		return resolvedSetID, resolvedAssetID, nil
	}
	switch {
	case errors.Is(err, assetcatalog.ErrSetNotFound):
		return "", "", ErrAvatarSetInvalid
	case errors.Is(err, assetcatalog.ErrAssetInvalid):
		return "", "", ErrAvatarAssetInvalid
	default:
		return "", "", err
	}
}
