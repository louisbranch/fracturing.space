package daggerheart

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideCompanionExperienceBegin(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeCompanionExperienceBegun, "character",
		func(p *CompanionExperienceBeginPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ SnapshotState, _ bool, p *CompanionExperienceBeginPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = ids.CharacterID(strings.TrimSpace(p.ActorCharacterID.String()))
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.ExperienceID = strings.TrimSpace(p.ExperienceID)
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
		func(_ SnapshotState, _ bool, p CompanionExperienceBeginPayload) CompanionExperienceBegunPayload {
			return CompanionExperienceBegunPayload{
				CharacterID:    p.CharacterID,
				ExperienceID:   p.ExperienceID,
				CompanionState: companionStatePtrValue(p.CompanionStateAfter),
				Source:         "companion.experience.begin",
			}
		},
		now)
}

func decideCompanionReturn(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncTransform(cmd, snapshotState, hasSnapshot,
		EventTypeCompanionReturned, "character",
		func(p *CompanionReturnPayload) string { return strings.TrimSpace(p.CharacterID.String()) },
		func(_ SnapshotState, _ bool, p *CompanionReturnPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = ids.CharacterID(strings.TrimSpace(p.ActorCharacterID.String()))
			p.CharacterID = ids.CharacterID(strings.TrimSpace(p.CharacterID.String()))
			p.Resolution = strings.TrimSpace(p.Resolution)
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
		func(_ SnapshotState, _ bool, p CompanionReturnPayload) CompanionReturnedPayload {
			return CompanionReturnedPayload{
				CharacterID:    p.CharacterID,
				Resolution:     p.Resolution,
				Stress:         p.StressAfter,
				CompanionState: companionStatePtrValue(p.CompanionStateAfter),
				Source:         "companion.return",
			}
		},
		now)
}

func companionStatePtrValue(value *CharacterCompanionState) *CharacterCompanionState {
	if value == nil {
		return nil
	}
	normalized := value.Normalized()
	return &normalized
}
