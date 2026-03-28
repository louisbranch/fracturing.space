package scenetransport

import (
	"context"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/interactiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// Deps holds all dependencies needed by the scene transport layer.
type Deps struct {
	Auth               authz.PolicyDeps
	Campaign           storage.CampaignStore
	Participant        storage.ParticipantStore
	Character          storage.CharacterStore
	Event              storage.EventHistoryStore
	Session            storage.SessionStore
	SessionInteraction storage.SessionInteractionStore
	Scene              storage.SceneStore
	SceneCharacter     storage.SceneCharacterStore
	SceneInteraction   storage.SceneInteractionStore
	SceneGMInteraction storage.SceneGMInteractionStore
	Write              domainwrite.WritePath
	Applier            projection.Applier
}

// interactionDeps derives the interactiontransport.Deps from the scene
// dependency set. This centralizes the mapping so callers don't repeat the
// field-by-field construction.
func (d Deps) interactionDeps() interactiontransport.Deps {
	return interactiontransport.Deps{
		Auth:               d.Auth,
		Campaign:           d.Campaign,
		Participant:        d.Participant,
		Character:          d.Character,
		Event:              d.Event,
		Session:            d.Session,
		SessionInteraction: d.SessionInteraction,
		Scene:              d.Scene,
		SceneCharacter:     d.SceneCharacter,
		SceneInteraction:   d.SceneInteraction,
		SceneGMInteraction: d.SceneGMInteraction,
		Write:              d.Write,
		Applier:            d.Applier,
	}
}

// sceneApplication coordinates scene transport use-cases across focused files
// (lifecycle, character membership, gates, and spotlight operations) while
// keeping scene-owned reads and write execution explicit.
type sceneApplication struct {
	auth        authz.PolicyDeps
	stores      sceneApplicationStores
	interaction interactiontransport.Deps
	write       domainwrite.WritePath
	applier     projection.Applier
	clock       func() time.Time
	idGenerator func() (string, error)
}

type sceneApplicationStores struct {
	Campaign storage.CampaignStore
	Scene    storage.SceneStore
}

func newSceneApplication(service *Service) sceneApplication {
	if service == nil {
		return sceneApplication{}
	}
	return service.app
}

func newSceneApplicationWithDependencies(deps Deps, clock func() time.Time, idGenerator func() (string, error)) sceneApplication {
	app := sceneApplication{
		auth: deps.Auth,
		stores: sceneApplicationStores{
			Campaign: deps.Campaign,
			Scene:    deps.Scene,
		},
		interaction: deps.interactionDeps(),
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

func (a sceneApplication) interactionState(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
	return interactiontransport.LoadInteractionState(ctx, a.interaction, campaignID)
}

func (a sceneApplication) activateScene(ctx context.Context, campaignID, sceneID string) (*gamev1.InteractionState, error) {
	return interactiontransport.ActivateScene(ctx, a.interaction, campaignID, &gamev1.ActivateSceneRequest{
		CampaignId: campaignID,
		SceneId:    sceneID,
	})
}
