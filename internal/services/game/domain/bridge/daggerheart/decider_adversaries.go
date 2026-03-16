package daggerheart

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideAdversaryConditionChange(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeAdversaryConditionChanged, "adversary",
		func(p *AdversaryConditionChangePayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
		func(s SnapshotState, hasState bool, p *AdversaryConditionChangePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if hasMissingAdversaryConditionRemovals(s, *p) {
					return &command.Rejection{
						Code:    rejectionCodeAdversaryConditionRemoveMissing,
						Message: "adversary condition remove requires an existing condition",
					}
				}
				if isAdversaryConditionChangeNoMutation(s, *p) {
					return &command.Rejection{
						Code:    rejectionCodeAdversaryConditionNoMutation,
						Message: "adversary condition change is unchanged",
					}
				}
			}
			p.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String()))
			p.Source = strings.TrimSpace(p.Source)
			return nil
		},
		func(_ SnapshotState, _ bool, p AdversaryConditionChangePayload) AdversaryConditionChangedPayload {
			return AdversaryConditionChangedPayload{
				AdversaryID: p.AdversaryID,
				Conditions:  p.ConditionsAfter,
				Added:       p.Added,
				Removed:     p.Removed,
				Source:      p.Source,
				RollSeq:     p.RollSeq,
			}
		},
		now)
}

func decideAdversaryCreate(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeAdversaryCreated, "adversary",
		func(p *AdversaryCreatePayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
		func(s SnapshotState, hasState bool, p *AdversaryCreatePayload, _ func() time.Time) *command.Rejection {
			if hasState && isAdversaryCreateNoMutation(s, *p) {
				return &command.Rejection{
					Code:    rejectionCodeAdversaryCreateNoMutation,
					Message: "adversary create is unchanged",
				}
			}
			p.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String()))
			p.AdversaryEntryID = strings.TrimSpace(p.AdversaryEntryID)
			p.Name = strings.TrimSpace(p.Name)
			p.Kind = strings.TrimSpace(p.Kind)
			p.SessionID = ids.SessionID(strings.TrimSpace(p.SessionID.String()))
			p.SceneID = ids.SceneID(strings.TrimSpace(p.SceneID.String()))
			p.Notes = strings.TrimSpace(p.Notes)
			p.SpotlightGateID = ids.GateID(strings.TrimSpace(p.SpotlightGateID.String()))
			return nil
		}, now)
}

func decideAdversaryUpdate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeAdversaryUpdated, "adversary",
		func(p *AdversaryUpdatePayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
		func(p *AdversaryUpdatePayload, _ func() time.Time) *command.Rejection {
			p.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String()))
			p.AdversaryEntryID = strings.TrimSpace(p.AdversaryEntryID)
			p.Name = strings.TrimSpace(p.Name)
			p.Kind = strings.TrimSpace(p.Kind)
			p.SessionID = ids.SessionID(strings.TrimSpace(p.SessionID.String()))
			p.SceneID = ids.SceneID(strings.TrimSpace(p.SceneID.String()))
			p.Notes = strings.TrimSpace(p.Notes)
			p.SpotlightGateID = ids.GateID(strings.TrimSpace(p.SpotlightGateID.String()))
			return nil
		}, now)
}

