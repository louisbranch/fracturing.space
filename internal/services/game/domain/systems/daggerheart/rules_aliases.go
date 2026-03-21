package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

// --- Condition types and constants ---

type ConditionClass = rules.ConditionClass

const (
	ConditionClassStandard = rules.ConditionClassStandard
	ConditionClassTag      = rules.ConditionClassTag
	ConditionClassSpecial  = rules.ConditionClassSpecial
)

type ConditionClearTrigger = rules.ConditionClearTrigger

const (
	ConditionClearTriggerShortRest   = rules.ConditionClearTriggerShortRest
	ConditionClearTriggerLongRest    = rules.ConditionClearTriggerLongRest
	ConditionClearTriggerSessionEnd  = rules.ConditionClearTriggerSessionEnd
	ConditionClearTriggerDamageTaken = rules.ConditionClearTriggerDamageTaken
)

type ConditionState = rules.ConditionState

const (
	ConditionHidden     = rules.ConditionHidden
	ConditionRestrained = rules.ConditionRestrained
	ConditionVulnerable = rules.ConditionVulnerable
	ConditionCloaked    = rules.ConditionCloaked
)

// StandardConditionState creates a normalized standard condition entry.
func StandardConditionState(code string, options ...func(*ConditionState)) (ConditionState, error) {
	return rules.StandardConditionState(code, options...)
}

// NormalizeConditionStates validates and normalizes a slice of condition states.
func NormalizeConditionStates(values []ConditionState) ([]ConditionState, error) {
	return rules.NormalizeConditionStates(values)
}

// ConditionStatesEqual reports whether two condition state slices match.
func ConditionStatesEqual(left, right []ConditionState) bool {
	return rules.ConditionStatesEqual(left, right)
}

// NormalizeConditions validates and normalizes legacy string conditions.
func NormalizeConditions(values []string) ([]string, error) {
	return rules.NormalizeConditions(values)
}

// ConditionsEqual reports whether two legacy string condition slices match.
func ConditionsEqual(left, right []string) bool {
	return rules.ConditionsEqual(left, right)
}

// DiffConditionStates returns added/removed condition states between before and after.
func DiffConditionStates(before, after []ConditionState) (added []ConditionState, removed []ConditionState) {
	return rules.DiffConditionStates(before, after)
}

// ConditionCodes extracts the effective code for each condition state.
func ConditionCodes(values []ConditionState) []string {
	return rules.ConditionCodes(values)
}

// Functions only exercised from dedicated tests (now in internal/rules/).
var (
	WithConditionSource           = rules.WithConditionSource
	WithConditionClearTriggers    = rules.WithConditionClearTriggers
	DiffConditions                = rules.DiffConditions
	ClearConditionStatesByTrigger = rules.ClearConditionStatesByTrigger
	HasConditionCode              = rules.HasConditionCode
	RemoveConditionCode           = rules.RemoveConditionCode
)

// --- Countdown types and constants ---

type Countdown = rules.Countdown
type CountdownUpdate = rules.CountdownUpdate

const (
	CountdownKindProgress    = rules.CountdownKindProgress
	CountdownKindConsequence = rules.CountdownKindConsequence
)

const (
	CountdownDirectionIncrease = rules.CountdownDirectionIncrease
	CountdownDirectionDecrease = rules.CountdownDirectionDecrease
)

// ApplyCountdownUpdate applies a delta or override to a countdown.
func ApplyCountdownUpdate(countdown Countdown, delta int, override *int) (CountdownUpdate, error) {
	return rules.ApplyCountdownUpdate(countdown, delta, override)
}

var (
	NormalizeCountdownKind      = rules.NormalizeCountdownKind
	NormalizeCountdownDirection = rules.NormalizeCountdownDirection
)

// --- GM Move types and constants ---

type GMMoveKind = rules.GMMoveKind
type GMMoveShape = rules.GMMoveShape
type GMMoveTargetType = rules.GMMoveTargetType

const (
	GMMoveKindUnspecified      = rules.GMMoveKindUnspecified
	GMMoveKindInterruptAndMove = rules.GMMoveKindInterruptAndMove
	GMMoveKindAdditionalMove   = rules.GMMoveKindAdditionalMove
)

