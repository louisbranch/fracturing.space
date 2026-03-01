package character

import (
	"errors"
	"testing"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
)

func TestCharacterAvatarRejection(t *testing.T) {
	t.Run("set not found", func(t *testing.T) {
		rejection := characterAvatarRejection(assetcatalog.ErrSetNotFound)
		if rejection.Code != rejectionCodeCharacterAvatarSetInvalid {
			t.Fatalf("code = %q, want %q", rejection.Code, rejectionCodeCharacterAvatarSetInvalid)
		}
	})

	t.Run("asset invalid", func(t *testing.T) {
		rejection := characterAvatarRejection(assetcatalog.ErrAssetInvalid)
		if rejection.Code != rejectionCodeCharacterAvatarAssetInvalid {
			t.Fatalf("code = %q, want %q", rejection.Code, rejectionCodeCharacterAvatarAssetInvalid)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		rejection := characterAvatarRejection(errors.New("unexpected"))
		if rejection.Code != rejectionCodeCharacterAvatarAssetInvalid {
			t.Fatalf("code = %q, want %q", rejection.Code, rejectionCodeCharacterAvatarAssetInvalid)
		}
		if rejection.Message != "character avatar is invalid" {
			t.Fatalf("message = %q, want %q", rejection.Message, "character avatar is invalid")
		}
	})
}
