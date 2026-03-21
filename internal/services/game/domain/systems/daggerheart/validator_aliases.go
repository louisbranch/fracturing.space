package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/validator"

// --- Unexported aliases for root-package test files ---

// Common helpers
var (
	requireTrimmedValue = validator.RequireTrimmedValue
	requirePositive     = validator.RequirePositive
	requireRange        = validator.RequireRange
)

// State validators
var (
	validateGMFearSetPayload                 = validator.ValidateGMFearSetPayload
	validateGMFearChangedPayload             = validator.ValidateGMFearChangedPayload
	validateGMMoveApplyPayload               = validator.ValidateGMMoveApplyPayload
	validateGMMoveAppliedPayload             = validator.ValidateGMMoveAppliedPayload
	validateGMMoveTarget                     = validator.ValidateGMMoveTarget
	validateCharacterProfileReplacePayload   = validator.ValidateCharacterProfileReplacePayload
	validateCharacterProfileReplacedPayload  = validator.ValidateCharacterProfileReplacedPayload
	validateCharacterProfileDeletePayload    = validator.ValidateCharacterProfileDeletePayload
	validateCharacterProfileDeletedPayload   = validator.ValidateCharacterProfileDeletedPayload
	validateCharacterStatePatchPayload       = validator.ValidateCharacterStatePatchPayload
	validateCharacterStatePatchedPayload     = validator.ValidateCharacterStatePatchedPayload
	validateClassFeatureApplyPayload         = validator.ValidateClassFeatureApplyPayload
	validateSubclassFeatureApplyPayload      = validator.ValidateSubclassFeatureApplyPayload
	validateBeastformTransformPayload        = validator.ValidateBeastformTransformPayload
	validateBeastformDropPayload             = validator.ValidateBeastformDropPayload
	validateBeastformTransformedPayload      = validator.ValidateBeastformTransformedPayload
	validateBeastformDroppedPayload          = validator.ValidateBeastformDroppedPayload
	validateCompanionExperienceBeginPayload  = validator.ValidateCompanionExperienceBeginPayload
	validateCompanionReturnPayload           = validator.ValidateCompanionReturnPayload
	validateCompanionExperienceBegunPayload  = validator.ValidateCompanionExperienceBegunPayload
	validateCompanionReturnedPayload         = validator.ValidateCompanionReturnedPayload
	validateHopeSpendPayload                 = validator.ValidateHopeSpendPayload
	validateStressSpendPayload               = validator.ValidateStressSpendPayload
	validateConditionChangePayload           = validator.ValidateConditionChangePayload
	validateConditionChangedPayload          = validator.ValidateConditionChangedPayload
	validateLoadoutSwapPayload               = validator.ValidateLoadoutSwapPayload
	validateLoadoutSwappedPayload            = validator.ValidateLoadoutSwappedPayload
	validateRestTakePayload                  = validator.ValidateRestTakePayload
	validateRestTakenPayload                 = validator.ValidateRestTakenPayload
	validateCountdownCreatePayload           = validator.ValidateCountdownCreatePayload
	validateCountdownCreatedPayload          = validator.ValidateCountdownCreatedPayload
	validateCountdownUpdatePayload           = validator.ValidateCountdownUpdatePayload
	validateCountdownUpdatedPayload          = validator.ValidateCountdownUpdatedPayload
	validateCountdownDeletePayload           = validator.ValidateCountdownDeletePayload
	validateCountdownDeletedPayload          = validator.ValidateCountdownDeletedPayload
	validateAdversaryConditionChangePayload  = validator.ValidateAdversaryConditionChangePayload
	validateAdversaryConditionChangedPayload = validator.ValidateAdversaryConditionChangedPayload
	validateConditionSetPayload              = validator.ValidateConditionSetPayload
	normalizeConditionStateListField         = validator.NormalizeConditionStateListField
	hasCharacterStateChange                  = validator.HasCharacterStateChange
	hasClassStateFieldChange                 = validator.HasClassStateFieldChange
	hasCompanionStateFieldChange             = validator.HasCompanionStateFieldChange
	hasSubclassStateFieldChange              = validator.HasSubclassStateFieldChange
	hasConditionListMutation                 = validator.HasConditionListMutation
	hasRestTakeMutation                      = validator.HasRestTakeMutation
	validateRestLongTermCountdownPayload     = validator.ValidateRestLongTermCountdownPayload
	hasIntFieldChange                        = validator.HasIntFieldChange
	hasStringFieldChange                     = validator.HasStringFieldChange
	hasBoolFieldChange                       = validator.HasBoolFieldChange
	abs                                      = validator.Abs
)