const (
	GMMoveShapeUnspecified            = rules.GMMoveShapeUnspecified
	GMMoveShapeShowWorldReaction      = rules.GMMoveShapeShowWorldReaction
	GMMoveShapeRevealDanger           = rules.GMMoveShapeRevealDanger
	GMMoveShapeForceSplit             = rules.GMMoveShapeForceSplit
	GMMoveShapeMarkStress             = rules.GMMoveShapeMarkStress
	GMMoveShapeShiftEnvironment       = rules.GMMoveShapeShiftEnvironment
	GMMoveShapeSpotlightAdversary     = rules.GMMoveShapeSpotlightAdversary
	GMMoveShapeCaptureImportantTarget = rules.GMMoveShapeCaptureImportantTarget
	GMMoveShapeCustom                 = rules.GMMoveShapeCustom
)

const (
	GMMoveTargetTypeUnspecified         = rules.GMMoveTargetTypeUnspecified
	GMMoveTargetTypeDirectMove          = rules.GMMoveTargetTypeDirectMove
	GMMoveTargetTypeAdversaryFeature    = rules.GMMoveTargetTypeAdversaryFeature
	GMMoveTargetTypeEnvironmentFeature  = rules.GMMoveTargetTypeEnvironmentFeature
	GMMoveTargetTypeAdversaryExperience = rules.GMMoveTargetTypeAdversaryExperience
)

// NormalizeGMMoveKind validates and normalizes a GM move kind.
func NormalizeGMMoveKind(value string) (GMMoveKind, bool) {
	return rules.NormalizeGMMoveKind(value)
}

// NormalizeGMMoveShape validates and normalizes a GM move shape.
func NormalizeGMMoveShape(value string) (GMMoveShape, bool) {
	return rules.NormalizeGMMoveShape(value)
}

// NormalizeGMMoveTargetType validates and normalizes a GM move target type.
func NormalizeGMMoveTargetType(value string) (GMMoveTargetType, bool) {
	return rules.NormalizeGMMoveTargetType(value)
}

// --- GM Fear ---

var (
	ApplyGMFearSpend = rules.ApplyGMFearSpend
	ApplyGMFearGain  = rules.ApplyGMFearGain
)

// --- Damage types and constants ---

type DamageSeverity = rules.DamageSeverity
type DamageType = rules.DamageType
type DamageTypes = rules.DamageTypes
type ResistanceProfile = rules.ResistanceProfile
type DamageOptions = rules.DamageOptions
type DamageResult = rules.DamageResult
type DamageApplication = rules.DamageApplication
type ArmorDamageRules = rules.ArmorDamageRules

const (
	DamageNone    = rules.DamageNone
	DamageMinor   = rules.DamageMinor
	DamageMajor   = rules.DamageMajor
	DamageSevere  = rules.DamageSevere
	DamageMassive = rules.DamageMassive
)

const MaxDamageMarks = rules.MaxDamageMarks

const (
	DamageTypePhysical = rules.DamageTypePhysical
	DamageTypeMagic    = rules.DamageTypeMagic
)

var (
	EvaluateDamage        = rules.EvaluateDamage
	ApplyDamageMarks      = rules.ApplyDamageMarks
	ApplyDamage           = rules.ApplyDamage
	ApplyDamageWithArmor  = rules.ApplyDamageWithArmor
	ReduceDamageWithArmor = rules.ReduceDamageWithArmor
	ApplyResistance       = rules.ApplyResistance
)

// --- Damage roll types ---

type DamageDieSpec = rules.DamageDieSpec
type DamageRollRequest = rules.DamageRollRequest
type DamageRollResult = rules.DamageRollResult

var RollDamage = rules.RollDamage

// --- Damage resolution types ---

type DamageApplyInput = rules.DamageApplyInput
type DamageTarget = rules.DamageTarget

// ResolveDamageApplication computes and applies damage for a target.
func ResolveDamageApplication(target DamageTarget, input DamageApplyInput) (DamageApplication, bool, error) {
	return rules.ResolveDamageApplication(target, input)
}

// armorCanMitigate is an unexported wrapper retained for white-box test coverage.
func armorCanMitigate(armorRules ArmorDamageRules, types DamageTypes) bool {
	return rules.ArmorCanMitigate(armorRules, types)
}

// --- Adversary feature types and constants ---

type AdversaryFeatureAutomationStatus = rules.AdversaryFeatureAutomationStatus
type AdversaryFeatureRuleKind = rules.AdversaryFeatureRuleKind
type AdversaryFeatureRule = rules.AdversaryFeatureRule
type AdversaryFeatureState = rules.AdversaryFeatureState
type AdversaryPendingExperience = rules.AdversaryPendingExperience

