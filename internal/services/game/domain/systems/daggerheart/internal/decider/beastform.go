package decider

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func decideBeastformTransform(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeBeastformTransformed, "character",
		func(p *payload.BeastformTransformPayload) string { return normalize.ID(p.CharacterID).String() },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.BeastformTransformPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = normalize.ID(p.ActorCharacterID)
			p.CharacterID = normalize.ID(p.CharacterID)
			p.BeastformID = normalize.String(p.BeastformID)
			p.EvolutionTrait = normalize.String(p.EvolutionTrait)
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
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.BeastformTransformPayload) payload.BeastformTransformedPayload {
			var active *daggerheartstate.CharacterActiveBeastformState
			if p.ClassStateAfter != nil {
				active = daggerheartstate.NormalizedActiveBeastformPtr(p.ClassStateAfter.ActiveBeastform)
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

func decideBeastformDrop(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeBeastformDropped, "character",
		func(p *payload.BeastformDropPayload) string { return normalize.ID(p.CharacterID).String() },
		func(s daggerheartstate.SnapshotState, hasState bool, p *payload.BeastformDropPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = normalize.ID(p.ActorCharacterID)
			p.CharacterID = normalize.ID(p.CharacterID)
			p.BeastformID = normalize.String(p.BeastformID)
			p.Source = normalize.String(p.Source)
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
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.BeastformDropPayload) payload.BeastformDroppedPayload {
			source := normalize.String(p.Source)
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