// Progression validators
var (
	validateAdversaryCreatePayload          = validator.ValidateAdversaryCreatePayload
	validateAdversaryCreatedPayload         = validator.ValidateAdversaryCreatedPayload
	validateAdversaryUpdatePayload          = validator.ValidateAdversaryUpdatePayload
	validateAdversaryUpdatedPayload         = validator.ValidateAdversaryUpdatedPayload
	validateAdversaryFeatureApplyPayload    = validator.ValidateAdversaryFeatureApplyPayload
	validateAdversaryDeletePayload          = validator.ValidateAdversaryDeletePayload
	validateAdversaryDeletedPayload         = validator.ValidateAdversaryDeletedPayload
	equalAdversaryFeatureStates             = validator.EqualAdversaryFeatureStates
	equalAdversaryPendingExperience         = validator.EqualAdversaryPendingExperience
	validateEnvironmentEntityCreatePayload  = validator.ValidateEnvironmentEntityCreatePayload
	validateEnvironmentEntityCreatedPayload = validator.ValidateEnvironmentEntityCreatedPayload
	validateEnvironmentEntityUpdatePayload  = validator.ValidateEnvironmentEntityUpdatePayload
	validateEnvironmentEntityUpdatedPayload = validator.ValidateEnvironmentEntityUpdatedPayload
	validateEnvironmentEntityDeletePayload  = validator.ValidateEnvironmentEntityDeletePayload
	validateEnvironmentEntityDeletedPayload = validator.ValidateEnvironmentEntityDeletedPayload
	validateLevelUpApplyPayload             = validator.ValidateLevelUpApplyPayload
	validateLevelUpAppliedPayload           = validator.ValidateLevelUpAppliedPayload
	validateGoldUpdatePayload               = validator.ValidateGoldUpdatePayload
	validateGoldUpdatedPayload              = validator.ValidateGoldUpdatedPayload
	validateDomainCardAcquirePayload        = validator.ValidateDomainCardAcquirePayload
	validateDomainCardAcquiredPayload       = validator.ValidateDomainCardAcquiredPayload
	validateEquipmentSwapPayload            = validator.ValidateEquipmentSwapPayload
	validateEquipmentSwappedPayload         = validator.ValidateEquipmentSwappedPayload
	validateConsumableUsePayload            = validator.ValidateConsumableUsePayload
	validateConsumableUsedPayload           = validator.ValidateConsumableUsedPayload
	validateConsumableAcquirePayload        = validator.ValidateConsumableAcquirePayload
	validateConsumableAcquiredPayload       = validator.ValidateConsumableAcquiredPayload
)

// Damage validators
var (
	validateDamageApplyPayload                    = validator.ValidateDamageApplyPayload
	validateDamageAppliedPayload                  = validator.ValidateDamageAppliedPayload
	validateDamageAppliedInvariants               = validator.ValidateDamageAppliedInvariants
	validateMultiTargetDamageApplyPayload         = validator.ValidateMultiTargetDamageApplyPayload
	validateAdversaryDamageApplyPayload           = validator.ValidateAdversaryDamageApplyPayload
	validateAdversaryDamageAppliedPayload         = validator.ValidateAdversaryDamageAppliedPayload
	validateDamageAdapterInvariants               = validator.ValidateDamageAdapterInvariants
	validateDowntimeMoveAppliedPayload            = validator.ValidateDowntimeMoveAppliedPayload
	validateDowntimeMoveAppliedPayloadFields      = validator.ValidateDowntimeMoveAppliedPayloadFields
	validateCharacterTemporaryArmorApplyPayload   = validator.ValidateCharacterTemporaryArmorApplyPayload
	validateCharacterTemporaryArmorAppliedPayload = validator.ValidateCharacterTemporaryArmorAppliedPayload
	hasDamagePatchMutation                        = validator.HasDamagePatchMutation
	isTemporaryArmorDuration                      = validator.IsTemporaryArmorDuration
)
