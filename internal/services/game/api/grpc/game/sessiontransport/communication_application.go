package sessiontransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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

func newCommunicationApplicationFromDeps(
	deps Deps,
	idGenerator func() (string, error),
) communicationApplication {
	auth := deps.Auth
	if auth.Participant == nil {
		auth = authz.PolicyDeps{Participant: deps.Participant, Character: deps.Character, Audit: auth.Audit}
	}
	return communicationApplication{
		auth: auth,
		stores: communicationApplicationStores{
			Campaign:    deps.Campaign,
			Participant: deps.Participant,
			Character:   deps.Character,
			Scene:       deps.Scene,
		},
		sessions:     newSessionApplicationFromDeps(deps, nil, idGenerator),
		gateCommands: newSessionGateCommandExecutor(deps.Write, deps.Applier),
		idGenerator:  idGenerator,
	}
}
