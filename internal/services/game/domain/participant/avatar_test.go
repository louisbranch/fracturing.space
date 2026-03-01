package participant

import (
	"errors"
	"testing"

	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
)

func TestParticipantAvatarRejection(t *testing.T) {
	t.Run("set not found", func(t *testing.T) {
		rejection := participantAvatarRejection(assetcatalog.ErrSetNotFound)
		if rejection.Code != rejectionCodeParticipantAvatarSetInvalid {
			t.Fatalf("code = %q, want %q", rejection.Code, rejectionCodeParticipantAvatarSetInvalid)
		}
	})

	t.Run("asset invalid", func(t *testing.T) {
		rejection := participantAvatarRejection(assetcatalog.ErrAssetInvalid)
		if rejection.Code != rejectionCodeParticipantAvatarAssetInvalid {
			t.Fatalf("code = %q, want %q", rejection.Code, rejectionCodeParticipantAvatarAssetInvalid)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		rejection := participantAvatarRejection(errors.New("unexpected"))
		if rejection.Code != rejectionCodeParticipantAvatarAssetInvalid {
			t.Fatalf("code = %q, want %q", rejection.Code, rejectionCodeParticipantAvatarAssetInvalid)
		}
		if rejection.Message != "participant avatar is invalid" {
			t.Fatalf("message = %q, want %q", rejection.Message, "participant avatar is invalid")
		}
	})
}
