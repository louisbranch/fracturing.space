package snapshottransport

import (
	"strings"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/charactertransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type daggerheartCharacterStatePatch struct {
	hp                   int
	hope                 int
	hopeMax              int
	stress               int
	stressMax            int
	armor                int
	lifeState            string
	conditionPatch       bool
	normalizedConditions []string
}

func buildDaggerheartCharacterStatePatch(
	current projectionstore.DaggerheartCharacterState,
	profile projectionstore.DaggerheartCharacterProfile,
	patch *daggerheartv1.DaggerheartCharacterState,
) (daggerheartCharacterStatePatch, error) {
	hp := int(patch.Hp)
	hpMax := profile.HpMax
	if hpMax == 0 {
		hpMax = 6
	}
	if hp < 0 || hp > hpMax {
		return daggerheartCharacterStatePatch{}, status.Errorf(codes.InvalidArgument, "hp %d exceeds range 0..%d", hp, hpMax)
	}

	hopeMax := int(patch.HopeMax)
	if hopeMax == 0 {
		hopeMax = current.HopeMax
		if hopeMax == 0 {
			hopeMax = mechanics.HopeMax
		}
	}
	if hopeMax < mechanics.HopeMin || hopeMax > mechanics.HopeMax {
		return daggerheartCharacterStatePatch{}, status.Errorf(
			codes.InvalidArgument,
			"hope_max %d exceeds range %d..%d",
			hopeMax,
			mechanics.HopeMin,
			mechanics.HopeMax,
		)
	}

	hope := int(patch.Hope)
	if hope < mechanics.HopeMin || hope > hopeMax {
		return daggerheartCharacterStatePatch{}, status.Errorf(codes.InvalidArgument, "hope %d exceeds range %d..%d", hope, mechanics.HopeMin, hopeMax)
	}

	stress := int(patch.Stress)
	stressMax := profile.StressMax
	if stressMax == 0 {
		stressMax = 6
	}
	if stress < 0 || stress > stressMax {
		return daggerheartCharacterStatePatch{}, status.Errorf(codes.InvalidArgument, "stress %d exceeds range 0..%d", stress, stressMax)
	}

	armor := int(patch.Armor)
	armorMax := profile.ArmorMax
	if armorMax < 0 {
		armorMax = 0
	}
	if armor < 0 || armor > armorMax {
		return daggerheartCharacterStatePatch{}, status.Errorf(codes.InvalidArgument, "armor %d exceeds range 0..%d", armor, armorMax)
	}

	var normalizedConditions []string
	conditionPatch := patch.ConditionStates != nil
	if conditionPatch {
		conditions, err := protoConditionCodes(patch.ConditionStates)
		if err != nil {
			return daggerheartCharacterStatePatch{}, status.Error(codes.InvalidArgument, err.Error())
		}
		normalizedConditions, err = rules.NormalizeConditions(conditions)
		if err != nil {
			return daggerheartCharacterStatePatch{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	lifeState := current.LifeState
	if patch.LifeState != daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED {
		var err error
		lifeState, err = charactertransport.DaggerheartLifeStateFromProto(patch.LifeState)
		if err != nil {
			return daggerheartCharacterStatePatch{}, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	if lifeState == "" {
		lifeState = daggerheartstate.LifeStateAlive
	}

	return daggerheartCharacterStatePatch{
		hp:                   hp,
		hope:                 hope,
		hopeMax:              hopeMax,
		stress:               stress,
		stressMax:            stressMax,
		armor:                armor,
		lifeState:            lifeState,
		conditionPatch:       conditionPatch,
		normalizedConditions: normalizedConditions,
	}, nil
}

func protoConditionCodes(states []*daggerheartv1.DaggerheartConditionState) ([]string, error) {
	if len(states) == 0 {
		return nil, nil
	}
	result := make([]string, 0, len(states))
	for _, state := range states {
		if state == nil {
			return nil, status.Error(codes.InvalidArgument, "condition state is required")
		}
		code := strings.TrimSpace(state.GetCode())
		if code == "" {
			code = strings.TrimSpace(strings.ToLower(strings.TrimPrefix(state.GetStandard().String(), "DAGGERHEART_CONDITION_")))
		}
		if code == "" {
			return nil, status.Error(codes.InvalidArgument, "condition code is required")
		}
		result = append(result, code)
	}
	return result, nil
}

func (p daggerheartCharacterStatePatch) stateUnchanged(current projectionstore.DaggerheartCharacterState) bool {
	lifeStateBefore := current.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheartstate.LifeStateAlive
	}
	return current.Hp == p.hp &&
		current.Hope == p.hope &&
		current.HopeMax == p.hopeMax &&
		current.Stress == p.stress &&
		current.Armor == p.armor &&
		lifeStateBefore == p.lifeState
}

func (p daggerheartCharacterStatePatch) payload(characterID string, current projectionstore.DaggerheartCharacterState) daggerheartpayload.CharacterStatePatchPayload {
	hpBefore := current.Hp
	hpAfter := p.hp
	hopeBefore := current.Hope
	hopeAfter := p.hope
	hopeMaxBefore := current.HopeMax
	hopeMaxAfter := p.hopeMax
	stressBefore := current.Stress
	stressAfter := p.stress
	armorBefore := current.Armor
	armorAfter := p.armor
	lifeStateBefore := current.LifeState
	if lifeStateBefore == "" {
		lifeStateBefore = daggerheartstate.LifeStateAlive
	}
	lifeStateAfter := p.lifeState

	return daggerheartpayload.CharacterStatePatchPayload{
		CharacterID:     ids.CharacterID(characterID),
		HPBefore:        &hpBefore,
		HPAfter:         &hpAfter,
		HopeBefore:      &hopeBefore,
		HopeAfter:       &hopeAfter,
		HopeMaxBefore:   &hopeMaxBefore,
		HopeMaxAfter:    &hopeMaxAfter,
		StressBefore:    &stressBefore,
		StressAfter:     &stressAfter,
		ArmorBefore:     &armorBefore,
		ArmorAfter:      &armorAfter,
		LifeStateBefore: &lifeStateBefore,
		LifeStateAfter:  &lifeStateAfter,
	}
}
