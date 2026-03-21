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

func decideStatModifierChange(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeStatModifierChanged, "character",
		func(p *payload.StatModifierChangePayload) string {
			return strings.TrimSpace(p.CharacterID.String())
		},
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.StatModifierChangePayload, _ func() time.Time) *command.Rejection {
			if hasState {
				if isStatModifierChangeNoMutation(s, *p) {
					return &command.Rejection{
						Code:    rejectionCodeStatModifierChangeNoMutation,
						Message: "stat modifier change is unchanged",
					}
				}
			}
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.Source = strings.TrimSpace(p.Source)
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.StatModifierChangePayload) payload.StatModifierChangedPayload {
			return payload.StatModifierChangedPayload{
				CharacterID: p.CharacterID,
				Modifiers:   p.ModifiersAfter,
				Added:       p.Added,
				Removed:     p.Removed,
				Source:      p.Source,
			}
		},
		now)
}

func isStatModifierChangeNoMutation(snapshot daggerheartstate.SnapshotState, p payload.StatModifierChangePayload) bool {
	characterID := ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
	current := snapshot.CharacterStatModifiers[characterID]
	currentNorm, err := rules.NormalizeStatModifiers(current)
	if err != nil {
		return false
	}
	afterNorm, err := rules.NormalizeStatModifiers(p.ModifiersAfter)
	if err != nil {
		return false
	}
	return rules.StatModifiersEqual(currentNorm, afterNorm)
}
