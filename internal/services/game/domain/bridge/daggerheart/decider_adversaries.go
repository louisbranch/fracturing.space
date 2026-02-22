package daggerheart

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideAdversaryConditionChange(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeAdversaryConditionChanged, "adversary",
		func(p *AdversaryConditionChangePayload) string { return strings.TrimSpace(p.AdversaryID) },
		func(s SnapshotState, hasState bool, p *AdversaryConditionChangePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if hasMissingAdversaryConditionRemovals(s, *p) {
					return &command.Rejection{
						Code:    rejectionCodeAdversaryConditionRemoveMissing,
						Message: "adversary condition remove requires an existing condition",
					}
				}
				if isAdversaryConditionChangeNoMutation(s, *p) {
					// FIXME(telemetry): metric for idempotent adversary condition changes.
					return &command.Rejection{
						Code:    rejectionCodeAdversaryConditionNoMutation,
						Message: "adversary condition change is unchanged",
					}
				}
			}
			p.AdversaryID = strings.TrimSpace(p.AdversaryID)
			p.Source = strings.TrimSpace(p.Source)
			return nil
		}, now)
}

func decideAdversaryCreate(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeAdversaryCreated, "adversary",
		func(p *AdversaryCreatePayload) string { return strings.TrimSpace(p.AdversaryID) },
		func(s SnapshotState, hasState bool, p *AdversaryCreatePayload, _ func() time.Time) *command.Rejection {
			if hasState && isAdversaryCreateNoMutation(s, *p) {
				// FIXME(telemetry): metric for idempotent adversary creation commands.
				return &command.Rejection{
					Code:    rejectionCodeAdversaryCreateNoMutation,
					Message: "adversary create is unchanged",
				}
			}
			p.AdversaryID = strings.TrimSpace(p.AdversaryID)
			p.Name = strings.TrimSpace(p.Name)
			p.Kind = strings.TrimSpace(p.Kind)
			p.SessionID = strings.TrimSpace(p.SessionID)
			p.Notes = strings.TrimSpace(p.Notes)
			return nil
		}, now)
}

func decideAdversaryUpdate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeAdversaryUpdated, "adversary",
		func(p *AdversaryUpdatePayload) string { return strings.TrimSpace(p.AdversaryID) },
		func(p *AdversaryUpdatePayload, _ func() time.Time) *command.Rejection {
			p.AdversaryID = strings.TrimSpace(p.AdversaryID)
			p.Name = strings.TrimSpace(p.Name)
			p.Kind = strings.TrimSpace(p.Kind)
			p.SessionID = strings.TrimSpace(p.SessionID)
			p.Notes = strings.TrimSpace(p.Notes)
			return nil
		}, now)
}

func decideAdversaryDelete(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeAdversaryDeleted, "adversary",
		func(p *AdversaryDeletePayload) string { return strings.TrimSpace(p.AdversaryID) },
		func(p *AdversaryDeletePayload, _ func() time.Time) *command.Rejection {
			p.AdversaryID = strings.TrimSpace(p.AdversaryID)
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		}, now)
}

func isAdversaryConditionChangeNoMutation(snapshot SnapshotState, payload AdversaryConditionChangePayload) bool {
	adversary, hasAdversary := snapshotAdversaryState(snapshot, payload.AdversaryID)
	if !hasAdversary {
		return false
	}

	current, err := NormalizeConditions(adversary.Conditions)
	if err != nil {
		return false
	}
	after, err := NormalizeConditions(payload.ConditionsAfter)
	if err != nil {
		return false
	}
	return ConditionsEqual(current, after)
}

func hasMissingAdversaryConditionRemovals(snapshot SnapshotState, payload AdversaryConditionChangePayload) bool {
	if len(payload.Removed) == 0 {
		return false
	}
	adversary, hasAdversary := snapshotAdversaryState(snapshot, payload.AdversaryID)
	if !hasAdversary {
		return false
	}
	return hasMissingConditionRemovals(adversary.Conditions, payload.Removed)
}

func hasMissingConditionRemovals(current, removed []string) bool {
	normalizedCurrent, err := NormalizeConditions(current)
	if err != nil {
		return false
	}
	normalizedRemoved, err := NormalizeConditions(removed)
	if err != nil {
		return false
	}

	currentSet := make(map[string]struct{}, len(normalizedCurrent))
	for _, value := range normalizedCurrent {
		currentSet[value] = struct{}{}
	}
	for _, value := range normalizedRemoved {
		if _, ok := currentSet[value]; !ok {
			return true
		}
	}
	return false
}

func isAdversaryCreateNoMutation(snapshot SnapshotState, payload AdversaryCreatePayload) bool {
	adversary, hasAdversary := snapshotAdversaryState(snapshot, payload.AdversaryID)
	if !hasAdversary {
		return false
	}
	return adversary.Name == strings.TrimSpace(payload.Name) &&
		adversary.Kind == strings.TrimSpace(payload.Kind) &&
		adversary.SessionID == strings.TrimSpace(payload.SessionID) &&
		adversary.Notes == strings.TrimSpace(payload.Notes) &&
		adversary.HP == payload.HP &&
		adversary.HPMax == payload.HPMax &&
		adversary.Stress == payload.Stress &&
		adversary.StressMax == payload.StressMax &&
		adversary.Evasion == payload.Evasion &&
		adversary.Major == payload.Major &&
		adversary.Severe == payload.Severe &&
		adversary.Armor == payload.Armor
}

func snapshotAdversaryState(snapshot SnapshotState, adversaryID string) (AdversaryState, bool) {
	adversaryID = strings.TrimSpace(adversaryID)
	if adversaryID == "" {
		return AdversaryState{}, false
	}
	adversary, ok := snapshot.AdversaryStates[adversaryID]
	if !ok {
		return AdversaryState{}, false
	}
	adversary.AdversaryID = adversaryID
	adversary.CampaignID = snapshot.CampaignID
	return adversary, true
}
