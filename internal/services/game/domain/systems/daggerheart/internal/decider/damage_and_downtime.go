package decider

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func decideDamageApply(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeDamageApplied, "character",
		func(p *payload.DamageApplyPayload) string { return normalize.ID(p.CharacterID).String() },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.DamageApplyPayload, _ func() time.Time) *command.Rejection {
			if r := rejectArmorSpendLimit(p.ArmorSpent); r != nil {
				return r
			}
			if hasState {
				if character, ok := snapshotCharacterState(s, p.CharacterID); ok {
					if r := rejectDamageBeforeMismatch(p.HpBefore, character.HP, p.ArmorBefore, character.Armor, rejectionCodeDamageBeforeMismatch, "damage before does not match current state"); r != nil {
						return r
					}
				}
			}
			p.CharacterID = normalize.ID(p.CharacterID)
			p.DamageType = normalize.String(p.DamageType)
			p.Source = normalize.String(p.Source)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.DamageApplyPayload) payload.DamageAppliedPayload {
			return characterDamageAppliedPayload(p)
		},
		now)
}

// decideMultiTargetDamageApply handles batch damage across multiple characters
// atomically. Each target entry produces one damage_applied event. All events
// are batch-appended in a single decision, avoiding the sequential failure
// window of N individual commands.
func decideMultiTargetDamageApply(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncMulti(cmd, snapshotState, hasSnapshot,
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.MultiTargetDamageApplyPayload, _ func() time.Time) *command.Rejection {
			if len(p.Targets) == 0 {
				return &command.Rejection{
					Code:    "MULTI_TARGET_NO_TARGETS",
					Message: "multi-target damage requires at least one target",
				}
			}
			for i := range p.Targets {
				t := &p.Targets[i]
				if r := rejectArmorSpendLimit(t.ArmorSpent); r != nil {
					return r
				}
				if hasState {
					if character, ok := snapshotCharacterState(s, t.CharacterID); ok {
						if r := rejectDamageBeforeMismatch(t.HpBefore, character.HP, t.ArmorBefore, character.Armor, rejectionCodeDamageBeforeMismatch, "damage before does not match current state"); r != nil {
							return r
						}
					}
				}
				t.CharacterID = normalize.ID(t.CharacterID)
				t.DamageType = normalize.String(t.DamageType)
				t.Source = normalize.String(t.Source)
			}
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.MultiTargetDamageApplyPayload, _ func() time.Time) ([]module.EventSpec, error) {
			specs := make([]module.EventSpec, 0, len(p.Targets))
			for _, t := range p.Targets {
				specs = append(specs, module.EventSpec{
					Type:       payload.EventTypeDamageApplied,
					EntityType: "character",
					EntityID:   t.CharacterID.String(),
					Payload:    characterDamageAppliedPayload(t),
				})
			}
			return specs, nil
		}, now)
}

func decideAdversaryDamageApply(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeAdversaryDamageApplied, "adversary",
		func(p *payload.AdversaryDamageApplyPayload) string { return normalize.ID(p.AdversaryID).String() },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.AdversaryDamageApplyPayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if adversary, ok := snapshotAdversaryState(s, p.AdversaryID); ok {
					if r := rejectDamageBeforeMismatch(p.HpBefore, adversary.HP, p.ArmorBefore, adversary.Armor, rejectionCodeAdversaryDamageBeforeMismatch, "adversary damage before does not match current state"); r != nil {
						return r
					}
				}
			}
			p.AdversaryID = normalize.ID(p.AdversaryID)
			p.DamageType = normalize.String(p.DamageType)
			p.Source = normalize.String(p.Source)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.AdversaryDamageApplyPayload) payload.AdversaryDamageAppliedPayload {
			return payload.AdversaryDamageAppliedPayload{
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
	return module.DecideFunc(cmd, payload.EventTypeCharacterTemporaryArmorApplied, "character",
		func(p *payload.CharacterTemporaryArmorApplyPayload) string {
			return normalize.ID(p.CharacterID).String()
		},
		func(p *payload.CharacterTemporaryArmorApplyPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = normalize.ID(p.CharacterID)
			p.Source = normalize.String(p.Source)
			p.Duration = normalize.String(p.Duration)
			p.SourceID = normalize.String(p.SourceID)
			return nil
		}, now)
}
