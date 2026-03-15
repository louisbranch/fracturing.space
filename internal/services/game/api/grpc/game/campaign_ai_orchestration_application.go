package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/interactiontransport"
)

func newCampaignAIOrchestrationApplicationWithDependencies(
	stores Stores,
	idGenerator func() (string, error),
) interactiontransport.AIOrchestrationApplication {
	return interactiontransport.NewAIOrchestrationApplication(interactiontransport.Deps{
		Auth: authz.PolicyDeps{
			Participant: stores.Participant,
			Character:   stores.Character,
			Audit:       stores.Audit,
		},
		Campaign:           stores.Campaign,
		Participant:        stores.Participant,
		Character:          stores.Character,
		Session:            stores.Session,
		SessionInteraction: stores.SessionInteraction,
		Scene:              stores.Scene,
		SceneCharacter:     stores.SceneCharacter,
		SceneInteraction:   stores.SceneInteraction,
		Write:              stores.Write,
		Applier:            stores.Applier(),
	}, idGenerator)
}
