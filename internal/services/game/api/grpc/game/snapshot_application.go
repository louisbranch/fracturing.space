package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// snapshotApplication coordinates snapshot transport use-cases across focused
// state patch/update helper files using Daggerheart-specific reads and explicit
// write execution seams.
type snapshotApplication struct {
	auth    policyDependencies
	stores  snapshotApplicationStores
	write   domainwriteexec.WritePath
	applier projection.Applier
}

type snapshotApplicationStores struct {
	Campaign    storage.CampaignStore
	Character   storage.CharacterStore
	Daggerheart projectionstore.Store
}

func newSnapshotApplication(service *SnapshotService) snapshotApplication {
	if service == nil {
		return snapshotApplication{}
	}
	return service.app
}

func newSnapshotApplicationWithDependencies(stores Stores) snapshotApplication {
	return snapshotApplication{
		auth: newPolicyDependencies(stores),
		stores: snapshotApplicationStores{
			Campaign:    stores.Campaign,
			Character:   stores.Character,
			Daggerheart: stores.SystemStores.Daggerheart,
		},
		write:   stores.Write,
		applier: stores.Applier(),
	}
}
