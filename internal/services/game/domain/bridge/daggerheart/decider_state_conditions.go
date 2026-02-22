package daggerheart

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideGMFearSet(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeGMFearChanged, "campaign",
		func(_ *GMFearSetPayload) string { return cmd.CampaignID },
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
				// FIXME(telemetry): metric for idempotent gm fear set commands (no-op reject).
				return &command.Rejection{
					Code:    rejectionCodeGMFearUnchanged,
					Message: "gm fear after is unchanged",
				}
			}
			return nil
		},
		func(s SnapshotState, hasState bool, p GMFearSetPayload) GMFearChangedPayload {
			before := GMFearDefault
			if hasState {
				before = s.GMFear
			}
			return GMFearChangedPayload{
				Before: before,
				After:  *p.After,
				Reason: strings.TrimSpace(p.Reason),
			}
		},
		now)
}

func decideCharacterStatePatch(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeCharacterStatePatched, "character",
		func(p *CharacterStatePatchPayload) string { return strings.TrimSpace(p.CharacterID) },
		func(s SnapshotState, hasState bool, p *CharacterStatePatchPayload, _ func() time.Time) *command.Rejection {
			if hasState && isCharacterStatePatchNoMutation(s, *p) {
				// FIXME(telemetry): metric for idempotent character state patch commands.
				return &command.Rejection{
					Code:    rejectionCodeCharacterStatePatchNoMutation,
					Message: "character state patch is unchanged",
				}
			}
			p.CharacterID = strings.TrimSpace(p.CharacterID)
			return nil
		}, now)
}

func decideConditionChange(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeConditionChanged, "character",
		func(p *ConditionChangePayload) string { return strings.TrimSpace(p.CharacterID) },
		func(s SnapshotState, hasState bool, p *ConditionChangePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if hasMissingCharacterConditionRemovals(s, *p) {
					return &command.Rejection{
						Code:    rejectionCodeConditionChangeRemoveMissing,
						Message: "condition remove requires an existing condition",
					}
				}
				if isConditionChangeNoMutation(s, *p) {
					// FIXME(telemetry): metric for idempotent character condition changes.
					return &command.Rejection{
						Code:    rejectionCodeConditionChangeNoMutation,
						Message: "condition change is unchanged",
					}
				}
			}
			p.CharacterID = strings.TrimSpace(p.CharacterID)
			p.Source = strings.TrimSpace(p.Source)
			return nil
		}, now)
}

func decideHopeSpend(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeCharacterStatePatched, "character",
		func(p *HopeSpendPayload) string { return strings.TrimSpace(p.CharacterID) },
		func(_ SnapshotState, _ bool, p *HopeSpendPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = strings.TrimSpace(p.CharacterID)
			return nil
		},
		func(_ SnapshotState, _ bool, p HopeSpendPayload) CharacterStatePatchedPayload {
			return CharacterStatePatchedPayload{
				CharacterID: p.CharacterID,
				HopeBefore:  &p.Before,
				HopeAfter:   &p.After,
			}
		},
		now)
}

func decideStressSpend(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeCharacterStatePatched, "character",
		func(p *StressSpendPayload) string { return strings.TrimSpace(p.CharacterID) },
		func(_ SnapshotState, _ bool, p *StressSpendPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = strings.TrimSpace(p.CharacterID)
			return nil
		},
		func(_ SnapshotState, _ bool, p StressSpendPayload) CharacterStatePatchedPayload {
			return CharacterStatePatchedPayload{
				CharacterID:  p.CharacterID,
				StressBefore: &p.Before,
				StressAfter:  &p.After,
			}
		},
		now)
}

func decideLoadoutSwap(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeLoadoutSwapped, "character",
		func(p *LoadoutSwapPayload) string { return strings.TrimSpace(p.CharacterID) },
		func(p *LoadoutSwapPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = strings.TrimSpace(p.CharacterID)
			p.CardID = strings.TrimSpace(p.CardID)
			p.From = strings.TrimSpace(p.From)
			p.To = strings.TrimSpace(p.To)
			return nil
		}, now)
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

	return true
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
	after, err := NormalizeConditions(payload.ConditionsAfter)
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
	return hasMissingConditionRemovals(character.Conditions, payload.Removed)
}

func snapshotCharacterState(snapshot SnapshotState, characterID string) (CharacterState, bool) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return CharacterState{}, false
	}
	character, ok := snapshot.CharacterStates[characterID]
	if !ok {
		return CharacterState{}, false
	}
	character.CharacterID = characterID
	character.CampaignID = snapshot.CampaignID
	if character.LifeState == "" {
		character.LifeState = LifeStateAlive
	}
	return character, true
}
