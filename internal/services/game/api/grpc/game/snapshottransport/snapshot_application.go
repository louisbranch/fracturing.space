package snapshottransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Deps holds all dependencies needed by the snapshot transport layer.
type Deps struct {
	Auth        authz.PolicyDeps
	Campaign    storage.CampaignStore
	Character   storage.CharacterStore
	Daggerheart projectionstore.Store
	Write       domainwriteexec.WritePath
	Applier     projection.Applier
}

// snapshotApplication coordinates snapshot transport use-cases across focused
// state patch/update helper files using Daggerheart-specific reads and explicit
// write execution seams.
type snapshotApplication struct {
	auth    authz.PolicyDeps
	stores  snapshotApplicationStores
	write   domainwriteexec.WritePath
	applier projection.Applier
}

type snapshotApplicationStores struct {
	Campaign    storage.CampaignStore
	Character   storage.CharacterStore
	Daggerheart projectionstore.Store
}

func newSnapshotApplication(service *Service) snapshotApplication {
	if service == nil {
		return snapshotApplication{}
	}
	return service.app
}

func newSnapshotApplicationWithDependencies(deps Deps) snapshotApplication {
	return snapshotApplication{
		auth: deps.Auth,
		stores: snapshotApplicationStores{
			Campaign:    deps.Campaign,
			Character:   deps.Character,
			Daggerheart: deps.Daggerheart,
		},
		write:   deps.Write,
		applier: deps.Applier,
	}
}
