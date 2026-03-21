package decider

// Exported wrappers for root-package alias access. The decider package lives
// under internal/ so these exports have no external visibility.

// DecisionHandler is the exported form of the decision handler type.
type DecisionHandler = decisionHandler

// DecisionHandlers is the exported form of the handler map.
var DecisionHandlers = decisionHandlers

// --- Exported decision functions called by root tests ---

var (
	DecideRestTake                = decideRestTake
	DecideCharacterProfileReplace = decideCharacterProfileReplace
	DecideCharacterProfileDelete  = decideCharacterProfileDelete
)

// --- Exported helper functions called by root tests ---

var (
	IsCharacterStatePatchNoMutation      = isCharacterStatePatchNoMutation
	IsConditionChangeNoMutation          = isConditionChangeNoMutation
	IsCountdownUpdateNoMutation          = isCountdownUpdateNoMutation
	IsEnvironmentEntityCreateNoMutation  = isEnvironmentEntityCreateNoMutation
	IsAdversaryFeatureApplyNoMutation    = isAdversaryFeatureApplyNoMutation
	IsAdversaryCreateNoMutation          = isAdversaryCreateNoMutation
	IsAdversaryConditionChangeNoMutation = isAdversaryConditionChangeNoMutation
	HasMissingAdversaryConditionRemovals = hasMissingAdversaryConditionRemovals
	HasMissingCharacterConditionRemovals = hasMissingCharacterConditionRemovals
	SnapshotCharacterState               = snapshotCharacterState
	SnapshotCountdownState               = snapshotCountdownState
	SnapshotAdversaryState               = snapshotAdversaryState
	DerefInt                             = derefInt
	HasMissingConditionRemovals          = hasMissingConditionRemovals
	HasIntFieldChange                    = hasIntFieldChange
	EqualAdversaryFeatureStates          = equalAdversaryFeatureStates
	EqualAdversaryPendingExperience      = equalAdversaryPendingExperience
)

// --- Exported constants ---

const (
	CommandTypeGMMoveApply                  = commandTypeGMMoveApply
	CommandTypeGMFearSet                    = commandTypeGMFearSet
	CommandTypeCharacterProfileReplace      = commandTypeCharacterProfileReplace
	CommandTypeCharacterProfileDelete       = commandTypeCharacterProfileDelete
	CommandTypeCharacterStatePatch          = commandTypeCharacterStatePatch
	CommandTypeConditionChange              = commandTypeConditionChange
	CommandTypeHopeSpend                    = commandTypeHopeSpend
	CommandTypeStressSpend                  = commandTypeStressSpend
	CommandTypeLoadoutSwap                  = commandTypeLoadoutSwap
	CommandTypeRestTake                     = commandTypeRestTake
	CommandTypeCountdownCreate              = commandTypeCountdownCreate
	CommandTypeCountdownUpdate              = commandTypeCountdownUpdate
	CommandTypeCountdownDelete              = commandTypeCountdownDelete
	CommandTypeDamageApply                  = commandTypeDamageApply
	CommandTypeAdversaryDamageApply         = commandTypeAdversaryDamageApply
	CommandTypeCharacterTemporaryArmorApply = commandTypeCharacterTemporaryArmorApply
	CommandTypeAdversaryConditionChange     = commandTypeAdversaryConditionChange
	CommandTypeAdversaryCreate              = commandTypeAdversaryCreate
	CommandTypeAdversaryUpdate              = commandTypeAdversaryUpdate
	CommandTypeAdversaryFeatureApply        = commandTypeAdversaryFeatureApply
	CommandTypeAdversaryDelete              = commandTypeAdversaryDelete
	CommandTypeEnvironmentEntityCreate      = commandTypeEnvironmentEntityCreate
	CommandTypeEnvironmentEntityUpdate      = commandTypeEnvironmentEntityUpdate
	CommandTypeEnvironmentEntityDelete      = commandTypeEnvironmentEntityDelete
	CommandTypeMultiTargetDamageApply       = commandTypeMultiTargetDamageApply
	CommandTypeLevelUpApply                 = commandTypeLevelUpApply
	CommandTypeClassFeatureApply            = commandTypeClassFeatureApply
	CommandTypeSubclassFeatureApply         = commandTypeSubclassFeatureApply
	CommandTypeBeastformTransform           = commandTypeBeastformTransform
	CommandTypeBeastformDrop                = commandTypeBeastformDrop
	CommandTypeCompanionExperienceBegin     = commandTypeCompanionExperienceBegin
	CommandTypeCompanionReturn              = commandTypeCompanionReturn
	CommandTypeGoldUpdate                   = commandTypeGoldUpdate
	CommandTypeDomainCardAcquire            = commandTypeDomainCardAcquire
	CommandTypeEquipmentSwap                = commandTypeEquipmentSwap
	CommandTypeConsumableUse                = commandTypeConsumableUse
	CommandTypeConsumableAcquire            = commandTypeConsumableAcquire
	CommandTypeStatModifierChange           = commandTypeStatModifierChange
)

