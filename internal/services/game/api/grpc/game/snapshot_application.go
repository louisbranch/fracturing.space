package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// snapshotApplication coordinates snapshot transport use-cases across focused
// state patch/update helper files using Daggerheart-specific reads and explicit
// write execution seams.
type snapshotApplication struct {
	auth    Stores
	stores  snapshotApplicationStores
	write   domainwriteexec.WritePath
	applier projection.Applier
}

type snapshotApplicationStores struct {
	Campaign    storage.CampaignStore
	Character   storage.CharacterStore
	Daggerheart storage.DaggerheartStore
}

func newSnapshotApplication(service *SnapshotService) snapshotApplication {
	return snapshotApplication{
		auth: service.stores,
		stores: snapshotApplicationStores{
			Campaign:    service.stores.Campaign,
			Character:   service.stores.Character,
			Daggerheart: service.stores.SystemStores.Daggerheart,
		},
		write:   service.stores.Write,
		applier: service.stores.Applier(),
	}
}
