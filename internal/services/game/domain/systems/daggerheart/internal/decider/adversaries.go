package decider

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func decideAdversaryConditionChange(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeAdversaryConditionChanged, "adversary",
		func(p *payload.AdversaryConditionChangePayload) string {
			return normalize.ID(p.AdversaryID).String()
		},
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.AdversaryConditionChangePayload, _ func() time.Time) *command.Rejection {
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
			p.AdversaryID = normalize.ID(p.AdversaryID)
			p.Source = normalize.String(p.Source)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.AdversaryConditionChangePayload) payload.AdversaryConditionChangedPayload {
			return payload.AdversaryConditionChangedPayload{
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

func decideAdversaryCreate(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, payload.EventTypeAdversaryCreated, "adversary",
		func(p *payload.AdversaryCreatePayload) string { return normalize.ID(p.AdversaryID).String() },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.AdversaryCreatePayload, _ func() time.Time) *command.Rejection {
			if hasState && isAdversaryCreateNoMutation(s, *p) {
				return &command.Rejection{
					Code:    rejectionCodeAdversaryCreateNoMutation,
					Message: "adversary create is unchanged",
				}
			}
			p.AdversaryID = normalize.ID(p.AdversaryID)
			p.AdversaryEntryID = normalize.String(p.AdversaryEntryID)
			p.Name = normalize.String(p.Name)
			p.Kind = normalize.String(p.Kind)
			p.SessionID = normalize.ID(p.SessionID)
			p.SceneID = normalize.ID(p.SceneID)
			p.Notes = normalize.String(p.Notes)
			p.SpotlightGateID = normalize.ID(p.SpotlightGateID)
			return nil
		}, now)
}

func decideAdversaryUpdate(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, payload.EventTypeAdversaryUpdated, "adversary",
		func(p *payload.AdversaryUpdatePayload) string { return normalize.ID(p.AdversaryID).String() },
		func(p *payload.AdversaryUpdatePayload, _ func() time.Time) *command.Rejection {
			p.AdversaryID = normalize.ID(p.AdversaryID)
			p.AdversaryEntryID = normalize.String(p.AdversaryEntryID)
			p.Name = normalize.String(p.Name)
			p.Kind = normalize.String(p.Kind)
			p.SessionID = normalize.ID(p.SessionID)
			p.SceneID = normalize.ID(p.SceneID)
			p.Notes = normalize.String(p.Notes)
			p.SpotlightGateID = normalize.ID(p.SpotlightGateID)
			return nil
		}, now)
}

func decideAdversaryFeatureApply(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeAdversaryUpdated, "adversary",
		func(p *payload.AdversaryFeatureApplyPayload) string { return normalize.ID(p.AdversaryID).String() },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.AdversaryFeatureApplyPayload, _ func() time.Time) *command.Rejection {
			if hasState && isAdversaryFeatureApplyNoMutation(s, *p) {
				return &command.Rejection{
					Code:    rejectionCodeAdversaryFeatureApplyNoMutation,
					Message: "adversary feature apply is unchanged",
				}
			}
			p.ActorAdversaryID = normalize.ID(p.ActorAdversaryID)
			p.AdversaryID = normalize.ID(p.AdversaryID)
			p.FeatureID = normalize.String(p.FeatureID)
			p.TargetCharacterID = normalize.ID(p.TargetCharacterID)
			p.TargetAdversaryID = normalize.ID(p.TargetAdversaryID)
			return nil
		},
		func(s daggerheartstate.SnapshotState, _ bool, p payload.AdversaryFeatureApplyPayload) payload.AdversaryUpdatedPayload {
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
			return payload.AdversaryUpdatedPayload{
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
	return module.DecideFunc(cmd, payload.EventTypeAdversaryDeleted, "adversary",
		func(p *payload.AdversaryDeletePayload) string { return normalize.ID(p.AdversaryID).String() },
		func(p *payload.AdversaryDeletePayload, _ func() time.Time) *command.Rejection {
			p.AdversaryID = normalize.ID(p.AdversaryID)
			p.Reason = normalize.String(p.Reason)
			return nil
		}, now)
}

// ── File-local helpers ─────────────────────────────────────────────────

func isAdversaryConditionChangeNoMutation(snapshot daggerheartstate.SnapshotState, p payload.AdversaryConditionChangePayload) bool {
	adversary, hasAdversary := snapshotAdversaryState(snapshot, p.AdversaryID)
	if !hasAdversary {
		return false
	}

	current, err := rules.NormalizeConditions(adversary.Conditions)
	if err != nil {
		return false
	}
	after, err := rules.NormalizeConditions(rules.ConditionCodes(p.ConditionsAfter))
	if err != nil {
		return false
	}
	return rules.ConditionsEqual(current, after)
}

func hasMissingAdversaryConditionRemovals(snapshot daggerheartstate.SnapshotState, p payload.AdversaryConditionChangePayload) bool {
	if len(p.Removed) == 0 {
		return false
	}
	adversary, hasAdversary := snapshotAdversaryState(snapshot, p.AdversaryID)
	if !hasAdversary {
		return false
	}
	return hasMissingConditionRemovals(adversary.Conditions, rules.ConditionCodes(p.Removed))
}

func isAdversaryCreateNoMutation(snapshot daggerheartstate.SnapshotState, p payload.AdversaryCreatePayload) bool {
	adversary, hasAdversary := snapshotAdversaryState(snapshot, p.AdversaryID)
	if !hasAdversary {
		return false
	}
	return adversary.Name == normalize.String(p.Name) &&
		adversary.AdversaryEntryID == normalize.String(p.AdversaryEntryID) &&
		adversary.Kind == normalize.String(p.Kind) &&
		adversary.SessionID == normalize.ID(p.SessionID) &&
		adversary.SceneID == normalize.ID(p.SceneID) &&
		adversary.Notes == normalize.String(p.Notes) &&
		adversary.HP == p.HP &&
		adversary.HPMax == p.HPMax &&
		adversary.Stress == p.Stress &&
		adversary.StressMax == p.StressMax &&
		adversary.Evasion == p.Evasion &&
		adversary.Major == p.Major &&
		adversary.Severe == p.Severe &&
		adversary.Armor == p.Armor &&
		equalAdversaryFeatureStates(adversary.FeatureStates, p.FeatureStates) &&
		equalAdversaryPendingExperience(adversary.PendingExperience, p.PendingExperience) &&
		adversary.SpotlightGateID == normalize.ID(p.SpotlightGateID) &&
		adversary.SpotlightCount == p.SpotlightCount
}

func isAdversaryFeatureApplyNoMutation(snapshot daggerheartstate.SnapshotState, p payload.AdversaryFeatureApplyPayload) bool {
	adversary, hasAdversary := snapshotAdversaryState(snapshot, p.AdversaryID)
	if !hasAdversary {
		return false
	}
	if hasIntFieldChange(p.StressBefore, p.StressAfter) {
		return false
	}
	if !equalAdversaryFeatureStates(adversary.FeatureStates, p.FeatureStatesAfter) {
		return false
	}
	if !equalAdversaryPendingExperience(adversary.PendingExperience, p.PendingExperienceAfter) {
		return false
	}
	return true
}
