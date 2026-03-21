package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/payload"

// --- Payload type aliases ---

type GMFearSetPayload = payload.GMFearSetPayload
type GMFearChangedPayload = payload.GMFearChangedPayload
type GMMoveTarget = payload.GMMoveTarget
type GMMoveApplyPayload = payload.GMMoveApplyPayload
type GMMoveAppliedPayload = payload.GMMoveAppliedPayload
type CharacterStatePatchPayload = payload.CharacterStatePatchPayload
type CharacterStatePatchedPayload = payload.CharacterStatePatchedPayload
type ClassFeatureTargetPatchPayload = payload.ClassFeatureTargetPatchPayload
type ClassFeatureApplyPayload = payload.ClassFeatureApplyPayload
type SubclassFeatureTargetPatchPayload = payload.SubclassFeatureTargetPatchPayload
type SubclassFeatureApplyPayload = payload.SubclassFeatureApplyPayload
type BeastformTransformPayload = payload.BeastformTransformPayload
type BeastformDropPayload = payload.BeastformDropPayload
type BeastformTransformedPayload = payload.BeastformTransformedPayload
type BeastformDroppedPayload = payload.BeastformDroppedPayload
type CompanionExperienceBeginPayload = payload.CompanionExperienceBeginPayload
type CompanionReturnPayload = payload.CompanionReturnPayload
type CompanionExperienceBegunPayload = payload.CompanionExperienceBegunPayload
type CompanionReturnedPayload = payload.CompanionReturnedPayload
type ConditionChangePayload = payload.ConditionChangePayload
type ConditionChangedPayload = payload.ConditionChangedPayload
type HopeSpendPayload = payload.HopeSpendPayload
type StressSpendPayload = payload.StressSpendPayload
type LoadoutSwapPayload = payload.LoadoutSwapPayload
type LoadoutSwappedPayload = payload.LoadoutSwappedPayload
type RestTakePayload = payload.RestTakePayload
type RestTakenPayload = payload.RestTakenPayload
type CharacterTemporaryArmorApplyPayload = payload.CharacterTemporaryArmorApplyPayload
type CharacterTemporaryArmorAppliedPayload = payload.CharacterTemporaryArmorAppliedPayload
type RollRngInfo = payload.RollRngInfo
type CountdownCreatePayload = payload.CountdownCreatePayload
type CountdownCreatedPayload = payload.CountdownCreatedPayload
type CountdownUpdatePayload = payload.CountdownUpdatePayload
type CountdownUpdatedPayload = payload.CountdownUpdatedPayload
type CountdownDeletePayload = payload.CountdownDeletePayload
type CountdownDeletedPayload = payload.CountdownDeletedPayload
type DamageApplyPayload = payload.DamageApplyPayload
type DamageAppliedPayload = payload.DamageAppliedPayload
type MultiTargetDamageApplyPayload = payload.MultiTargetDamageApplyPayload
type AdversaryDamageApplyPayload = payload.AdversaryDamageApplyPayload
type AdversaryDamageAppliedPayload = payload.AdversaryDamageAppliedPayload
type DowntimeMoveAppliedPayload = payload.DowntimeMoveAppliedPayload
type AdversaryConditionChangePayload = payload.AdversaryConditionChangePayload
type AdversaryConditionChangedPayload = payload.AdversaryConditionChangedPayload
type AdversaryCreatePayload = payload.AdversaryCreatePayload
type AdversaryCreatedPayload = payload.AdversaryCreatedPayload
type AdversaryUpdatePayload = payload.AdversaryUpdatePayload
type AdversaryFeatureApplyPayload = payload.AdversaryFeatureApplyPayload
type AdversaryUpdatedPayload = payload.AdversaryUpdatedPayload
type AdversaryDeletePayload = payload.AdversaryDeletePayload
type AdversaryDeletedPayload = payload.AdversaryDeletedPayload
type EnvironmentEntityCreatePayload = payload.EnvironmentEntityCreatePayload
type EnvironmentEntityCreatedPayload = payload.EnvironmentEntityCreatedPayload
type EnvironmentEntityUpdatePayload = payload.EnvironmentEntityUpdatePayload
type EnvironmentEntityUpdatedPayload = payload.EnvironmentEntityUpdatedPayload
type EnvironmentEntityDeletePayload = payload.EnvironmentEntityDeletePayload
type EnvironmentEntityDeletedPayload = payload.EnvironmentEntityDeletedPayload
type LevelUpApplyPayload = payload.LevelUpApplyPayload
type LevelUpAdvancementPayload = payload.LevelUpAdvancementPayload
type LevelUpRewardPayload = payload.LevelUpRewardPayload
type LevelUpMulticlassPayload = payload.LevelUpMulticlassPayload
type LevelUpAppliedPayload = payload.LevelUpAppliedPayload
type GoldUpdatePayload = payload.GoldUpdatePayload
type GoldUpdatedPayload = payload.GoldUpdatedPayload
type DomainCardAcquirePayload = payload.DomainCardAcquirePayload
type DomainCardAcquiredPayload = payload.DomainCardAcquiredPayload
type EquipmentSwapPayload = payload.EquipmentSwapPayload
type EquipmentSwappedPayload = payload.EquipmentSwappedPayload
type ConsumableUsePayload = payload.ConsumableUsePayload
type ConsumableUsedPayload = payload.ConsumableUsedPayload
type ConsumableAcquirePayload = payload.ConsumableAcquirePayload
type ConsumableAcquiredPayload = payload.ConsumableAcquiredPayload
type StatModifierChangePayload = payload.StatModifierChangePayload
type StatModifierChangedPayload = payload.StatModifierChangedPayload

