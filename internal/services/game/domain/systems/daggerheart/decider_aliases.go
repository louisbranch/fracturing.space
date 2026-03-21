package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"

// --- Decider type and constructor aliases ---

type Decider = decider.Decider

var NewDecider = decider.NewDecider

// --- Command type constant aliases ---

const (
	commandTypeGMMoveApply                  = decider.CommandTypeGMMoveApply
	commandTypeGMFearSet                    = decider.CommandTypeGMFearSet
	commandTypeCharacterProfileReplace      = decider.CommandTypeCharacterProfileReplace
	commandTypeCharacterProfileDelete       = decider.CommandTypeCharacterProfileDelete
	commandTypeCharacterStatePatch          = decider.CommandTypeCharacterStatePatch
	commandTypeConditionChange              = decider.CommandTypeConditionChange
	commandTypeHopeSpend                    = decider.CommandTypeHopeSpend
	commandTypeStressSpend                  = decider.CommandTypeStressSpend
	commandTypeLoadoutSwap                  = decider.CommandTypeLoadoutSwap
	commandTypeRestTake                     = decider.CommandTypeRestTake
	commandTypeCountdownCreate              = decider.CommandTypeCountdownCreate
	commandTypeCountdownUpdate              = decider.CommandTypeCountdownUpdate
	commandTypeCountdownDelete              = decider.CommandTypeCountdownDelete
	commandTypeDamageApply                  = decider.CommandTypeDamageApply
	commandTypeAdversaryDamageApply         = decider.CommandTypeAdversaryDamageApply
	commandTypeCharacterTemporaryArmorApply = decider.CommandTypeCharacterTemporaryArmorApply
	commandTypeAdversaryConditionChange     = decider.CommandTypeAdversaryConditionChange
	commandTypeAdversaryCreate              = decider.CommandTypeAdversaryCreate
	commandTypeAdversaryUpdate              = decider.CommandTypeAdversaryUpdate
	commandTypeAdversaryFeatureApply        = decider.CommandTypeAdversaryFeatureApply
	commandTypeAdversaryDelete              = decider.CommandTypeAdversaryDelete
	commandTypeEnvironmentEntityCreate      = decider.CommandTypeEnvironmentEntityCreate
	commandTypeEnvironmentEntityUpdate      = decider.CommandTypeEnvironmentEntityUpdate
	commandTypeEnvironmentEntityDelete      = decider.CommandTypeEnvironmentEntityDelete
	commandTypeMultiTargetDamageApply       = decider.CommandTypeMultiTargetDamageApply
	commandTypeLevelUpApply                 = decider.CommandTypeLevelUpApply
	commandTypeClassFeatureApply            = decider.CommandTypeClassFeatureApply
	commandTypeSubclassFeatureApply         = decider.CommandTypeSubclassFeatureApply
	commandTypeBeastformTransform           = decider.CommandTypeBeastformTransform
	commandTypeBeastformDrop                = decider.CommandTypeBeastformDrop
	commandTypeCompanionExperienceBegin     = decider.CommandTypeCompanionExperienceBegin
	commandTypeCompanionReturn              = decider.CommandTypeCompanionReturn
	commandTypeGoldUpdate                   = decider.CommandTypeGoldUpdate
	commandTypeDomainCardAcquire            = decider.CommandTypeDomainCardAcquire
	commandTypeEquipmentSwap                = decider.CommandTypeEquipmentSwap
	commandTypeConsumableUse                = decider.CommandTypeConsumableUse
	commandTypeConsumableAcquire            = decider.CommandTypeConsumableAcquire
	commandTypeStatModifierChange           = decider.CommandTypeStatModifierChange
)

// --- Rejection code constant aliases ---

