package avatarreject

import (
	"errors"
	"testing"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

func TestFromSelectionError(t *testing.T) {
	setRejection := command.Rejection{Code: "SET_INVALID", Message: "set invalid"}
	assetRejection := command.Rejection{Code: "ASSET_INVALID", Message: "asset invalid"}
	fallbackRejection := command.Rejection{Code: "AVATAR_INVALID", Message: "avatar invalid"}

	t.Run("set not found", func(t *testing.T) {
		got := FromSelectionError(assetcatalog.ErrSetNotFound, setRejection, assetRejection, fallbackRejection)
		if got != setRejection {
			t.Fatalf("rejection = %+v, want %+v", got, setRejection)
		}
	})

	t.Run("asset invalid", func(t *testing.T) {
		got := FromSelectionError(assetcatalog.ErrAssetInvalid, setRejection, assetRejection, fallbackRejection)
		if got != assetRejection {
			t.Fatalf("rejection = %+v, want %+v", got, assetRejection)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		got := FromSelectionError(errors.New("boom"), setRejection, assetRejection, fallbackRejection)
		if got != fallbackRejection {
			t.Fatalf("rejection = %+v, want %+v", got, fallbackRejection)
		}
	})
}
