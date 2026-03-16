package statetransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/conditiontransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

// CharacterStateToProto maps stored Daggerheart character state into the gRPC
// response shape shared by damage, condition, and recovery wrappers.
func CharacterStateToProto(state projectionstore.DaggerheartCharacterState) *pb.DaggerheartCharacterState {
	temporaryArmorBuckets := make([]*pb.DaggerheartTemporaryArmorBucket, 0, len(state.TemporaryArmor))
	for _, bucket := range state.TemporaryArmor {
		temporaryArmorBuckets = append(temporaryArmorBuckets, &pb.DaggerheartTemporaryArmorBucket{
			Source:   bucket.Source,
			Duration: bucket.Duration,
			SourceId: bucket.SourceID,
			Amount:   int32(bucket.Amount),
		})
	}

	return &pb.DaggerheartCharacterState{
		Hp:                            int32(state.Hp),
		Hope:                          int32(state.Hope),
		HopeMax:                       int32(state.HopeMax),
		Stress:                        int32(state.Stress),
		Armor:                         int32(state.Armor),
		ConditionStates:               conditiontransport.ProjectionConditionStatesToProto(state.Conditions),
		TemporaryArmorBuckets:         temporaryArmorBuckets,
		LifeState:                     conditiontransport.LifeStateToProto(state.LifeState),
		ClassState:                    classStateToProto(state.ClassState),
		CompanionState:                companionStateToProto(state.CompanionState),
		SubclassState:                 subclassStateToProto(state.SubclassState),
		ImpenetrableUsedThisShortRest: state.ImpenetrableUsedThisShortRest,
	}
}

func classStateToProto(state projectionstore.DaggerheartClassState) *pb.DaggerheartClassState {
	return &pb.DaggerheartClassState{
		AttackBonusUntilRest:       int32(state.AttackBonusUntilRest),
		EvasionBonusUntilHitOrRest: int32(state.EvasionBonusUntilHitOrRest),
		DifficultyPenaltyUntilRest: int32(state.DifficultyPenaltyUntilRest),
		FocusTargetId:              state.FocusTargetID,
		ActiveBeastform:            activeBeastformToProto(state.ActiveBeastform),
		StrangePatternsNumber:      int32(state.StrangePatternsNumber),
		RallyDice:                  intsToInt32(state.RallyDice),
		PrayerDice:                 intsToInt32(state.PrayerDice),
		Unstoppable: &pb.DaggerheartUnstoppableState{
			Active:           state.Unstoppable.Active,
			CurrentValue:     int32(state.Unstoppable.CurrentValue),
			DieSides:         int32(state.Unstoppable.DieSides),
			UsedThisLongRest: state.Unstoppable.UsedThisLongRest,
		},
		ChannelRawPowerUsedThisLongRest: state.ChannelRawPowerUsedThisLongRest,
	}
}

func activeBeastformToProto(state *projectionstore.DaggerheartActiveBeastformState) *pb.DaggerheartActiveBeastformState {
	if state == nil {
		return nil
	}
	damageDice := make([]*pb.DaggerheartBeastformAttackDie, 0, len(state.DamageDice))
	for _, die := range state.DamageDice {
		damageDice = append(damageDice, &pb.DaggerheartBeastformAttackDie{
			Count: int32(die.Count),
			Sides: int32(die.Sides),
		})
	}
	return &pb.DaggerheartActiveBeastformState{
		BeastformId:            state.BeastformID,
		BaseTrait:              state.BaseTrait,
		AttackTrait:            state.AttackTrait,
		TraitBonus:             int32(state.TraitBonus),
		EvasionBonus:           int32(state.EvasionBonus),
		AttackRange:            state.AttackRange,
		DamageDice:             damageDice,
		DamageBonus:            int32(state.DamageBonus),
		DamageType:             state.DamageType,
		EvolutionTraitOverride: state.EvolutionTraitOverride,
		DropOnAnyHpMark:        state.DropOnAnyHPMark,
	}
}

func companionStateToProto(state *projectionstore.DaggerheartCompanionState) *pb.DaggerheartCompanionState {
	if state == nil {
		return nil
	}
	return &pb.DaggerheartCompanionState{
		Status:             state.Status,
		ActiveExperienceId: state.ActiveExperienceID,
	}
}

func subclassStateToProto(state *projectionstore.DaggerheartSubclassState) *pb.DaggerheartSubclassState {
	if state == nil {
		return nil
	}
	return &pb.DaggerheartSubclassState{
		BattleRitualUsedThisLongRest:           state.BattleRitualUsedThisLongRest,
		GiftedPerformerRelaxingSongUses:        int32(state.GiftedPerformerRelaxingSongUses),
		GiftedPerformerEpicSongUses:            int32(state.GiftedPerformerEpicSongUses),
		GiftedPerformerHeartbreakingSongUses:   int32(state.GiftedPerformerHeartbreakingSongUses),
		ContactsEverywhereUsesThisSession:      int32(state.ContactsEverywhereUsesThisSession),
		ContactsEverywhereActionDieBonus:       int32(state.ContactsEverywhereActionDieBonus),
		ContactsEverywhereDamageDiceBonusCount: int32(state.ContactsEverywhereDamageDiceBonusCount),
		SparingTouchUsesThisLongRest:           int32(state.SparingTouchUsesThisLongRest),
		ElementalistActionBonus:                int32(state.ElementalistActionBonus),
		ElementalistDamageBonus:                int32(state.ElementalistDamageBonus),
		TranscendenceActive:                    state.TranscendenceActive,
		TranscendenceTraitBonusTarget:          state.TranscendenceTraitBonusTarget,
		TranscendenceTraitBonusValue:           int32(state.TranscendenceTraitBonusValue),
		TranscendenceProficiencyBonus:          int32(state.TranscendenceProficiencyBonus),
		TranscendenceEvasionBonus:              int32(state.TranscendenceEvasionBonus),
		TranscendenceSevereThresholdBonus:      int32(state.TranscendenceSevereThresholdBonus),
		ClarityOfNatureUsedThisLongRest:        state.ClarityOfNatureUsedThisLongRest,
		ElementalChannel:                       state.ElementalChannel,
		NemesisTargetId:                        state.NemesisTargetID,
		RousingSpeechUsedThisLongRest:          state.RousingSpeechUsedThisLongRest,
		WardensProtectionUsedThisLongRest:      state.WardensProtectionUsedThisLongRest,
	}
}

func intsToInt32(values []int) []int32 {
	if len(values) == 0 {
		return nil
	}
	items := make([]int32, 0, len(values))
	for _, value := range values {
		items = append(items, int32(value))
	}
	return items
}

// OptionalInt32 preserves optional roll details in transport responses without
// teaching wrappers to repeat pointer conversion noise.
func OptionalInt32(value *int) *int32 {
	if value == nil {
		return nil
	}
	v := int32(*value)
	return &v
}
