package decider

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
)

func decideBeastformTransform(snapshotState snapstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeBeastformTransformed, "character",
		func(p *payload.BeastformTransformPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(s snapstate.SnapshotState, hasState bool, p *payload.BeastformTransformPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = ids.CharacterID(strings.TrimSpace(p.ActorCharacterID.String()))
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.BeastformID = strings.TrimSpace(p.BeastformID)
			p.EvolutionTrait = strings.TrimSpace(p.EvolutionTrait)
			if p.ActorCharacterID == "" {
				return &command.Rejection{Code: "BEASTFORM_ACTOR_REQUIRED", Message: "actor character id is required"}
			}
			if p.CharacterID == "" {
				return &command.Rejection{Code: "BEASTFORM_TARGET_REQUIRED", Message: "character id is required"}
			}
			if p.BeastformID == "" {
				return &command.Rejection{Code: "BEASTFORM_ID_REQUIRED", Message: "beastform id is required"}
			}
			if hasState && isCharacterStatePatchNoMutation(s, payload.CharacterStatePatchPayload{
				CharacterID:      p.CharacterID,
				StressBefore:     p.StressBefore,
				StressAfter:      p.StressAfter,
				HopeBefore:       p.HopeBefore,
				HopeAfter:        p.HopeAfter,
				ClassStateBefore: p.ClassStateBefore,
				ClassStateAfter:  p.ClassStateAfter,
			}) {
				return &command.Rejection{Code: rejectionCodeCharacterStatePatchNoMutation, Message: "beastform transform is unchanged"}
			}
			return nil
		},
		func(_ snapstate.SnapshotState, _ bool, p payload.BeastformTransformPayload) payload.BeastformTransformedPayload {
			var active *snapstate.CharacterActiveBeastformState
			if p.ClassStateAfter != nil {
				active = snapstate.NormalizedActiveBeastformPtr(p.ClassStateAfter.ActiveBeastform)
			}
			return payload.BeastformTransformedPayload{
				CharacterID:     p.CharacterID,
				BeastformID:     p.BeastformID,
				Stress:          p.StressAfter,
				Hope:            p.HopeAfter,
				ActiveBeastform: active,
				Source:          "beastform.transform",
			}
		},
		now)
}

func decideBeastformDrop(snapshotState snapstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeBeastformDropped, "character",
		func(p *payload.BeastformDropPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(s snapstate.SnapshotState, hasState bool, p *payload.BeastformDropPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = ids.CharacterID(strings.TrimSpace(p.ActorCharacterID.String()))
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.BeastformID = strings.TrimSpace(p.BeastformID)
			p.Source = strings.TrimSpace(p.Source)
			if p.ActorCharacterID == "" {
				return &command.Rejection{Code: "BEASTFORM_ACTOR_REQUIRED", Message: "actor character id is required"}
			}
			if p.CharacterID == "" {
				return &command.Rejection{Code: "BEASTFORM_TARGET_REQUIRED", Message: "character id is required"}
			}
			if p.BeastformID == "" {
				return &command.Rejection{Code: "BEASTFORM_ID_REQUIRED", Message: "beastform id is required"}
			}
			if hasState && isCharacterStatePatchNoMutation(s, payload.CharacterStatePatchPayload{
				CharacterID:      p.CharacterID,
				ClassStateBefore: p.ClassStateBefore,
				ClassStateAfter:  p.ClassStateAfter,
			}) {
				return &command.Rejection{Code: rejectionCodeCharacterStatePatchNoMutation, Message: "beastform drop is unchanged"}
			}
			return nil
		},
		func(_ snapstate.SnapshotState, _ bool, p payload.BeastformDropPayload) payload.BeastformDroppedPayload {
			source := strings.TrimSpace(p.Source)
			if source == "" {
				source = "beastform.drop"
			}
			return payload.BeastformDroppedPayload{
				CharacterID: p.CharacterID,
				BeastformID: p.BeastformID,
				Source:      source,
			}
		},
		now)
}
