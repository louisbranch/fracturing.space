package daggerheart

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideDamageApply(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeDamageApplied, "character",
		func(p *DamageApplyPayload) string { return strings.TrimSpace(p.CharacterID) },
		func(s SnapshotState, hasState bool, p *DamageApplyPayload, _ func() time.Time) *command.Rejection {
			if p.ArmorSpent > 1 {
				return &command.Rejection{
					Code:    rejectionCodeDamageArmorSpendLimit,
					Message: "damage apply can spend at most one armor slot",
				}
			}
			if hasState {
				if character, ok := snapshotCharacterState(s, p.CharacterID); ok {
					if p.HpBefore != nil && character.HP != *p.HpBefore {
						return &command.Rejection{
							Code:    rejectionCodeDamageBeforeMismatch,
							Message: "damage before does not match current state",
						}
					}
					if p.ArmorBefore != nil && character.Armor != *p.ArmorBefore {
						return &command.Rejection{
							Code:    rejectionCodeDamageBeforeMismatch,
							Message: "damage before does not match current state",
						}
					}
				}
			}
			p.CharacterID = strings.TrimSpace(p.CharacterID)
			p.DamageType = strings.TrimSpace(p.DamageType)
			p.Source = strings.TrimSpace(p.Source)
			return nil
		}, now)
}

func decideAdversaryDamageApply(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncWithState(cmd, snapshotState, hasSnapshot, EventTypeAdversaryDamageApplied, "adversary",
		func(p *AdversaryDamageApplyPayload) string { return strings.TrimSpace(p.AdversaryID) },
		func(s SnapshotState, hasState bool, p *AdversaryDamageApplyPayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if adversary, ok := snapshotAdversaryState(s, p.AdversaryID); ok {
					if p.HpBefore != nil && adversary.HP != *p.HpBefore {
						return &command.Rejection{
							Code:    rejectionCodeAdversaryDamageBeforeMismatch,
							Message: "adversary damage before does not match current state",
						}
					}
					if p.ArmorBefore != nil && adversary.Armor != *p.ArmorBefore {
						return &command.Rejection{
							Code:    rejectionCodeAdversaryDamageBeforeMismatch,
							Message: "adversary damage before does not match current state",
						}
					}
				}
			}
			p.AdversaryID = strings.TrimSpace(p.AdversaryID)
			p.DamageType = strings.TrimSpace(p.DamageType)
			p.Source = strings.TrimSpace(p.Source)
			return nil
		}, now)
}

func decideDowntimeMoveApply(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeDowntimeMoveApplied, "character",
		func(p *DowntimeMoveApplyPayload) string { return strings.TrimSpace(p.CharacterID) },
		func(p *DowntimeMoveApplyPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = strings.TrimSpace(p.CharacterID)
			p.Move = strings.TrimSpace(p.Move)
			return nil
		}, now)
}

func decideCharacterTemporaryArmorApply(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeCharacterTemporaryArmorApplied, "character",
		func(p *CharacterTemporaryArmorApplyPayload) string { return strings.TrimSpace(p.CharacterID) },
		func(p *CharacterTemporaryArmorApplyPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = strings.TrimSpace(p.CharacterID)
			p.Source = strings.TrimSpace(p.Source)
			p.Duration = strings.TrimSpace(p.Duration)
			p.SourceID = strings.TrimSpace(p.SourceID)
			return nil
		}, now)
}
