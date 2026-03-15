package charactertransport

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Deps holds the explicit dependencies for the character transport subpackage.
type Deps struct {
	Auth               authz.PolicyDeps
	Campaign           storage.CampaignStore
	Character          storage.CharacterStore
	Participant        storage.ParticipantStore
	Daggerheart        projectionstore.Store
	DaggerheartContent contentstore.DaggerheartContentReadStore
	Write              domainwriteexec.WritePath
	Applier            projection.Applier
}

// characterApplication coordinates character transport use-cases across focused
// method files (create, update, delete, control, workflow, and profile patching).
type characterApplication struct {
	auth        authz.PolicyDeps
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

func newCharacterApplicationFromDeps(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) characterApplication {
	app := characterApplication{
		auth: deps.Auth,
		stores: characterApplicationStores{
			Campaign:           deps.Campaign,
			Character:          deps.Character,
			Participant:        deps.Participant,
			Daggerheart:        deps.Daggerheart,
			DaggerheartContent: deps.DaggerheartContent,
		},
		write:       deps.Write,
		applier:     deps.Applier,
		clock:       clock,
		idGenerator: idGenerator,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}
