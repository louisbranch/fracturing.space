package decider

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func decideLevelUpApply(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeLevelUpApplied, "character",
		func(p *payload.LevelUpApplyPayload) string { return normalize.ID(p.CharacterID).String() },
		func(_ daggerheartstate.SnapshotState, _ bool, p *payload.LevelUpApplyPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = normalize.ID(p.CharacterID)

			// Convert payload advancements to domain model.
			domainAdvs := make([]mechanics.Advancement, 0, len(p.Advancements))
			for _, a := range p.Advancements {
				da := mechanics.Advancement{
					Type:            mechanics.AdvancementType(a.Type),
					Trait:           a.Trait,
					DomainCardID:    a.DomainCardID,
					DomainCardLevel: a.DomainCardLevel,
				}
				if a.Multiclass != nil {
					da.Multiclass = &mechanics.MulticlassChoice{
						SecondaryClassID:    a.Multiclass.SecondaryClassID,
						SecondarySubclassID: a.Multiclass.SecondarySubclassID,
						SpellcastTrait:      a.Multiclass.SpellcastTrait,
						DomainID:            a.Multiclass.DomainID,
					}
				}
				domainAdvs = append(domainAdvs, da)
			}

			req := mechanics.LevelUpRequest{
				CharacterID:  p.CharacterID.String(),
				LevelBefore:  p.LevelBefore,
				LevelAfter:   p.LevelAfter,
				Advancements: domainAdvs,
				MarkedTraits: p.MarkedTraits,
			}

			result, err := mechanics.ValidateLevelUp(req)
			if err != nil {
				return &command.Rejection{
					Code:    "LEVEL_UP_INVALID",
					Message: err.Error(),
				}
			}

			// Populate derived fields on the payload so they flow into the event.
			p.Tier = result.Tier
			p.PreviousTier = result.PreviousTier
			p.IsTierEntry = result.IsTierEntry
			p.ClearMarks = result.ClearMarks
			p.MarkedAfter = result.MarkedTraits
			p.ThresholdDelta = result.ThresholdDelta
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.LevelUpApplyPayload) payload.LevelUpAppliedPayload {
			return payload.LevelUpAppliedPayload{
				CharacterID:                  p.CharacterID,
				Level:                        p.LevelAfter,
				Advancements:                 p.Advancements,
				Rewards:                      p.Rewards,
				SubclassTracksAfter:          p.SubclassTracksAfter,
				SubclassHpMaxDelta:           p.SubclassHpMaxDelta,
				SubclassStressMaxDelta:       p.SubclassStressMaxDelta,
				SubclassEvasionDelta:         p.SubclassEvasionDelta,
				SubclassMajorThresholdDelta:  p.SubclassMajorThresholdDelta,
				SubclassSevereThresholdDelta: p.SubclassSevereThresholdDelta,
				Tier:                         p.Tier,
				IsTierEntry:                  p.IsTierEntry,
				ClearMarks:                   p.ClearMarks,
				Marked:                       p.MarkedAfter,
				ThresholdDelta:               p.ThresholdDelta,
			}
		},
		now)
}
