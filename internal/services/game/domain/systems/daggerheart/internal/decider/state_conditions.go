package decider

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func decideGMMoveApply(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncMulti(cmd, snapshotState, hasSnapshot,
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.GMMoveApplyPayload, _ func() time.Time) *command.Rejection {
			targetType, ok := rules.NormalizeGMMoveTargetType(string(p.Target.Type))
			if !ok {
				return &command.Rejection{
					Code:    rejectionCodeGMMoveKindUnsupported,
					Message: "gm move target is unsupported",
				}
			}
			switch targetType {
			case rules.GMMoveTargetTypeDirectMove:
				kind, ok := rules.NormalizeGMMoveKind(string(p.Target.Kind))
				if !ok {
					return &command.Rejection{
						Code:    rejectionCodeGMMoveKindUnsupported,
						Message: "gm move kind is unsupported",
					}
				}
				shape, ok := rules.NormalizeGMMoveShape(string(p.Target.Shape))
				if !ok {
					return &command.Rejection{
						Code:    rejectionCodeGMMoveShapeUnsupported,
						Message: "gm move shape is unsupported",
					}
				}
				description := strings.TrimSpace(p.Target.Description)
				if shape == rules.GMMoveShapeCustom && description == "" {
					return &command.Rejection{
						Code:    rejectionCodeGMMoveDescriptionRequired,
						Message: "gm move description is required for custom shape",
					}
				}
				p.Target.Kind = kind
				p.Target.Shape = shape
				p.Target.Description = description
				p.Target.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.Target.AdversaryID.String()))
			case rules.GMMoveTargetTypeAdversaryFeature:
				p.Target.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.Target.AdversaryID.String()))
				p.Target.FeatureID = strings.TrimSpace(p.Target.FeatureID)
				p.Target.Description = strings.TrimSpace(p.Target.Description)
			case rules.GMMoveTargetTypeEnvironmentFeature:
				p.Target.EnvironmentEntityID = ids.EnvironmentEntityID(strings.TrimSpace(p.Target.EnvironmentEntityID.String()))
				p.Target.EnvironmentID = strings.TrimSpace(p.Target.EnvironmentID)
				p.Target.FeatureID = strings.TrimSpace(p.Target.FeatureID)
				p.Target.Description = strings.TrimSpace(p.Target.Description)
			case rules.GMMoveTargetTypeAdversaryExperience:
				p.Target.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.Target.AdversaryID.String()))
				p.Target.ExperienceName = strings.TrimSpace(p.Target.ExperienceName)
				p.Target.Description = strings.TrimSpace(p.Target.Description)
			default:
				return &command.Rejection{
					Code:    rejectionCodeGMMoveKindUnsupported,
					Message: "gm move target is unsupported",
				}
			}
			p.Target.Type = targetType
			if p.FearSpent <= 0 {
				return &command.Rejection{
					Code:    rejectionCodeGMMoveFearSpentRequired,
					Message: "gm move fear_spent must be greater than zero",
				}
			}
			currentFear := daggerheartstate.GMFearDefault
			if hasState {
				currentFear = s.GMFear
			}
			if currentFear < p.FearSpent {
				return &command.Rejection{
					Code:    rejectionCodeGMMoveInsufficientFear,
					Message: "gm fear is insufficient",
				}
			}
			return nil
		},
		func(s daggerheartstate.SnapshotState, hasState bool, p payload.GMMoveApplyPayload, _ func() time.Time) ([]module.EventSpec, error) {
			currentFear := daggerheartstate.GMFearDefault
			if hasState {
				currentFear = s.GMFear
			}
			specs := []module.EventSpec{{
				Type:       payload.EventTypeGMMoveApplied,
				EntityType: "session",
				EntityID:   strings.TrimSpace(cmd.SessionID.String()),
				Payload: payload.GMMoveAppliedPayload{
					Target:    p.Target,
					FearSpent: p.FearSpent,
				},
			}}
			after := currentFear - p.FearSpent
			specs = append(specs, module.EventSpec{
				Type:       payload.EventTypeGMFearChanged,
				EntityType: "campaign",
				EntityID:   strings.TrimSpace(string(cmd.CampaignID)),
				Payload: payload.GMFearChangedPayload{
					Value:  after,
					Reason: "gm_move",
				},
			})
			return specs, nil
		},
		now,
	)
}

func decideGMFearSet(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeGMFearChanged, "campaign",
		func(_ *payload.GMFearSetPayload) string { return string(cmd.CampaignID) },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.GMFearSetPayload, _ func() time.Time) *command.Rejection {
			if p.After == nil {
				return &command.Rejection{
					Code:    rejectionCodeGMFearAfterRequired,
					Message: "gm fear after is required",
				}
			}
			after := *p.After
			if after < daggerheartstate.GMFearMin || after > daggerheartstate.GMFearMax {
				return &command.Rejection{
					Code:    rejectionCodeGMFearOutOfRange,
					Message: "gm fear after is out of range",
				}
			}
			before := daggerheartstate.GMFearDefault
			if hasState {
				before = s.GMFear
			}
			if after == before {
				return &command.Rejection{
					Code:    rejectionCodeGMFearUnchanged,
					Message: "gm fear after is unchanged",
				}
			}
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.GMFearSetPayload) payload.GMFearChangedPayload {
			return payload.GMFearChangedPayload{
				Value:  *p.After,
				Reason: strings.TrimSpace(p.Reason),
			}
		},
		now)
}