const (
	RejectionCodeGMFearAfterRequired               = rejectionCodeGMFearAfterRequired
	RejectionCodeGMFearOutOfRange                  = rejectionCodeGMFearOutOfRange
	RejectionCodeGMFearUnchanged                   = rejectionCodeGMFearUnchanged
	RejectionCodeGMMoveKindUnsupported             = rejectionCodeGMMoveKindUnsupported
	RejectionCodeGMMoveShapeUnsupported            = rejectionCodeGMMoveShapeUnsupported
	RejectionCodeGMMoveDescriptionRequired         = rejectionCodeGMMoveDescriptionRequired
	RejectionCodeGMMoveFearSpentRequired           = rejectionCodeGMMoveFearSpentRequired
	RejectionCodeGMMoveInsufficientFear            = rejectionCodeGMMoveInsufficientFear
	RejectionCodeCharacterStatePatchNoMutation     = rejectionCodeCharacterStatePatchNoMutation
	RejectionCodeConditionChangeNoMutation         = rejectionCodeConditionChangeNoMutation
	RejectionCodeConditionChangeRemoveMissing      = rejectionCodeConditionChangeRemoveMissing
	RejectionCodeCountdownUpdateNoMutation         = rejectionCodeCountdownUpdateNoMutation
	RejectionCodeCountdownBeforeMismatch           = rejectionCodeCountdownBeforeMismatch
	RejectionCodeDamageBeforeMismatch              = rejectionCodeDamageBeforeMismatch
	RejectionCodeDamageArmorSpendLimit             = rejectionCodeDamageArmorSpendLimit
	RejectionCodeAdversaryDamageBeforeMismatch     = rejectionCodeAdversaryDamageBeforeMismatch
	RejectionCodeAdversaryConditionNoMutation      = rejectionCodeAdversaryConditionNoMutation
	RejectionCodeAdversaryConditionRemoveMissing   = rejectionCodeAdversaryConditionRemoveMissing
	RejectionCodeAdversaryCreateNoMutation         = rejectionCodeAdversaryCreateNoMutation
	RejectionCodeAdversaryFeatureApplyNoMutation   = rejectionCodeAdversaryFeatureApplyNoMutation
	RejectionCodeEnvironmentEntityCreateNoMutation = rejectionCodeEnvironmentEntityCreateNoMutation
	RejectionCodeStatModifierChangeNoMutation      = rejectionCodeStatModifierChangeNoMutation
	RejectionCodePayloadDecodeFailed               = rejectionCodePayloadDecodeFailed
	RejectionCodeCommandTypeUnsupported            = rejectionCodeCommandTypeUnsupported

	RejectionCodeGoldInvalid              = rejectionCodeGoldInvalid
	RejectionCodeDomainCardAcquireInvalid = rejectionCodeDomainCardAcquireInvalid
	RejectionCodeEquipmentSwapInvalid     = rejectionCodeEquipmentSwapInvalid
	RejectionCodeConsumableInvalid        = rejectionCodeConsumableInvalid

	GoldHandfulsMax    = goldHandfulsMax
	GoldBagsMax        = goldBagsMax
	GoldChestsMax      = goldChestsMax
	ConsumableStackMax = consumableStackMax
)

// NormalizedClassStatePtr is the exported form for root alias access.
var NormalizedClassStatePtr = normalizedClassStatePtr

// CompanionStatePtrValue is the exported form for root alias access.
var CompanionStatePtrValue = companionStatePtrValue

// SnapshotEnvironmentEntityState is exported for root alias access.
var SnapshotEnvironmentEntityState = snapshotEnvironmentEntityState

// CountdownUpdateSnapshotRejection is exported for root alias access.
var CountdownUpdateSnapshotRejection = countdownUpdateSnapshotRejection
