package game

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func applyStressVulnerableCondition(
	ctx context.Context,
	daggerheartStore storage.DaggerheartStore,
	write domainwriteexec.WritePath,
	applier projection.Applier,
	campaignID string,
	sessionID string,
	characterID string,
	conditions []string,
	stressBefore int,
	stressAfter int,
	stressMax int,
	actorType event.ActorType,
	actorID string,
) error {
	if daggerheartStore == nil {
		return status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if stressMax <= 0 {
		return nil
	}
	if stressBefore == stressAfter {
		return nil
	}
	shouldAdd := stressBefore < stressMax && stressAfter == stressMax
	shouldRemove := stressBefore == stressMax && stressAfter < stressMax
	if !shouldAdd && !shouldRemove {
		return nil
	}

	normalized, err := daggerheart.NormalizeConditions(conditions)
	if err != nil {
		return grpcerror.Internal("invalid stored conditions", err)
	}
	hasVulnerable := false
	for _, value := range normalized {
		if value == daggerheart.ConditionVulnerable {
			hasVulnerable = true
			break
		}
	}
	if shouldAdd && hasVulnerable {
		return nil
	}
	if shouldRemove && !hasVulnerable {
		return nil
	}

	afterSet := make(map[string]struct{}, len(normalized)+1)
	for _, value := range normalized {
		afterSet[value] = struct{}{}
	}
	if shouldAdd {
		afterSet[daggerheart.ConditionVulnerable] = struct{}{}
	}
	if shouldRemove {
		delete(afterSet, daggerheart.ConditionVulnerable)
	}
	afterList := make([]string, 0, len(afterSet))
	for value := range afterSet {
		afterList = append(afterList, value)
	}
	after, err := daggerheart.NormalizeConditions(afterList)
	if err != nil {
		return grpcerror.Internal("invalid condition set", err)
	}
	added, removed := daggerheart.DiffConditions(normalized, after)
	if len(added) == 0 && len(removed) == 0 {
		return nil
	}

	payload := daggerheart.ConditionChangePayload{
		CharacterID:      ids.CharacterID(characterID),
		ConditionsBefore: normalized,
		ConditionsAfter:  after,
		Added:            added,
		Removed:          removed,
	}
	if err := executeDaggerheartConditionChangeCommand(
		ctx,
		write,
		applier,
		campaignID,
		characterID,
		actorType,
		actorID,
		sessionID,
		payload,
		"apply condition event",
	); err != nil {
		return err
	}

	return nil
}
