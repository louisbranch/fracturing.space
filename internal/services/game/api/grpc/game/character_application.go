package game

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
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
	Daggerheart        storage.DaggerheartStore
	DaggerheartContent storage.DaggerheartContentReadStore
}

func newCharacterApplication(service *CharacterService) characterApplication {
	app := characterApplication{
		auth: newPolicyDependencies(service.stores),
		stores: characterApplicationStores{
			Campaign:           service.stores.Campaign,
			Character:          service.stores.Character,
			Participant:        service.stores.Participant,
			Daggerheart:        service.stores.SystemStores.Daggerheart,
			DaggerheartContent: service.stores.DaggerheartContent,
		},
		write:       service.stores.Write,
		applier:     service.stores.Applier(),
		clock:       service.clock,
		idGenerator: service.idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
