package decider

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
)

func decideAdversaryConditionChange(snapshotState snapstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeAdversaryConditionChanged, "adversary",
		func(p *payload.AdversaryConditionChangePayload) string {
			return strings.TrimSpace(p.AdversaryID.String())
		},
		func(s snapstate.SnapshotState, hasState bool, p *payload.AdversaryConditionChangePayload, _ func() time.Time) *command.Rejection {
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
		func(_ snapstate.SnapshotState, _ bool, p payload.AdversaryConditionChangePayload) payload.AdversaryConditionChangedPayload {
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

func decideAdversaryCreate(snapshotState snapstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, payload.EventTypeAdversaryCreated, "adversary",
		func(p *payload.AdversaryCreatePayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
		func(s snapstate.SnapshotState, hasState bool, p *payload.AdversaryCreatePayload, _ func() time.Time) *command.Rejection {
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
	return module.DecideFunc(cmd, payload.EventTypeAdversaryUpdated, "adversary",
		func(p *payload.AdversaryUpdatePayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
		func(p *payload.AdversaryUpdatePayload, _ func() time.Time) *command.Rejection {
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

func decideAdversaryFeatureApply(snapshotState snapstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeAdversaryUpdated, "adversary",
		func(p *payload.AdversaryFeatureApplyPayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
		func(s snapstate.SnapshotState, hasState bool, p *payload.AdversaryFeatureApplyPayload, _ func() time.Time) *command.Rejection {
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
		func(s snapstate.SnapshotState, _ bool, p payload.AdversaryFeatureApplyPayload) payload.AdversaryUpdatedPayload {
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
		func(p *payload.AdversaryDeletePayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
		func(p *payload.AdversaryDeletePayload, _ func() time.Time) *command.Rejection {
			p.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String()))
			p.Reason = strings.TrimSpace(p.Reason)
			return nil
		}, now)
}

// ── File-local helpers ─────────────────────────────────────────────────

func isAdversaryConditionChangeNoMutation(snapshot snapstate.SnapshotState, p payload.AdversaryConditionChangePayload) bool {
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

func hasMissingAdversaryConditionRemovals(snapshot snapstate.SnapshotState, p payload.AdversaryConditionChangePayload) bool {
	if len(p.Removed) == 0 {
		return false
	}
	adversary, hasAdversary := snapshotAdversaryState(snapshot, p.AdversaryID)
	if !hasAdversary {
		return false
	}
	return hasMissingConditionRemovals(adversary.Conditions, rules.ConditionCodes(p.Removed))
}

func isAdversaryCreateNoMutation(snapshot snapstate.SnapshotState, p payload.AdversaryCreatePayload) bool {
	adversary, hasAdversary := snapshotAdversaryState(snapshot, p.AdversaryID)
	if !hasAdversary {
		return false
	}
	return adversary.Name == strings.TrimSpace(p.Name) &&
		adversary.AdversaryEntryID == strings.TrimSpace(p.AdversaryEntryID) &&
		adversary.Kind == strings.TrimSpace(p.Kind) &&
		adversary.SessionID == ids.SessionID(strings.TrimSpace(p.SessionID.String())) &&
		adversary.SceneID == ids.SceneID(strings.TrimSpace(p.SceneID.String())) &&
		adversary.Notes == strings.TrimSpace(p.Notes) &&
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
		adversary.SpotlightGateID == ids.GateID(strings.TrimSpace(p.SpotlightGateID.String())) &&
		adversary.SpotlightCount == p.SpotlightCount
}

func isAdversaryFeatureApplyNoMutation(snapshot snapstate.SnapshotState, p payload.AdversaryFeatureApplyPayload) bool {
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
