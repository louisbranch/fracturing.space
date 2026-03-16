package daggerheart

import (
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
)

func decideSubclassFeatureApply(snapshotState SnapshotState, hasSnapshot bool, cmd command.Command, now func() time.Time) command.Decision {
	return module.DecideFuncMulti(cmd, snapshotState, hasSnapshot,
		func(s SnapshotState, hasState bool, p *SubclassFeatureApplyPayload, _ func() time.Time) *command.Rejection {
			p.ActorCharacterID = ids.CharacterID(strings.TrimSpace(p.ActorCharacterID.String()))
			p.Feature = strings.TrimSpace(p.Feature)
			if p.ActorCharacterID == "" {
				return &command.Rejection{Code: "SUBCLASS_FEATURE_ACTOR_REQUIRED", Message: "actor character id is required"}
			}
			if p.Feature == "" {
				return &command.Rejection{Code: "SUBCLASS_FEATURE_NAME_REQUIRED", Message: "feature is required"}
			}
			if len(p.Targets) == 0 && len(p.CharacterConditionTargets) == 0 && len(p.AdversaryConditionTargets) == 0 {
				return &command.Rejection{Code: "SUBCLASS_FEATURE_TARGET_REQUIRED", Message: "subclass feature requires at least one consequence"}
			}
			hasMutation := len(p.CharacterConditionTargets) > 0 || len(p.AdversaryConditionTargets) > 0
			if hasState {
				for _, targetPatch := range p.Targets {
					targetPatch.CharacterID = ids.CharacterID(strings.TrimSpace(targetPatch.CharacterID.String()))
					target, ok := snapshotCharacterState(s, targetPatch.CharacterID)
					if ok {
						if targetPatch.HPAfter != nil && target.HP != derefInt(targetPatch.HPBefore, target.HP) {
							return &command.Rejection{Code: rejectionCodeDamageBeforeMismatch, Message: "subclass feature hp_before does not match current state"}
						}
						if targetPatch.HopeAfter != nil && target.Hope != derefInt(targetPatch.HopeBefore, target.Hope) {
							return &command.Rejection{Code: rejectionCodeDamageBeforeMismatch, Message: "subclass feature hope_before does not match current state"}
						}
						if targetPatch.StressAfter != nil && target.Stress != derefInt(targetPatch.StressBefore, target.Stress) {
							return &command.Rejection{Code: rejectionCodeDamageBeforeMismatch, Message: "subclass feature stress_before does not match current state"}
						}
						if targetPatch.ArmorAfter != nil && target.Armor != derefInt(targetPatch.ArmorBefore, target.Armor) {
							return &command.Rejection{Code: rejectionCodeDamageBeforeMismatch, Message: "subclass feature armor_before does not match current state"}
						}
					}
					if isCharacterStatePatchNoMutation(s, CharacterStatePatchPayload{
						CharacterID:         targetPatch.CharacterID,
						HPBefore:            targetPatch.HPBefore,
						HPAfter:             targetPatch.HPAfter,
						HopeBefore:          targetPatch.HopeBefore,
						HopeAfter:           targetPatch.HopeAfter,
						StressBefore:        targetPatch.StressBefore,
						StressAfter:         targetPatch.StressAfter,
						ArmorBefore:         targetPatch.ArmorBefore,
						ArmorAfter:          targetPatch.ArmorAfter,
						ClassStateBefore:    targetPatch.ClassStateBefore,
						ClassStateAfter:     targetPatch.ClassStateAfter,
						SubclassStateBefore: targetPatch.SubclassStateBefore,
						SubclassStateAfter:  targetPatch.SubclassStateAfter,
					}) {
						continue
					}
					hasMutation = true
				}
			}
			if !hasMutation {
				return &command.Rejection{Code: rejectionCodeCharacterStatePatchNoMutation, Message: "subclass feature is unchanged"}
			}
			return nil
		},
		func(_ SnapshotState, _ bool, p SubclassFeatureApplyPayload, _ func() time.Time) ([]module.EventSpec, error) {
			source := fmt.Sprintf("subclass_feature:%s:%s", p.Feature, strings.TrimSpace(p.ActorCharacterID.String()))
			specs := make([]module.EventSpec, 0, len(p.Targets)+len(p.CharacterConditionTargets)+len(p.AdversaryConditionTargets))
			for _, targetPatch := range p.Targets {
				specs = append(specs, module.EventSpec{
					Type:       EventTypeCharacterStatePatched,
					EntityType: "character",
					EntityID:   strings.TrimSpace(targetPatch.CharacterID.String()),
					Payload: CharacterStatePatchedPayload{
						CharacterID:   targetPatch.CharacterID,
						Source:        source,
						HP:            targetPatch.HPAfter,
						Hope:          targetPatch.HopeAfter,
						Stress:        targetPatch.StressAfter,
						Armor:         targetPatch.ArmorAfter,
						ClassState:    normalizedClassStatePtr(targetPatch.ClassStateAfter),
						SubclassState: normalizedSubclassStatePtr(targetPatch.SubclassStateAfter),
					},
				})
			}
			for _, targetPatch := range p.CharacterConditionTargets {
				eventSource := strings.TrimSpace(targetPatch.Source)
				if eventSource == "" {
					eventSource = source
				}
				specs = append(specs, module.EventSpec{
					Type:       EventTypeConditionChanged,
					EntityType: "character",
					EntityID:   strings.TrimSpace(targetPatch.CharacterID.String()),
					Payload: ConditionChangedPayload{
						CharacterID: targetPatch.CharacterID,
						Conditions:  targetPatch.ConditionsAfter,
						Added:       targetPatch.Added,
						Removed:     targetPatch.Removed,
						Source:      eventSource,
						RollSeq:     targetPatch.RollSeq,
					},
				})
			}
			for _, targetPatch := range p.AdversaryConditionTargets {
				eventSource := strings.TrimSpace(targetPatch.Source)
				if eventSource == "" {
					eventSource = source
				}
				specs = append(specs, module.EventSpec{
					Type:       EventTypeAdversaryConditionChanged,
					EntityType: "adversary",
					EntityID:   strings.TrimSpace(targetPatch.AdversaryID.String()),
					Payload: AdversaryConditionChangedPayload{
						AdversaryID: targetPatch.AdversaryID,
						Conditions:  targetPatch.ConditionsAfter,
						Added:       targetPatch.Added,
						Removed:     targetPatch.Removed,
						Source:      eventSource,
						RollSeq:     targetPatch.RollSeq,
					},
				})
			}
			return specs, nil
		},
		now)
}