const (
	AdversaryFeatureAutomationStatusUnspecified = rules.AdversaryFeatureAutomationStatusUnspecified
	AdversaryFeatureAutomationStatusSupported   = rules.AdversaryFeatureAutomationStatusSupported
	AdversaryFeatureAutomationStatusUnsupported = rules.AdversaryFeatureAutomationStatusUnsupported
)

const (
	AdversaryFeatureRuleKindUnspecified                                 = rules.AdversaryFeatureRuleKindUnspecified
	AdversaryFeatureRuleKindMomentumGainFearOnSuccessfulAttack          = rules.AdversaryFeatureRuleKindMomentumGainFearOnSuccessfulAttack
	AdversaryFeatureRuleKindTerrifyingHopeLossOnSuccessfulAttack        = rules.AdversaryFeatureRuleKindTerrifyingHopeLossOnSuccessfulAttack
	AdversaryFeatureRuleKindGroupAttack                                 = rules.AdversaryFeatureRuleKindGroupAttack
	AdversaryFeatureRuleKindHiddenUntilNextAttack                       = rules.AdversaryFeatureRuleKindHiddenUntilNextAttack
	AdversaryFeatureRuleKindDamageReplacementOnAdvantagedAttack         = rules.AdversaryFeatureRuleKindDamageReplacementOnAdvantagedAttack
	AdversaryFeatureRuleKindDifficultyBonusWhileActive                  = rules.AdversaryFeatureRuleKindDifficultyBonusWhileActive
	AdversaryFeatureRuleKindConditionalDamageReplacementWithContributor = rules.AdversaryFeatureRuleKindConditionalDamageReplacementWithContributor
	AdversaryFeatureRuleKindArmorShredOnSuccessfulAttack                = rules.AdversaryFeatureRuleKindArmorShredOnSuccessfulAttack
	AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit                 = rules.AdversaryFeatureRuleKindRetaliatoryDamageOnCloseHit
	AdversaryFeatureRuleKindFocusTargetDisadvantage                     = rules.AdversaryFeatureRuleKindFocusTargetDisadvantage
)

var (
	ResolveAdversaryFeatureRuntime  = rules.ResolveAdversaryFeatureRuntime
	AdversarySpotlightCap           = rules.AdversarySpotlightCap
	AdversaryIsBloodied             = rules.AdversaryIsBloodied
	AdversaryStandardAttack         = rules.AdversaryStandardAttack
	AdversaryIsMinion               = rules.AdversaryIsMinion
	AdversaryMinionSpilloverDefeats = rules.AdversaryMinionSpilloverDefeats
)

// --- Adversary rules constants ---

const AdversaryDefaultSpotlightCap = rules.AdversaryDefaultSpotlightCap

// --- Armor profile types ---

// CurrentBaseArmor returns the current equipped-armor slots excluding temporary armor.
func CurrentBaseArmor(state projectionstore.DaggerheartCharacterState, armorMax int) int {
	return rules.CurrentBaseArmor(state, armorMax)
}

// --- Stat modifier types and constants ---

type StatModifierTarget = rules.StatModifierTarget
type StatModifierState = rules.StatModifierState

const (
	StatModifierTargetEvasion         = rules.StatModifierTargetEvasion
	StatModifierTargetMajorThreshold  = rules.StatModifierTargetMajorThreshold
	StatModifierTargetSevereThreshold = rules.StatModifierTargetSevereThreshold
	StatModifierTargetProficiency     = rules.StatModifierTargetProficiency
	StatModifierTargetArmorScore      = rules.StatModifierTargetArmorScore
)

var (
	ValidStatModifierTarget     = rules.ValidStatModifierTarget
	NormalizeStatModifiers      = rules.NormalizeStatModifiers
	StatModifiersEqual          = rules.StatModifiersEqual
	DiffStatModifiers           = rules.DiffStatModifiers
	ClearStatModifiersByTrigger = rules.ClearStatModifiersByTrigger
)

var (
	EffectiveArmorRules       = rules.EffectiveArmorRules
	RemoveArmorPassiveEffects = rules.RemoveArmorPassiveEffects
	ApplyArmorProfileEffects  = rules.ApplyArmorProfileEffects
	RemapArmorCurrent         = rules.RemapArmorCurrent
	TemporaryArmorAmount      = rules.TemporaryArmorAmount
	SpendBaseArmorSlot        = rules.SpendBaseArmorSlot
	ArmorTotalAfterBaseSpend  = rules.ArmorTotalAfterBaseSpend
	IsLastBaseArmorSlot       = rules.IsLastBaseArmorSlot
)
