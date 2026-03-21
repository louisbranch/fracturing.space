package decider

import (
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/normalize"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func decideCompanionExperienceBegin(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeCompanionExperienceBegun, "character",
		func(p *payload.CompanionExperienceBeginPayload) string {
			return normalize.ID(p.CharacterID).String()
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p *payload.CompanionExperienceBeginPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = normalize.ID(p.ActorCharacterID)
			p.CharacterID = normalize.ID(p.CharacterID)
			p.ExperienceID = normalize.String(p.ExperienceID)
			if p.ActorCharacterID == "" {
				return &command.Rejection{Code: "COMPANION_ACTOR_REQUIRED", Message: "actor character id is required"}
			}
			if p.CharacterID == "" {
				return &command.Rejection{Code: "COMPANION_TARGET_REQUIRED", Message: "character id is required"}
			}
			if p.ExperienceID == "" {
				return &command.Rejection{Code: "COMPANION_EXPERIENCE_REQUIRED", Message: "experience id is required"}
			}
			if p.CompanionStateBefore != nil && p.CompanionStateAfter != nil &&
				p.CompanionStateBefore.Normalized() == p.CompanionStateAfter.Normalized() {
				return &command.Rejection{Code: rejectionCodeCharacterStatePatchNoMutation, Message: "companion begin is unchanged"}
			}
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.CompanionExperienceBeginPayload) payload.CompanionExperienceBegunPayload {
			return payload.CompanionExperienceBegunPayload{
				CharacterID:    p.CharacterID,
				ExperienceID:   p.ExperienceID,
				CompanionState: companionStatePtrValue(p.CompanionStateAfter),
				Source:         "companion.experience.begin",
			}
		},
		now)
}

func decideCompanionReturn(snapshotState daggerheartstate.SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		payload.EventTypeCompanionReturned, "character",
		func(p *payload.CompanionReturnPayload) string { return normalize.ID(p.CharacterID).String() },
		func(_ daggerheartstate.SnapshotState, _ bool, p *payload.CompanionReturnPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = normalize.ID(p.ActorCharacterID)
			p.CharacterID = normalize.ID(p.CharacterID)
			p.Resolution = normalize.String(p.Resolution)
			if p.ActorCharacterID == "" {
				return &command.Rejection{Code: "COMPANION_ACTOR_REQUIRED", Message: "actor character id is required"}
			}
			if p.CharacterID == "" {
				return &command.Rejection{Code: "COMPANION_TARGET_REQUIRED", Message: "character id is required"}
			}
			if p.Resolution == "" {
				return &command.Rejection{Code: "COMPANION_RETURN_RESOLUTION_REQUIRED", Message: "resolution is required"}
			}
			if !hasIntFieldChange(p.StressBefore, p.StressAfter) &&
				((p.CompanionStateBefore == nil && p.CompanionStateAfter == nil) ||
					(p.CompanionStateBefore != nil && p.CompanionStateAfter != nil &&
						p.CompanionStateBefore.Normalized() == p.CompanionStateAfter.Normalized())) {
				return &command.Rejection{Code: rejectionCodeCharacterStatePatchNoMutation, Message: "companion return is unchanged"}
			}
			return nil
		},
		func(_ daggerheartstate.SnapshotState, _ bool, p payload.CompanionReturnPayload) payload.CompanionReturnedPayload {
			return payload.CompanionReturnedPayload{
				CharacterID:    p.CharacterID,
				Resolution:     p.Resolution,
				Stress:         p.StressAfter,
				CompanionState: companionStatePtrValue(p.CompanionStateAfter),
				Source:         "companion.return",
			}
		},
		now)
}

func companionStatePtrValue(value *daggerheartstate.CharacterCompanionState) *daggerheartstate.CharacterCompanionState {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &normalized
}
