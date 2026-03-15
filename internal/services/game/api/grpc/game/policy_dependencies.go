package game

import (
	"context"

	gameauthz "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// policyDependencies keeps write-path auth scoped to the stores actually needed
// for capability enforcement instead of carrying the full Stores bundle.
type policyDependencies = gameauthz.PolicyDeps

func newPolicyDependencies(stores Stores) policyDependencies {
	return gameauthz.PolicyDeps{
		Participant: stores.Participant,
		Character:   stores.Character,
		Audit:       stores.Audit,
	}
}

func requirePolicyWithDependencies(
	ctx context.Context,
	deps policyDependencies,
	capability domainauthz.Capability,
	campaignRecord storage.CampaignRecord,
) error {
	return gameauthz.RequirePolicy(ctx, deps, capability, campaignRecord)
}

func requirePolicyActorWithDependencies(
	ctx context.Context,
	deps policyDependencies,
	capability domainauthz.Capability,
	campaignRecord storage.CampaignRecord,
) (storage.ParticipantRecord, error) {
	return gameauthz.RequirePolicyActor(ctx, deps, capability, campaignRecord)
}
