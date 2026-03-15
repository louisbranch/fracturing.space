package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
)

// communicationApplication coordinates communication-specific read assembly and
// control workflows behind a narrow dependency bundle.
type communicationApplication struct {
	auth         authz.PolicyDeps
	stores       communicationApplicationStores
	sessions     sessionApplication
	gateCommands sessionGateCommandExecutor
	idGenerator  func() (string, error)
}

type communicationApplicationStores struct {
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Scene       storage.SceneStore
}

func newCommunicationApplicationWithDependencies(
	stores Stores,
	idGenerator func() (string, error),
) communicationApplication {
	return communicationApplication{
		auth: authz.PolicyDeps{Participant: stores.Participant, Character: stores.Character, Audit: stores.Audit},
		stores: communicationApplicationStores{
			Campaign:    stores.Campaign,
			Participant: stores.Participant,
			Character:   stores.Character,
			Scene:       stores.Scene,
		},
		sessions:     newSessionApplicationWithDependencies(stores, nil, idGenerator),
		gateCommands: newSessionGateCommandExecutor(stores.Write, stores.Applier()),
		idGenerator:  idGenerator,
	}
}
