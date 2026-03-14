package game

import "github.com/louisbranch/fracturing.space/internal/services/game/storage"

// communicationApplication coordinates communication-specific read assembly and
// control workflows behind a narrow dependency bundle.
type communicationApplication struct {
	auth         policyDependencies
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

func newCommunicationApplication(service *CommunicationService) communicationApplication {
	return communicationApplication{
		auth: newPolicyDependencies(service.stores),
		stores: communicationApplicationStores{
			Campaign:    service.stores.Campaign,
			Participant: service.stores.Participant,
			Character:   service.stores.Character,
			Scene:       service.stores.Scene,
		},
		sessions:     newSessionApplicationWithDependencies(service.stores, nil, service.idGenerator),
		gateCommands: newSessionGateCommandExecutor(service.stores.Write, service.stores.Applier()),
		idGenerator:  service.idGenerator,
	}
}