func decideCharacterStatePatch(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeCharacterStatePatched, "character",
		func(p *payload.CharacterStatePatchPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.CharacterStatePatchPayload, _ func() time.Time) *command.Rejection {
			if hasState && isCharacterStatePatchNoMutation(s, *p) {
				return &command.Rejection{
					Code:    rejectionCodeCharacterStatePatchNoMutation,
					Message: "character state patch is unchanged",
				}
			}
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.CharacterStatePatchPayload) payload.CharacterStatePatchedPayload {
			return payload.CharacterStatePatchedPayload{
				CharacterID:   p.CharacterID,
				Source:        strings.TrimSpace(p.Source),
				HP:            p.HPAfter,
				Hope:          p.HopeAfter,
				HopeMax:       p.HopeMaxAfter,
				Stress:        p.StressAfter,
				Armor:         p.ArmorAfter,
				LifeState:     p.LifeStateAfter,
				ClassState:    normalizedClassStatePtr(p.ClassStateAfter),
				SubclassState: daggerheartstate.NormalizedSubclassStatePtr(p.SubclassStateAfter),
			}
		},
		now)
}

func decideConditionChange(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeConditionChanged, "character",
		func(p *payload.ConditionChangePayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.ConditionChangePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if hasMissingCharacterConditionRemovals(s, *p) {
					return &command.Rejection{
						Code:    rejectionCodeConditionChangeRemoveMissing,
						Message: "condition remove requires an existing condition",
					}
				}
				if isConditionChangeNoMutation(s, *p) {
					return &command.Rejection{
						Code:    rejectionCodeConditionChangeNoMutation,
						Message: "condition change is unchanged",
					}
				}
			}
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.Source = strings.TrimSpace(p.Source)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.ConditionChangePayload) payload.ConditionChangedPayload {
			return payload.ConditionChangedPayload{
				CharacterID: p.CharacterID,
				Conditions:  p.ConditionsAfter,
				Added:       p.Added,
				Removed:     p.Removed,
				Source:      p.Source,
				RollSeq:     p.RollSeq,
			}
		},
		now)
}

func decideHopeSpend(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeCharacterStatePatched, "character",
		func(p *payload.HopeSpendPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ daggerheartstate.SnapshotState, _ bool, p *payload.HopeSpendPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.HopeSpendPayload) payload.CharacterStatePatchedPayload {
			return payload.CharacterStatePatchedPayload{
				CharacterID: p.CharacterID,
				Source:      "hope.spend",
				Hope:        &p.After,
			}
		},
		now)
}

func decideStressSpend(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeCharacterStatePatched, "character",
		func(p *payload.StressSpendPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ daggerheartstate.SnapshotState, _ bool, p *payload.StressSpendPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.StressSpendPayload) payload.CharacterStatePatchedPayload {
			return payload.CharacterStatePatchedPayload{
				CharacterID: p.CharacterID,
				Source:      "stress.spend",
				Stress:      &p.After,
			}
		},
		now)
}

func decideLoadoutSwap(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeLoadoutSwapped, "character",
		func(p *payload.LoadoutSwapPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ daggerheartstate.SnapshotState, _ bool, p *payload.LoadoutSwapPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.CardID = strings.TrimSpace(p.CardID)
			p.From = strings.TrimSpace(p.From)
			p.To = strings.TrimSpace(p.To)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.LoadoutSwapPayload) payload.LoadoutSwappedPayload {
			return payload.LoadoutSwappedPayload{
				CharacterID: p.CharacterID,
				CardID:      p.CardID,
				From:        p.From,
				To:          p.To,
				RecallCost:  p.RecallCost,
				Stress:      p.StressAfter,
			}
		},
		now)
}

// ── File-local helpers ─────────────────────────────────────────────────

func isConditionChangeNoMutation(snapshot daggerheartstate.SnapshotState, p payload.ConditionChangePayload) bool {
	character, hasCharacter := snapshotCharacterState(snapshot, p.CharacterID)
	if !hasCharacter {
		return false
	}

	current, err := rules.NormalizeConditions(character.Conditions)
	if err != nil {
		return false
	}
	after, err := rules.NormalizeConditions(rules.ConditionCodes(p.ConditionsAfter))
	if err != nil {
		return false
	}
	return rules.ConditionsEqual(current, after)
}

func hasMissingCharacterConditionRemovals(snapshot daggerheartstate.SnapshotState, p payload.ConditionChangePayload) bool {
	if len(p.Removed) == 0 {
		return false
	}
	character, hasCharacter := snapshotCharacterState(snapshot, p.CharacterID)
	if !hasCharacter {
		return false
	}
	return hasMissingConditionRemovals(character.Conditions, rules.ConditionCodes(p.Removed))
}
