package testkit

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type fakeDaggerheartStore struct {
	storage.DaggerheartStore
}

func TestValidateSystemConformance_Daggerheart(t *testing.T) {
	mod := daggerheart.NewModule()
	adapter := daggerheart.NewAdapter(fakeDaggerheartStore{})
	ValidateSystemConformance(t, mod, adapter)
}
