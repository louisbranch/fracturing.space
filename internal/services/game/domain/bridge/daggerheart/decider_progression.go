package daggerheart

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideLevelUpApply(cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFunc(cmd, EventTypeLevelUpApplied, "character",
		func(p *LevelUpApplyPayload) string { return strings.TrimSpace(p.CharacterID) },
		func(p *LevelUpApplyPayload, _ func() time.Time) *command.Rejection {
			p.CharacterID = strings.TrimSpace(p.CharacterID)
			p.NewDomainCardID = strings.TrimSpace(p.NewDomainCardID)
			if p.NewDomainCardID != "" && p.NewDomainCardLevel < 1 {
				return &command.Rejection{Code: "LEVEL_UP_INVALID", Message: "new_domain_card_level must be at least 1 when new_domain_card_id is set"}
			}
			if p.NewDomainCardID == "" && p.NewDomainCardLevel > 0 {
				return &command.Rejection{Code: "LEVEL_UP_INVALID", Message: "new_domain_card_id is required when new_domain_card_level is set"}
			}

			// Convert payload advancements to domain model.
			domainAdvs := make([]mechanics.Advancement, 0, len(p.Advancements))
			for _, a := range p.Advancements {
				da := mechanics.Advancement{
					Type:            mechanics.AdvancementType(a.Type),
					Trait:           a.Trait,
					DomainCardID:    a.DomainCardID,
					DomainCardLevel: a.DomainCardLevel,
					SubclassCardID:  a.SubclassCardID,
				}
				if a.Multiclass != nil {
					da.Multiclass = &mechanics.MulticlassChoice{
						SecondaryClassID:    a.Multiclass.SecondaryClassID,
						SecondarySubclassID: a.Multiclass.SecondarySubclassID,
						FoundationCardID:    a.Multiclass.FoundationCardID,
						SpellcastTrait:      a.Multiclass.SpellcastTrait,
						DomainID:            a.Multiclass.DomainID,
					}
				}
				domainAdvs = append(domainAdvs, da)
			}

			req := mechanics.LevelUpRequest{
				CharacterID:  p.CharacterID,
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
		}, now)
}
