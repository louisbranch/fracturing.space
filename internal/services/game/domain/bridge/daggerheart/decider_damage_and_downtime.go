package daggerheart

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideDamageApply(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeDamageApplied, "character",
		func(p *DamageApplyPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
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
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.DamageType = strings.TrimSpace(p.DamageType)
			p.Source = strings.TrimSpace(p.Source)
			return nil
		},
		func(_ SnapshotState, _ bool, p DamageApplyPayload) DamageAppliedPayload {
			return DamageAppliedPayload{
				CharacterID:        p.CharacterID,
				Hp:                 p.HpAfter,
				Stress:             p.StressAfter,
				Armor:              p.ArmorAfter,
				ArmorSpent:         p.ArmorSpent,
				Severity:           p.Severity,
				Marks:              p.Marks,
				DamageType:         p.DamageType,
				RollSeq:            p.RollSeq,
				ResistPhysical:     p.ResistPhysical,
				ResistMagic:        p.ResistMagic,
				ImmunePhysical:     p.ImmunePhysical,
				ImmuneMagic:        p.ImmuneMagic,
				Direct:             p.Direct,
				MassiveDamage:      p.MassiveDamage,
				Mitigated:          p.Mitigated,
				Source:             p.Source,
				SourceCharacterIDs: p.SourceCharacterIDs,
			}
		},
		now)
}

// decideMultiTargetDamageApply handles batch damage across multiple characters
// atomically. Each target entry produces one damage_applied event. All events
// are batch-appended in a single decision, avoiding the sequential failure
// window of N individual commands.
func decideMultiTargetDamageApply(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncMulti(cmd, snapshotState, hasSnapshot,
		// validate: reject if no targets
		func(s SnapshotState, hasState bool, p *MultiTargetDamageApplyPayload, _ func() time.Time) *command.Rejection {
			if len(p.Targets) == 0 {
				return &command.Rejection{
					Code:    "MULTI_TARGET_NO_TARGETS",
					Message: "multi-target damage requires at least one target",
				}
			}
			for i := range p.Targets {
				t := &p.Targets[i]
				if t.ArmorSpent > 1 {
					return &command.Rejection{
						Code:    rejectionCodeDamageArmorSpendLimit,
						Message: "damage apply can spend at most one armor slot",
					}
				}
				if hasState {
					if character, ok := snapshotCharacterState(s, t.CharacterID); ok {
						if t.HpBefore != nil && character.HP != *t.HpBefore {
							return &command.Rejection{
								Code:    rejectionCodeDamageBeforeMismatch,
								Message: "damage before does not match current state",
							}
						}
						if t.ArmorBefore != nil && character.Armor != *t.ArmorBefore {
							return &command.Rejection{
								Code:    rejectionCodeDamageBeforeMismatch,
								Message: "damage before does not match current state",
							}
						}
					}
				}
				t.CharacterID = ids.CharacterID(strings.TrimSpace(t.CharacterID.String()))
				t.DamageType = strings.TrimSpace(t.DamageType)
				t.Source = strings.TrimSpace(t.Source)
			}
			return nil
		},
		// expand: one EventSpec per target, all emitting damage_applied
		func(s SnapshotState, _ bool, p MultiTargetDamageApplyPayload, _ func() time.Time) ([]module.EventSpec, error) {
			specs := make([]module.EventSpec, 0, len(p.Targets))
			for _, t := range p.Targets {
				specs = append(specs, module.EventSpec{
					Type:       EventTypeDamageApplied,
					EntityType: "character",
					EntityID:   t.CharacterID.String(),
					Payload: DamageAppliedPayload{
						CharacterID:        t.CharacterID,
						Hp:                 t.HpAfter,
						Stress:             t.StressAfter,
						Armor:              t.ArmorAfter,
						ArmorSpent:         t.ArmorSpent,
						Severity:           t.Severity,
						Marks:              t.Marks,
						DamageType:         t.DamageType,
						RollSeq:            t.RollSeq,
						ResistPhysical:     t.ResistPhysical,
						ResistMagic:        t.ResistMagic,
						ImmunePhysical:     t.ImmunePhysical,
						ImmuneMagic:        t.ImmuneMagic,
						Direct:             t.Direct,
						MassiveDamage:      t.MassiveDamage,
						Mitigated:          t.Mitigated,
						Source:             t.Source,
						SourceCharacterIDs: t.SourceCharacterIDs,
					},
				})
			}
			return specs, nil
		}, now)
}

func decideAdversaryDamageApply(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeAdversaryDamageApplied, "adversary",
		func(p *AdversaryDamageApplyPayload) string { return strings.TrimSpace(p.AdversaryID.String()) },
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
			p.AdversaryID = ids.AdversaryID(strings.TrimSpace(p.AdversaryID.String()))
			p.DamageType = strings.TrimSpace(p.DamageType)
			p.Source = strings.TrimSpace(p.Source)
			return nil
		},
		func(_ SnapshotState, _ bool, p AdversaryDamageApplyPayload) AdversaryDamageAppliedPayload {
			return AdversaryDamageAppliedPayload{
				AdversaryID:        p.AdversaryID,
				Hp:                 p.HpAfter,
				Armor:              p.ArmorAfter,
				ArmorSpent:         p.ArmorSpent,
				Severity:           p.Severity,
				Marks:              p.Marks,
				DamageType:         p.DamageType,
				RollSeq:            p.RollSeq,
				ResistPhysical:     p.ResistPhysical,
				ResistMagic:        p.ResistMagic,
				ImmunePhysical:     p.ImmunePhysical,
				ImmuneMagic:        p.ImmuneMagic,
				Direct:             p.Direct,
				MassiveDamage:      p.MassiveDamage,
				Mitigated:          p.Mitigated,
				Source:             p.Source,
				SourceCharacterIDs: p.SourceCharacterIDs,
			}
		},
		now)
}

func decideCharacterTemporaryArmorApply(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeCharacterTemporaryArmorApplied, "character",
		func(p *CharacterTemporaryArmorApplyPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(p *CharacterTemporaryArmorApplyPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.Source = strings.TrimSpace(p.Source)
			p.Duration = strings.TrimSpace(p.Duration)
			p.SourceID = strings.TrimSpace(p.SourceID)
			return nil
		}, now)
}
