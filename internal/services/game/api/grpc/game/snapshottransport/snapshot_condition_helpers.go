package snapshottransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func applyStressVulnerableCondition(
	ctx context.Context,
	daggerheartStore projectionstore.Store,
	write domainwrite.WritePath,
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

	normalized, err := rules.NormalizeConditions(conditions)
	if err != nil {
		return grpcerror.Internal("invalid stored conditions", err)
	}
	hasVulnerable := false
	for _, value := range normalized {
		if value == rules.ConditionVulnerable {
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
		afterSet[rules.ConditionVulnerable] = struct{}{}
	}
	if shouldRemove {
		delete(afterSet, rules.ConditionVulnerable)
	}
	afterList := make([]string, 0, len(afterSet))
	for value := range afterSet {
		afterList = append(afterList, value)
	}
	after, err := rules.NormalizeConditions(afterList)
	if err != nil {
		return grpcerror.Internal("invalid condition set", err)
	}
	added, removed := rules.DiffConditions(normalized, after)
	if len(added) == 0 && len(removed) == 0 {
		return nil
	}
	beforeStates, err := conditionStatesFromCodes(normalized)
	if err != nil {
		return grpcerror.Internal("invalid condition set", err)
	}
	afterStates, err := conditionStatesFromCodes(after)
	if err != nil {
		return grpcerror.Internal("invalid condition set", err)
	}
	addedStates, err := conditionStatesFromCodes(added)
	if err != nil {
		return grpcerror.Internal("invalid condition set", err)
	}
	removedStates, err := conditionStatesFromCodes(removed)
	if err != nil {
		return grpcerror.Internal("invalid condition set", err)
	}

	payload := daggerheartpayload.ConditionChangePayload{
		CharacterID:      ids.CharacterID(characterID),
		ConditionsBefore: beforeStates,
		ConditionsAfter:  afterStates,
		Added:            addedStates,
		Removed:          removedStates,
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

func conditionStatesFromCodes(values []string) ([]rules.ConditionState, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]rules.ConditionState, 0, len(values))
	for _, value := range values {
		state, err := rules.StandardConditionState(value)
		if err != nil {
			return nil, err
		}
		result = append(result, state)
	}
	return result, nil
}
