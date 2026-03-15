package game

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// characterApplication coordinates character transport use-cases across focused
// method files (create, update, delete, control, workflow, and profile patching).
type characterApplication struct {
	auth        policyDependencies
	stores      characterApplicationStores
	write       domainwriteexec.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
}

type characterApplicationStores struct {
	Campaign           storage.CampaignStore
	Character          storage.CharacterStore
	Participant        storage.ParticipantStore
	Daggerheart        projectionstore.Store
	DaggerheartContent contentstore.DaggerheartContentReadStore
}

func newCharacterApplicationWithDependencies(
	stores Stores,
	clock func() time.Time,
	idGenerator func() (string, error),
) characterApplication {
	app := characterApplication{
		auth: newPolicyDependencies(stores),
		stores: characterApplicationStores{
			Campaign:           stores.Campaign,
			Character:          stores.Character,
			Participant:        stores.Participant,
			Daggerheart:        stores.SystemStores.Daggerheart,
			DaggerheartContent: stores.DaggerheartContent,
		},
		write:       stores.Write,
		applier:     stores.Applier(),
		clock:       clock,
		idGenerator: idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
