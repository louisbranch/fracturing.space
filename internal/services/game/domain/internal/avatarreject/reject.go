package avatarreject

import (
	"errors"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

// FromSelectionError maps asset catalog selection failures to domain rejections.
func FromSelectionError(
	err error,
	setRejection command.Rejection,
	assetRejection command.Rejection,
	fallbackRejection command.Rejection,
) command.Rejection {
	switch {
	case errors.Is(err, assetcatalog.ErrSetNotFound):
		return setRejection
	case errors.Is(err, assetcatalog.ErrAssetInvalid):
		return assetRejection
	default:
		return fallbackRejection
	}
}
