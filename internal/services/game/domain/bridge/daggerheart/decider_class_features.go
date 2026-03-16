package daggerheart

import (
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideClassFeatureApply(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncMulti(cmd, snapshotState, hasSnapshot,
		func(s SnapshotState, hasState bool, p *ClassFeatureApplyPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = ids.CharacterID(strings.TrimSpace(p.ActorCharacterID.String()))
			p.Feature = strings.TrimSpace(p.Feature)
			if p.ActorCharacterID == "" {
				return &command.Rejection{Code: "CLASS_FEATURE_ACTOR_REQUIRED", Message: "actor character id is required"}
			}
			if p.Feature == "" {
				return &command.Rejection{Code: "CLASS_FEATURE_NAME_REQUIRED", Message: "feature is required"}
			}
			if len(p.Targets) == 0 {
				return &command.Rejection{Code: "CLASS_FEATURE_TARGET_REQUIRED", Message: "class feature requires at least one target"}
			}
			if hasState {
				for _, targetPatch := range p.Targets {
					targetPatch.CharacterID = ids.CharacterID(strings.TrimSpace(targetPatch.CharacterID.String()))
					target, ok := snapshotCharacterState(s, targetPatch.CharacterID)
					if ok {
						if targetPatch.HPAfter != nil && target.HP != derefInt(targetPatch.HPBefore, target.HP) {
							return &command.Rejection{Code: rejectionCodeDamageBeforeMismatch, Message: "class feature hp_before does not match current state"}
						}
						if targetPatch.HopeAfter != nil && target.Hope != derefInt(targetPatch.HopeBefore, target.Hope) {
							return &command.Rejection{Code: rejectionCodeDamageBeforeMismatch, Message: "class feature hope_before does not match current state"}
						}
						if targetPatch.ArmorAfter != nil && target.Armor != derefInt(targetPatch.ArmorBefore, target.Armor) {
							return &command.Rejection{Code: rejectionCodeDamageBeforeMismatch, Message: "class feature armor_before does not match current state"}
						}
					}
					if isCharacterStatePatchNoMutation(s, CharacterStatePatchPayload{
						CharacterID:      targetPatch.CharacterID,
						HPBefore:         targetPatch.HPBefore,
						HPAfter:          targetPatch.HPAfter,
						ClassStateBefore: targetPatch.ClassStateBefore,
						ClassStateAfter:  targetPatch.ClassStateAfter,
						HopeBefore:       targetPatch.HopeBefore,
						HopeAfter:        targetPatch.HopeAfter,
						ArmorBefore:      targetPatch.ArmorBefore,
						ArmorAfter:       targetPatch.ArmorAfter,
					}) {
						return &command.Rejection{Code: rejectionCodeCharacterStatePatchNoMutation, Message: "class feature is unchanged"}
					}
				}
			}
			return nil
		},
		func(_ SnapshotState, _ bool, p ClassFeatureApplyPayload, _ func() time.Time) ([]module.EventSpec, error) {
			source := fmt.Sprintf("class_feature:%s:%s", p.Feature, strings.TrimSpace(p.ActorCharacterID.String()))
			specs := make([]module.EventSpec, 0, len(p.Targets))
			for _, targetPatch := range p.Targets {
				specs = append(specs, module.EventSpec{
					Type:       EventTypeCharacterStatePatched,
					EntityType: "character",
					EntityID:   strings.TrimSpace(targetPatch.CharacterID.String()),
					Payload: CharacterStatePatchedPayload{
						CharacterID: targetPatch.CharacterID,
						Source:      source,
						HP:          targetPatch.HPAfter,
						Hope:        targetPatch.HopeAfter,
						Armor:       targetPatch.ArmorAfter,
						ClassState:  normalizedClassStatePtr(targetPatch.ClassStateAfter),
					},
				})
			}
			return specs, nil
		},
		now)
}

func derefInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
