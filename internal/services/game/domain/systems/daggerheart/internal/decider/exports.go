package decider

// Exported wrappers for root package wiring and root-package tests. The
// decider package lives under internal/ so these exports stay scoped to the
// Daggerheart implementation tree.

// DecisionHandler is the exported form of the decision handler type.
type DecisionHandler = decisionHandler

// DecisionHandlers is the exported form of the handler map.
var DecisionHandlers = decisionHandlers

// --- Exported decision functions used by root-package tests ---

var (
	DecideRestTake                = decideRestTake
	DecideCharacterProfileReplace = decideCharacterProfileReplace
	DecideCharacterProfileDelete  = decideCharacterProfileDelete
)

// --- Exported helper functions used by root-package tests ---

var (
	IsCharacterStatePatchNoMutation      = isCharacterStatePatchNoMutation
	IsConditionChangeNoMutation          = isConditionChangeNoMutation
	IsSceneCountdownAdvanceNoMutation    = isSceneCountdownAdvanceNoMutation
	IsCampaignCountdownAdvanceNoMutation = isCampaignCountdownAdvanceNoMutation
	IsEnvironmentEntityCreateNoMutation  = isEnvironmentEntityCreateNoMutation
	IsAdversaryFeatureApplyNoMutation    = isAdversaryFeatureApplyNoMutation
	IsAdversaryCreateNoMutation          = isAdversaryCreateNoMutation
	IsAdversaryConditionChangeNoMutation = isAdversaryConditionChangeNoMutation
	HasMissingAdversaryConditionRemovals = hasMissingAdversaryConditionRemovals
	HasMissingCharacterConditionRemovals = hasMissingCharacterConditionRemovals
	SnapshotCharacterState               = snapshotCharacterState
	SnapshotSceneCountdownState          = snapshotSceneCountdownState
	SnapshotCampaignCountdownState       = snapshotCampaignCountdownState
	SnapshotAdversaryState               = snapshotAdversaryState
	DerefInt                             = derefInt
	HasMissingConditionRemovals          = hasMissingConditionRemovals
	HasIntFieldChange                    = hasIntFieldChange
	EqualAdversaryFeatureStates          = equalAdversaryFeatureStates
	EqualAdversaryPendingExperience      = equalAdversaryPendingExperience
)

// --- Exported constants ---

const (
	CommandTypeGMMoveApply                     = commandTypeGMMoveApply
	CommandTypeGMFearSet                       = commandTypeGMFearSet
	CommandTypeCharacterProfileReplace         = commandTypeCharacterProfileReplace
	CommandTypeCharacterProfileDelete          = commandTypeCharacterProfileDelete
	CommandTypeCharacterStatePatch             = commandTypeCharacterStatePatch
	CommandTypeConditionChange                 = commandTypeConditionChange
	CommandTypeHopeSpend                       = commandTypeHopeSpend
	CommandTypeStressSpend                     = commandTypeStressSpend
	CommandTypeLoadoutSwap                     = commandTypeLoadoutSwap
	CommandTypeRestTake                        = commandTypeRestTake
	CommandTypeSceneCountdownCreate            = commandTypeSceneCountdownCreate
	CommandTypeSceneCountdownAdvance           = commandTypeSceneCountdownAdvance
	CommandTypeSceneCountdownTriggerResolve    = commandTypeSceneCountdownTriggerResolve
	CommandTypeSceneCountdownDelete            = commandTypeSceneCountdownDelete
	CommandTypeCampaignCountdownCreate         = commandTypeCampaignCountdownCreate
	CommandTypeCampaignCountdownAdvance        = commandTypeCampaignCountdownAdvance
	CommandTypeCampaignCountdownTriggerResolve = commandTypeCampaignCountdownTriggerResolve
	CommandTypeCampaignCountdownDelete         = commandTypeCampaignCountdownDelete
	CommandTypeDamageApply                     = commandTypeDamageApply
	CommandTypeAdversaryDamageApply            = commandTypeAdversaryDamageApply
	CommandTypeCharacterTemporaryArmorApply    = commandTypeCharacterTemporaryArmorApply
	CommandTypeAdversaryConditionChange        = commandTypeAdversaryConditionChange
	CommandTypeAdversaryCreate                 = commandTypeAdversaryCreate
	CommandTypeAdversaryUpdate                 = commandTypeAdversaryUpdate
	CommandTypeAdversaryFeatureApply           = commandTypeAdversaryFeatureApply
	CommandTypeAdversaryDelete                 = commandTypeAdversaryDelete
	CommandTypeEnvironmentEntityCreate         = commandTypeEnvironmentEntityCreate
	CommandTypeEnvironmentEntityUpdate         = commandTypeEnvironmentEntityUpdate
	CommandTypeEnvironmentEntityDelete         = commandTypeEnvironmentEntityDelete
	CommandTypeMultiTargetDamageApply          = commandTypeMultiTargetDamageApply
	CommandTypeLevelUpApply                    = commandTypeLevelUpApply
	CommandTypeClassFeatureApply               = commandTypeClassFeatureApply
	CommandTypeSubclassFeatureApply            = commandTypeSubclassFeatureApply
	CommandTypeBeastformTransform              = commandTypeBeastformTransform
	CommandTypeBeastformDrop                   = commandTypeBeastformDrop
	CommandTypeCompanionExperienceBegin        = commandTypeCompanionExperienceBegin
	CommandTypeCompanionReturn                 = commandTypeCompanionReturn
	CommandTypeGoldUpdate                      = commandTypeGoldUpdate
	CommandTypeDomainCardAcquire               = commandTypeDomainCardAcquire
	CommandTypeEquipmentSwap                   = commandTypeEquipmentSwap
	CommandTypeConsumableUse                   = commandTypeConsumableUse
	CommandTypeConsumableAcquire               = commandTypeConsumableAcquire
	CommandTypeStatModifierChange              = commandTypeStatModifierChange
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
	RejectionCodeCountdownUpdateNoMutation         = rejectionCodeCountdownAdvanceNoMutation
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

// NormalizedClassStatePtr exposes class-state normalization for root tests.
var NormalizedClassStatePtr = normalizedClassStatePtr

// CompanionStatePtrValue exposes companion-state pointer normalization for
// root tests.
var CompanionStatePtrValue = companionStatePtrValue

// SnapshotEnvironmentEntityState exposes environment snapshot lookup for root
// tests.
var SnapshotEnvironmentEntityState = snapshotEnvironmentEntityState

// SceneCountdownUpdateSnapshotRejection exposes scene-countdown precondition
// logic for root tests.
var SceneCountdownUpdateSnapshotRejection = sceneCountdownAdvanceSnapshotRejection

// CampaignCountdownUpdateSnapshotRejection exposes campaign-countdown
// precondition logic for root tests.
var CampaignCountdownUpdateSnapshotRejection = campaignCountdownAdvanceSnapshotRejection
