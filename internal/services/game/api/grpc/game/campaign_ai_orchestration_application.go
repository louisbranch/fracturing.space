package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/interactiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// CampaignAIOrchestrationDeps declares the explicit collaborators used by the
// internal AI GM turn orchestration surface.
type CampaignAIOrchestrationDeps struct {
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	Session            storage.SessionStore
	SessionRecap       storage.SessionRecapStore
	SessionInteraction storage.SessionInteractionStore
	Scene              storage.SceneStore
	SceneInteraction   storage.SceneInteractionStore
	Write              WritePath
	Applier            projection.Applier
}

// NewCampaignAIOrchestrationApplication builds the interaction-owned AI
// orchestration application from explicit root-game dependencies.
func NewCampaignAIOrchestrationApplication(
	deps CampaignAIOrchestrationDeps,
	idGenerator func() (string, error),
) interactiontransport.AIOrchestrationApplication {
	return newCampaignAIOrchestrationApplicationWithDependencies(deps, idGenerator)
}

func newCampaignAIOrchestrationApplicationWithDependencies(
	deps CampaignAIOrchestrationDeps,
	idGenerator func() (string, error),
) interactiontransport.AIOrchestrationApplication {
	return interactiontransport.NewAIOrchestrationApplication(interactiontransport.Deps{
		Campaign:           deps.Campaign,
		Participant:        deps.Participant,
		Session:            deps.Session,
		SessionRecap:       deps.SessionRecap,
		SessionInteraction: deps.SessionInteraction,
		Scene:              deps.Scene,
		SceneInteraction:   deps.SceneInteraction,
		Write:              deps.Write,
		Applier:            deps.Applier,
	}, idGenerator)
}
