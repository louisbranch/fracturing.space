package daggerheart

import (
	"reflect"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideGMMoveApply(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncMulti(cmd, snapshotState, hasSnapshot,
		func(s SnapshotState, hasState bool, p *GMMoveApplyPayload, _ func() time.Time) *command.Rejection {
			targetType, ok := NormalizeGMMoveTargetType(string(p.Target.Type))
			if !ok {
				return &command.Rejection{
					Code:    rejectionCodeGMMoveKindUnsupported,
					Message: "gm move target is unsupported",
				}
			}
			switch targetType {
			case GMMoveTargetTypeDirectMove:
				kind, ok := NormalizeGMMoveKind(string(p.Target.Kind))
				if !ok {
					return &command.Rejection{
						Code:    rejectionCodeGMMoveKindUnsupported,
						Message: "gm move kind is unsupported",
					}
				}
				shape, ok := NormalizeGMMoveShape(string(p.Target.Shape))
				if !ok {
					return &command.Rejection{
						Code:    rejectionCodeGMMoveShapeUnsupported,
						Message: "gm move shape is unsupported",
					}
				}
				description := strings.TrimSpace(p.Target.Description)
				if shape == GMMoveShapeCustom && description == "" {
					return &command.Rejection{
						Code:    rejectionCodeGMMoveDescriptionRequired,
						Message: "gm move description is required for custom shape",
					}
				}
				p.Target.Kind = kind
				p.Target.Shape = shape
				p.Target.Description = description
				p.Target.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.Target.AdversaryID.String()))
			case GMMoveTargetTypeAdversaryFeature:
				p.Target.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.Target.AdversaryID.String()))
				p.Target.FeatureID = strings.TrimSpace(p.Target.FeatureID)
				p.Target.Description = strings.TrimSpace(p.Target.Description)
			case GMMoveTargetTypeEnvironmentFeature:
				p.Target.EnvironmentEntityID = ids.EnvironmentEntityID(strings.TrimSpace(p.Target.EnvironmentEntityID.String()))
				p.Target.EnvironmentID = strings.TrimSpace(p.Target.EnvironmentID)
				p.Target.FeatureID = strings.TrimSpace(p.Target.FeatureID)
				p.Target.Description = strings.TrimSpace(p.Target.Description)
			case GMMoveTargetTypeAdversaryExperience:
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
			currentFear := GMFearDefault
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
		func(s SnapshotState, hasState bool, p GMMoveApplyPayload, _ func() time.Time) ([]module.EventSpec, error) {
			currentFear := GMFearDefault
			if hasState {
				currentFear = s.GMFear
			}
			specs := []module.EventSpec{{
				Type:       EventTypeGMMoveApplied,
				EntityType: "session",
				EntityID:   strings.TrimSpace(cmd.SessionID.String()),
				Payload: GMMoveAppliedPayload{
					Target:    p.Target,
					FearSpent: p.FearSpent,
				},
			}}
			after := currentFear - p.FearSpent
			specs = append(specs, module.EventSpec{
				Type:       EventTypeGMFearChanged,
				EntityType: "campaign",
				EntityID:   strings.TrimSpace(string(cmd.CampaignID)),
				Payload: GMFearChangedPayload{
					Value:  after,
					Reason: "gm_move",
				},
			})
			return specs, nil
		},
		now,
	)
}

func decideGMFearSet(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeGMFearChanged, "campaign",
		func(_ *GMFearSetPayload) string { return string(cmd.CampaignID) },
		func(s SnapshotState, hasState bool, p *GMFearSetPayload, _ func() time.Time) *command.Rejection {
			if p.After == nil {
				return &command.Rejection{
					Code:    rejectionCodeGMFearAfterRequired,
					Message: "gm fear after is required",
				}
			}
			after := *p.After
			if after < GMFearMin || after > GMFearMax {
				return &command.Rejection{
					Code:    rejectionCodeGMFearOutOfRange,
					Message: "gm fear after is out of range",
				}
			}
			before := GMFearDefault
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
		func(_ SnapshotState, _ bool, p GMFearSetPayload) GMFearChangedPayload {
			return GMFearChangedPayload{
				Value:  *p.After,
				Reason: strings.TrimSpace(p.Reason),
			}
		},
		now)
}

func decideCharacterStatePatch(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeCharacterStatePatched, "character",
		func(p *CharacterStatePatchPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(s SnapshotState, hasState bool, p *CharacterStatePatchPayload, _ func() time.Time) *command.Rejection {
			if hasState && isCharacterStatePatchNoMutation(s, *p) {
				return &command.Rejection{
					Code:    rejectionCodeCharacterStatePatchNoMutation,
					Message: "character state patch is unchanged",
				}
			}
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			return nil
		},
		func(_ SnapshotState, _ bool, p CharacterStatePatchPayload) CharacterStatePatchedPayload {
			return CharacterStatePatchedPayload{
				CharacterID:   p.CharacterID,
				Source:        strings.TrimSpace(p.Source),
				HP:            p.HPAfter,
				Hope:          p.HopeAfter,
				HopeMax:       p.HopeMaxAfter,
				Stress:        p.StressAfter,
				Armor:         p.ArmorAfter,
				LifeState:     p.LifeStateAfter,
				ClassState:    normalizedClassStatePtr(p.ClassStateAfter),
				SubclassState: normalizedSubclassStatePtr(p.SubclassStateAfter),
			}
		},
		now)
}