func decideAdversaryFeatureApply(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeAdversaryUpdated, "adversary",
		func(p *AdversaryFeatureApplyPayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
		func(s SnapshotState, hasState bool, p *AdversaryFeatureApplyPayload, _ func() time.Time) *command.Rejection {
			if hasState && isAdversaryFeatureApplyNoMutation(s, *p) {
				return &command.Rejection{
					Code:    rejectionCodeAdversaryFeatureApplyNoMutation,
					Message: "adversary feature apply is unchanged",
				}
			}
			p.ActorAdversaryID = ids.AdversaryID(strings.TrimSpace(p.ActorAdversaryID.String()))
			p.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String()))
			p.FeatureID = strings.TrimSpace(p.FeatureID)
			p.TargetCharacterID = ids.CharacterID(strings.TrimSpace(p.TargetCharacterID.String()))
			p.TargetAdversaryID = ids.AdversaryID(strings.TrimSpace(p.TargetAdversaryID.String()))
			return nil
		},
		func(s SnapshotState, _ bool, p AdversaryFeatureApplyPayload) AdversaryUpdatedPayload {
			current, _ := snapshotAdversaryState(s, p.AdversaryID)
			updatedStress := current.Stress
			if p.StressAfter != nil {
				updatedStress = *p.StressAfter
			}
			updatedFeatureStates := current.FeatureStates
			if p.FeatureStatesAfter != nil {
				updatedFeatureStates = p.FeatureStatesAfter
			}
			updatedPendingExperience := current.PendingExperience
			if p.PendingExperienceAfter != nil || p.PendingExperienceBefore != nil {
				updatedPendingExperience = p.PendingExperienceAfter
			}
			return AdversaryUpdatedPayload{
				AdversaryID:       current.AdversaryID,
				AdversaryEntryID:  current.AdversaryEntryID,
				Name:              current.Name,
				Kind:              current.Kind,
				SessionID:         current.SessionID,
				SceneID:           current.SceneID,
				Notes:             current.Notes,
				HP:                current.HP,
				HPMax:             current.HPMax,
				Stress:            updatedStress,
				StressMax:         current.StressMax,
				Evasion:           current.Evasion,
				Major:             current.Major,
				Severe:            current.Severe,
				Armor:             current.Armor,
				FeatureStates:     updatedFeatureStates,
				PendingExperience: updatedPendingExperience,
				SpotlightGateID:   current.SpotlightGateID,
				SpotlightCount:    current.SpotlightCount,
			}
		},
		now)
}

func decideAdversaryDelete(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeAdversaryDeleted, "adversary",
		func(p *AdversaryDeletePayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
		func(p *AdversaryDeletePayload, _ func() time.Time) *command.Rejection {
			p.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String()))
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
	after, err := NormalizeConditions(ConditionCodes(payload.ConditionsAfter))
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
	return hasMissingConditionRemovals(adversary.Conditions, ConditionCodes(payload.Removed))
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
		adversary.AdversaryEntryID == strings.TrimSpace(payload.AdversaryEntryID) &&
		adversary.Kind == strings.TrimSpace(payload.Kind) &&
		adversary.SessionID == ids.SessionID(strings.TrimSpace(payload.SessionID.String())) &&
		adversary.SceneID == ids.SceneID(strings.TrimSpace(payload.SceneID.String())) &&
		adversary.Notes == strings.TrimSpace(payload.Notes) &&
		adversary.HP == payload.HP &&
		adversary.HPMax == payload.HPMax &&
		adversary.Stress == payload.Stress &&
		adversary.StressMax == payload.StressMax &&
		adversary.Evasion == payload.Evasion &&
		adversary.Major == payload.Major &&
		adversary.Severe == payload.Severe &&
		adversary.Armor == payload.Armor &&
		equalAdversaryFeatureStates(adversary.FeatureStates, payload.FeatureStates) &&
		equalAdversaryPendingExperience(adversary.PendingExperience, payload.PendingExperience) &&
		adversary.SpotlightGateID == ids.GateID(strings.TrimSpace(payload.SpotlightGateID.String())) &&
		adversary.SpotlightCount == payload.SpotlightCount
}

func isAdversaryFeatureApplyNoMutation(snapshot SnapshotState, payload AdversaryFeatureApplyPayload) bool {
	adversary, hasAdversary := snapshotAdversaryState(snapshot, payload.AdversaryID)
	if !hasAdversary {
		return false
	}
	if hasIntFieldChange(payload.StressBefore, payload.StressAfter) {
		return false
	}
	if !equalAdversaryFeatureStates(adversary.FeatureStates, payload.FeatureStatesAfter) {
		return false
	}
	if !equalAdversaryPendingExperience(adversary.PendingExperience, payload.PendingExperienceAfter) {
		return false
	}
	return true
}

func snapshotAdversaryState(snapshot SnapshotState, adversaryID ids.AdversaryID) (AdversaryState, bool) {
	trimmed := ids.AdversaryID(strings.TrimSpace(adversaryID.String()))
	if trimmed == "" {
		return AdversaryState{}, false
	}
	adversary, ok := snapshot.AdversaryStates[trimmed]
	if !ok {
		return AdversaryState{}, false
	}
	adversary.AdversaryID = trimmed
	adversary.CampaignID = snapshot.CampaignID
	return adversary, true
}
