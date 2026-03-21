package reducer

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
)

type CharacterStatePatch struct {
	HPAfter        *int
	HopeAfter      *int
	HopeMaxAfter   *int
	StressAfter    *int
	ArmorAfter     *int
	LifeStateAfter *string
}

type TemporaryArmorPatch struct {
	Source   string
	Duration string
	SourceID string
	Amount   int
}

type RestCharacterPatch struct {
	HopeAfter   *int
	StressAfter *int
	ArmorAfter  *int
}

func ApplyCharacterStatePatch(state *mechanics.CharacterState, patch CharacterStatePatch) {
	if state == nil {
		return
	}
	if patch.HPAfter != nil {
		state.HP = *patch.HPAfter
	}
	if patch.HopeAfter != nil {
		state.Hope = *patch.HopeAfter
	}
	if patch.HopeMaxAfter != nil {
		state.HopeMax = *patch.HopeMaxAfter
	}
	if patch.StressAfter != nil {
		state.Stress = *patch.StressAfter
	}
	if patch.ArmorAfter != nil {
		state.Armor = *patch.ArmorAfter
	}
	if patch.LifeStateAfter != nil {
		state.LifeState = *patch.LifeStateAfter
	}
}

func ApplyConditionPatch(state *mechanics.CharacterState, conditions []string) {
	if state == nil {
		return
	}
	state.Conditions = append([]string(nil), conditions...)
}

func ApplyLoadoutSwap(state *mechanics.CharacterState, stressAfter *int) {
	if state == nil || stressAfter == nil {
		return
	}
	state.Stress = *stressAfter
}

func ApplyTemporaryArmor(state *mechanics.CharacterState, patch TemporaryArmorPatch) {
	if state == nil {
		return
	}
	state.ApplyTemporaryArmor(mechanics.TemporaryArmorBucket{
		Source:   strings.TrimSpace(patch.Source),
		Duration: strings.TrimSpace(patch.Duration),
		SourceID: strings.TrimSpace(patch.SourceID),
		Amount:   patch.Amount,
	})
	if strings.TrimSpace(state.LifeState) == "" {
		state.LifeState = mechanics.LifeStateAlive
	}
}

func ApplyRestPatch(state *mechanics.CharacterState, patch RestCharacterPatch) {
	if state == nil {
		return
	}
	if patch.HopeAfter != nil {
		state.Hope = *patch.HopeAfter
	}
	if patch.StressAfter != nil {
		state.Stress = *patch.StressAfter
	}
	if patch.ArmorAfter != nil {
		state.Armor = *patch.ArmorAfter
	}
}

func ClearRestTemporaryArmor(state *mechanics.CharacterState, clearShortRest bool, clearLongRest bool) {
	if state == nil {
		return
	}
	if clearShortRest {
		state.ClearTemporaryArmorByDuration("short_rest")
	}
	if clearLongRest {
		state.ClearTemporaryArmorByDuration("long_rest")
	}
	state.SetArmor(state.ResourceCap(mechanics.ResourceArmor))
}

func ApplyDamage(state *mechanics.CharacterState, hpAfter *int, armorAfter *int) {
	if state == nil {
		return
	}
	if hpAfter != nil {
		state.HP = *hpAfter
	}
	if armorAfter != nil {
		state.Armor = *armorAfter
	}
}

func ApplyDowntimeMove(state *mechanics.CharacterState, move string, hopeAfter, stressAfter, armorAfter *int) {
	if state == nil {
		return
	}
	if strings.TrimSpace(move) == "repair_all_armor" {
		state.ClearTemporaryArmorByDuration("short_rest")
		if armorAfter == nil {
			armor := state.ResourceCap(mechanics.ResourceArmor)
			armorAfter = &armor
		}
	}
	if hopeAfter != nil {
		state.Hope = *hopeAfter
	}
	if stressAfter != nil {
		state.Stress = *stressAfter
	}
	if armorAfter != nil {
		state.Armor = *armorAfter
	}
}

func NormalizeAndValidateCharacterState(state *mechanics.CharacterState) error {
	if state == nil {
		return nil
	}
	if state.HP < mechanics.HPMin || state.HP > mechanics.HPMaxCap {
		return fmt.Errorf("character_state hp must be in range %d..%d", mechanics.HPMin, mechanics.HPMaxCap)
	}
	if state.HopeMax == 0 {
		state.HopeMax = mechanics.HopeMax
	}
	if state.HopeMax < mechanics.HopeMin || state.HopeMax > mechanics.HopeMax {
		return fmt.Errorf("character_state hope_max must be in range %d..%d", mechanics.HopeMin, mechanics.HopeMax)
	}
	if state.Hope < mechanics.HopeMin || state.Hope > state.HopeMax {
		return fmt.Errorf("character_state hope must be in range %d..%d", mechanics.HopeMin, state.HopeMax)
	}
	if state.Stress < mechanics.StressMin || state.Stress > mechanics.StressMaxCap {
		return fmt.Errorf("character_state stress must be in range %d..%d", mechanics.StressMin, mechanics.StressMaxCap)
	}
	if state.Armor < mechanics.ArmorMin || state.Armor > mechanics.ArmorMaxCap {
		return fmt.Errorf("character_state armor must be in range %d..%d", mechanics.ArmorMin, mechanics.ArmorMaxCap)
	}
	if strings.TrimSpace(state.LifeState) == "" {
		state.LifeState = mechanics.LifeStateAlive
	} else if _, err := mechanics.NormalizeLifeState(state.LifeState); err != nil {
		return fmt.Errorf("character_state life_state: %w", err)
	}
	if state.Hope > state.HopeMax {
		state.Hope = state.HopeMax
	}
	return nil
}