// --- Event type constant aliases ---

const (
	EventTypeDamageApplied                  = payload.EventTypeDamageApplied
	EventTypeRestTaken                      = payload.EventTypeRestTaken
	EventTypeDowntimeMoveApplied            = payload.EventTypeDowntimeMoveApplied
	EventTypeLoadoutSwapped                 = payload.EventTypeLoadoutSwapped
	EventTypeCharacterProfileReplaced       = payload.EventTypeCharacterProfileReplaced
	EventTypeCharacterProfileDeleted        = payload.EventTypeCharacterProfileDeleted
	EventTypeCharacterStatePatched          = payload.EventTypeCharacterStatePatched
	EventTypeBeastformTransformed           = payload.EventTypeBeastformTransformed
	EventTypeBeastformDropped               = payload.EventTypeBeastformDropped
	EventTypeCompanionExperienceBegun       = payload.EventTypeCompanionExperienceBegun
	EventTypeCompanionReturned              = payload.EventTypeCompanionReturned
	EventTypeConditionChanged               = payload.EventTypeConditionChanged
	EventTypeGMMoveApplied                  = payload.EventTypeGMMoveApplied
	EventTypeGMFearChanged                  = payload.EventTypeGMFearChanged
	EventTypeCountdownCreated               = payload.EventTypeCountdownCreated
	EventTypeCountdownUpdated               = payload.EventTypeCountdownUpdated
	EventTypeCountdownDeleted               = payload.EventTypeCountdownDeleted
	EventTypeCharacterTemporaryArmorApplied = payload.EventTypeCharacterTemporaryArmorApplied
	EventTypeAdversaryCreated               = payload.EventTypeAdversaryCreated
	EventTypeAdversaryConditionChanged      = payload.EventTypeAdversaryConditionChanged
	EventTypeAdversaryDamageApplied         = payload.EventTypeAdversaryDamageApplied
	EventTypeAdversaryUpdated               = payload.EventTypeAdversaryUpdated
	EventTypeAdversaryDeleted               = payload.EventTypeAdversaryDeleted
	EventTypeEnvironmentEntityCreated       = payload.EventTypeEnvironmentEntityCreated
	EventTypeEnvironmentEntityUpdated       = payload.EventTypeEnvironmentEntityUpdated
	EventTypeEnvironmentEntityDeleted       = payload.EventTypeEnvironmentEntityDeleted
	EventTypeLevelUpApplied                 = payload.EventTypeLevelUpApplied
	EventTypeGoldUpdated                    = payload.EventTypeGoldUpdated
	EventTypeDomainCardAcquired             = payload.EventTypeDomainCardAcquired
	EventTypeEquipmentSwapped               = payload.EventTypeEquipmentSwapped
	EventTypeConsumableUsed                 = payload.EventTypeConsumableUsed
	EventTypeConsumableAcquired             = payload.EventTypeConsumableAcquired
	EventTypeStatModifierChanged            = payload.EventTypeStatModifierChanged
)