func decideConditionChange(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeConditionChanged, "character",
		func(p *ConditionChangePayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(s SnapshotState, hasState bool, p *ConditionChangePayload, _ func() time.Time) *command.Rejection {
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
		func(_ SnapshotState, _ bool, p ConditionChangePayload) ConditionChangedPayload {
			return ConditionChangedPayload{
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

func decideHopeSpend(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeCharacterStatePatched, "character",
		func(p *HopeSpendPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ SnapshotState, _ bool, p *HopeSpendPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			return nil
		},
		func(_ SnapshotState, _ bool, p HopeSpendPayload) CharacterStatePatchedPayload {
			return CharacterStatePatchedPayload{
				CharacterID: p.CharacterID,
				Source:      "hope.spend",
				Hope:        &p.After,
			}
		},
		now)
}

func decideStressSpend(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeCharacterStatePatched, "character",
		func(p *StressSpendPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ SnapshotState, _ bool, p *StressSpendPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			return nil
		},
		func(_ SnapshotState, _ bool, p StressSpendPayload) CharacterStatePatchedPayload {
			return CharacterStatePatchedPayload{
				CharacterID: p.CharacterID,
				Source:      "stress.spend",
				Stress:      &p.After,
			}
		},
		now)
}

func decideLoadoutSwap(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeLoadoutSwapped, "character",
		func(p *LoadoutSwapPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ SnapshotState, _ bool, p *LoadoutSwapPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.CardID = strings.TrimSpace(p.CardID)
			p.From = strings.TrimSpace(p.From)
			p.To = strings.TrimSpace(p.To)
			return nil
		},
		func(_ SnapshotState, _ bool, p LoadoutSwapPayload) LoadoutSwappedPayload {
			return LoadoutSwappedPayload{
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

func isCharacterStatePatchNoMutation(snapshot SnapshotState, payload CharacterStatePatchPayload) bool {
	character, hasCharacter := snapshotCharacterState(snapshot, payload.CharacterID)
	if !hasCharacter {
		return false
	}

	if payload.HPAfter != nil {
		if character.HP != *payload.HPAfter {
			return false
		}
	} else if payload.HPBefore != nil && character.HP == 0 && character.HP != *payload.HPBefore {
		return false
	}
	if payload.HopeAfter != nil && character.Hope != *payload.HopeAfter {
		return false
	}
	if payload.HopeMaxAfter != nil && character.HopeMax != *payload.HopeMaxAfter {
		return false
	}
	if payload.StressAfter != nil && character.Stress != *payload.StressAfter {
		return false
	}
	if payload.ArmorAfter != nil && character.Armor != *payload.ArmorAfter {
		return false
	}
	if payload.LifeStateAfter != nil && character.LifeState != *payload.LifeStateAfter {
		return false
	}
	if payload.ClassStateAfter != nil {
		current := snapshot.CharacterClassStates[payload.CharacterID].Normalized()
		if !reflect.DeepEqual(current, payload.ClassStateAfter.Normalized()) {
			return false
		}
	}
	if payload.SubclassStateAfter != nil {
		current := snapshot.CharacterSubclassStates[payload.CharacterID].Normalized()
		if !reflect.DeepEqual(current, payload.SubclassStateAfter.Normalized()) {
			return false
		}
	}

	return true
}

func normalizedClassStatePtr(value *CharacterClassState) *CharacterClassState {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &normalized
}

func isConditionChangeNoMutation(snapshot SnapshotState, payload ConditionChangePayload) bool {
	character, hasCharacter := snapshotCharacterState(snapshot, payload.CharacterID)
	if !hasCharacter {
		return false
	}

	current, err := NormalizeConditions(character.Conditions)
	if err != nil {
		return false
	}
	after, err := NormalizeConditions(ConditionCodes(payload.ConditionsAfter))
	if err != nil {
		return false
	}
	return ConditionsEqual(current, after)
}

func hasMissingCharacterConditionRemovals(snapshot SnapshotState, payload ConditionChangePayload) bool {
	if len(payload.Removed) == 0 {
		return false
	}
	character, hasCharacter := snapshotCharacterState(snapshot, payload.CharacterID)
	if !hasCharacter {
		return false
	}
	return hasMissingConditionRemovals(character.Conditions, ConditionCodes(payload.Removed))
}

func snapshotCharacterState(snapshot SnapshotState, characterID ids.CharacterID) (CharacterState, bool) {
	trimmed := ids.CharacterID(strings.TrimSpace(characterID.String()))
	if trimmed == "" {
		return CharacterState{}, false
	}
	character, ok := snapshot.CharacterStates[trimmed]
	if !ok {
		return CharacterState{}, false
	}
	character.CharacterID = trimmed.String()
	character.CampaignID = snapshot.CampaignID.String()
	if character.LifeState == "" {
		character.LifeState = LifeStateAlive
	}
	return character, true
}
