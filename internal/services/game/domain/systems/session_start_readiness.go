package systems

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

// SessionStartReadinessStateLoader enriches aggregate state with any
// system-owned state required to evaluate session-start readiness on the read
// side.
//
// Implementations keep system-specific projection loading inside the owning
// system package so transport callers do not reconstruct typed snapshot state
// themselves.
type SessionStartReadinessStateLoader interface {
	LoadSessionStartReadinessState(
		ctx context.Context,
		campaignID ids.CampaignID,
		storeSource any,
		state aggregate.State,
	) (aggregate.State, error)
}

// SessionStartReadinessStateProvider is an optional metadata-registry
// extension for systems that need additional read-side state before the
// session-start readiness workflow can execute bound module hooks.
type SessionStartReadinessStateProvider interface {
	SessionStartReadinessStateLoader() SessionStartReadinessStateLoader
}

// ResolveSessionStartReadinessState lets one registered system enrich
// aggregate state for session-start readiness preview.
//
// Missing registries, unknown systems, and systems without the optional loader
// all leave the state unchanged.
func ResolveSessionStartReadinessState(
	ctx context.Context,
	registry *MetadataRegistry,
	campaignID ids.CampaignID,
	systemID SystemID,
	storeSource any,
	state aggregate.State,
) (aggregate.State, error) {
	if registry == nil || systemID == SystemIDUnspecified {
		return state, nil
	}
	system := registry.Get(systemID)
	if system == nil {
		return state, nil
	}
	provider, ok := system.(SessionStartReadinessStateProvider)
	if !ok {
		return state, nil
	}
	loader := provider.SessionStartReadinessStateLoader()
	if loader == nil {
		return state, fmt.Errorf("system %s@%s returned a nil session-start readiness state loader", system.ID(), system.Version())
	}
	return loader.LoadSessionStartReadinessState(ctx, campaignID, storeSource, state)
}