const (
	rejectionCodeGMFearAfterRequired               = decider.RejectionCodeGMFearAfterRequired
	rejectionCodeGMFearOutOfRange                  = decider.RejectionCodeGMFearOutOfRange
	rejectionCodeGMFearUnchanged                   = decider.RejectionCodeGMFearUnchanged
	rejectionCodeGMMoveKindUnsupported             = decider.RejectionCodeGMMoveKindUnsupported
	rejectionCodeGMMoveShapeUnsupported            = decider.RejectionCodeGMMoveShapeUnsupported
	rejectionCodeGMMoveDescriptionRequired         = decider.RejectionCodeGMMoveDescriptionRequired
	rejectionCodeGMMoveFearSpentRequired           = decider.RejectionCodeGMMoveFearSpentRequired
	rejectionCodeGMMoveInsufficientFear            = decider.RejectionCodeGMMoveInsufficientFear
	rejectionCodeCharacterStatePatchNoMutation     = decider.RejectionCodeCharacterStatePatchNoMutation
	rejectionCodeConditionChangeNoMutation         = decider.RejectionCodeConditionChangeNoMutation
	rejectionCodeConditionChangeRemoveMissing      = decider.RejectionCodeConditionChangeRemoveMissing
	rejectionCodeCountdownUpdateNoMutation         = decider.RejectionCodeCountdownUpdateNoMutation
	rejectionCodeCountdownBeforeMismatch           = decider.RejectionCodeCountdownBeforeMismatch
	rejectionCodeDamageBeforeMismatch              = decider.RejectionCodeDamageBeforeMismatch
	rejectionCodeDamageArmorSpendLimit             = decider.RejectionCodeDamageArmorSpendLimit
	rejectionCodeAdversaryDamageBeforeMismatch     = decider.RejectionCodeAdversaryDamageBeforeMismatch
	rejectionCodeAdversaryConditionNoMutation      = decider.RejectionCodeAdversaryConditionNoMutation
	rejectionCodeAdversaryConditionRemoveMissing   = decider.RejectionCodeAdversaryConditionRemoveMissing
	rejectionCodeAdversaryCreateNoMutation         = decider.RejectionCodeAdversaryCreateNoMutation
	rejectionCodeAdversaryFeatureApplyNoMutation   = decider.RejectionCodeAdversaryFeatureApplyNoMutation
	rejectionCodeEnvironmentEntityCreateNoMutation = decider.RejectionCodeEnvironmentEntityCreateNoMutation
	rejectionCodeStatModifierChangeNoMutation      = decider.RejectionCodeStatModifierChangeNoMutation
	rejectionCodePayloadDecodeFailed               = decider.RejectionCodePayloadDecodeFailed
	rejectionCodeCommandTypeUnsupported            = decider.RejectionCodeCommandTypeUnsupported
	rejectionCodeGoldInvalid                       = decider.RejectionCodeGoldInvalid
	rejectionCodeDomainCardAcquireInvalid          = decider.RejectionCodeDomainCardAcquireInvalid
	rejectionCodeEquipmentSwapInvalid              = decider.RejectionCodeEquipmentSwapInvalid
	rejectionCodeConsumableInvalid                 = decider.RejectionCodeConsumableInvalid
)

// --- Domain constant aliases ---

const (
	goldHandfulsMax    = decider.GoldHandfulsMax
	goldBagsMax        = decider.GoldBagsMax
	goldChestsMax      = decider.GoldChestsMax
	consumableStackMax = decider.ConsumableStackMax
)

// --- Handler map and type aliases ---

type daggerheartDecisionHandler = decider.DecisionHandler

var daggerheartDecisionHandlers = decider.DecisionHandlers

// --- Decision function aliases (for root tests) ---

var (
	decideRestTake                = decider.DecideRestTake
	decideCharacterProfileReplace = decider.DecideCharacterProfileReplace
	decideCharacterProfileDelete  = decider.DecideCharacterProfileDelete
)

// --- Helper function aliases (for root tests and production code) ---

var (
	isCharacterStatePatchNoMutation      = decider.IsCharacterStatePatchNoMutation
	isConditionChangeNoMutation          = decider.IsConditionChangeNoMutation
	isCountdownUpdateNoMutation          = decider.IsCountdownUpdateNoMutation
	isEnvironmentEntityCreateNoMutation  = decider.IsEnvironmentEntityCreateNoMutation
	isAdversaryFeatureApplyNoMutation    = decider.IsAdversaryFeatureApplyNoMutation
	isAdversaryCreateNoMutation          = decider.IsAdversaryCreateNoMutation
	isAdversaryConditionChangeNoMutation = decider.IsAdversaryConditionChangeNoMutation
	hasMissingAdversaryConditionRemovals = decider.HasMissingAdversaryConditionRemovals
	hasMissingCharacterConditionRemovals = decider.HasMissingCharacterConditionRemovals
	snapshotCharacterState               = decider.SnapshotCharacterState
	snapshotCountdownState               = decider.SnapshotCountdownState
	snapshotAdversaryState               = decider.SnapshotAdversaryState
	snapshotEnvironmentEntityState       = decider.SnapshotEnvironmentEntityState
	derefInt                             = decider.DerefInt
	hasMissingConditionRemovals          = decider.HasMissingConditionRemovals
	normalizedClassStatePtr              = decider.NormalizedClassStatePtr
	companionStatePtrValue               = decider.CompanionStatePtrValue
	countdownUpdateSnapshotRejection     = decider.CountdownUpdateSnapshotRejection
)
