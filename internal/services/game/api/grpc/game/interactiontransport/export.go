package interactiontransport

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
)

// LoadInteractionState exposes the canonical interaction read model to sibling
// transport packages so scene lifecycle responses reuse the same control
// contract instead of rebuilding it locally.
func LoadInteractionState(ctx context.Context, deps Deps, campaignID string) (*campaignv1.InteractionState, error) {
	return newInteractionApplicationWithDependencies(deps, id.NewID).GetInteractionState(ctx, campaignID)
}

// ActivateScene loads the scene-activation transition through the canonical
// interaction application so callers share the same active-scene rules.
func ActivateScene(ctx context.Context, deps Deps, campaignID string, req *campaignv1.ActivateSceneRequest) (*campaignv1.InteractionState, error) {
	return newInteractionApplicationWithDependencies(deps, id.NewID).ActivateScene(ctx, campaignID, req)
}
