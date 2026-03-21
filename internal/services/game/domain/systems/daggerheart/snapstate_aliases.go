package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"

// --- Aggregate state type aliases ---

type SnapshotState = snapstate.SnapshotState
type AdversaryState = snapstate.AdversaryState
type EnvironmentEntityState = snapstate.EnvironmentEntityState
type CountdownState = snapstate.CountdownState
type TemporaryArmorBucket = snapstate.TemporaryArmorBucket
type CharacterState = snapstate.CharacterState

// --- Character class state type aliases ---

type CharacterClassState = snapstate.CharacterClassState
type CharacterActiveBeastformState = snapstate.CharacterActiveBeastformState
type CharacterDamageDie = snapstate.CharacterDamageDie
type CharacterUnstoppableState = snapstate.CharacterUnstoppableState

// --- Character subclass state type aliases ---

type CharacterSubclassState = snapstate.CharacterSubclassState

const (
	ElementalChannelAir   = snapstate.ElementalChannelAir
	ElementalChannelEarth = snapstate.ElementalChannelEarth
	ElementalChannelFire  = snapstate.ElementalChannelFire
	ElementalChannelWater = snapstate.ElementalChannelWater
)

// --- Character companion state type aliases ---

type CharacterCompanionState = snapstate.CharacterCompanionState

const (
	CompanionStatusPresent = snapstate.CompanionStatusPresent
	CompanionStatusAway    = snapstate.CompanionStatusAway
)

// --- Character profile type aliases ---

type CharacterProfile = snapstate.CharacterProfile
type CharacterProfileExperience = snapstate.CharacterProfileExperience
type CharacterHeritage = snapstate.CharacterHeritage
type CharacterCompanionExperience = snapstate.CharacterCompanionExperience
type CharacterCompanionSheet = snapstate.CharacterCompanionSheet
type CharacterSubclassTrack = snapstate.CharacterSubclassTrack
type CharacterProfileReplacePayload = snapstate.CharacterProfileReplacePayload
type CharacterProfileReplacedPayload = snapstate.CharacterProfileReplacedPayload
type CharacterProfileDeletePayload = snapstate.CharacterProfileDeletePayload
type CharacterProfileDeletedPayload = snapstate.CharacterProfileDeletedPayload
type CreationProfile = snapstate.CreationProfile

const (
	CompanionSheetDefaultEvasion              = snapstate.CompanionSheetDefaultEvasion
	CompanionSheetDefaultAttackRange          = snapstate.CompanionSheetDefaultAttackRange
	CompanionSheetDefaultDamageDieSides       = snapstate.CompanionSheetDefaultDamageDieSides
	CompanionSheetExperienceModifier          = snapstate.CompanionSheetExperienceModifier
	CompanionDamageTypePhysical               = snapstate.CompanionDamageTypePhysical
	CompanionDamageTypeMagic                  = snapstate.CompanionDamageTypeMagic
	SubclassCreationRequirementCompanionSheet = snapstate.SubclassCreationRequirementCompanionSheet
	SubclassTrackOriginPrimary                = snapstate.SubclassTrackOriginPrimary
	SubclassTrackOriginMulticlass             = snapstate.SubclassTrackOriginMulticlass
	SubclassTrackRankFoundation               = snapstate.SubclassTrackRankFoundation
	SubclassTrackRankSpecialization           = snapstate.SubclassTrackRankSpecialization
	SubclassTrackRankMastery                  = snapstate.SubclassTrackRankMastery
)

// --- Subclass track type aliases ---

type SubclassStatBonuses = snapstate.SubclassStatBonuses
type ActiveSubclassTrackFeatures = snapstate.ActiveSubclassTrackFeatures
type ActiveSubclassRuleSummary = snapstate.ActiveSubclassRuleSummary

// --- System constants ---

const (
	SystemID      = snapstate.SystemID
	SystemVersion = snapstate.SystemVersion

	GMFearMin     = snapstate.GMFearMin
	GMFearMax     = snapstate.GMFearMax
	GMFearDefault = snapstate.GMFearDefault

	HPDefault        = snapstate.HPDefault
	HPMaxDefault     = snapstate.HPMaxDefault
	HopeDefault      = snapstate.HopeDefault
	HopeMaxDefault   = snapstate.HopeMaxDefault
	StressDefault    = snapstate.StressDefault
	StressMaxDefault = snapstate.StressMaxDefault
	ArmorDefault     = snapstate.ArmorDefault
	ArmorMaxDefault  = snapstate.ArmorMaxDefault
	LifeStateAlive   = snapstate.LifeStateAlive
)

// --- Exported helper functions ---

var (
	WithActiveBeastform           = snapstate.WithActiveBeastform
	WithActiveCompanionExperience = snapstate.WithActiveCompanionExperience
	WithCompanionPresent          = snapstate.WithCompanionPresent
)

var (
	CharacterProfileFromStorage           = snapstate.CharacterProfileFromStorage
	EnsurePrimarySubclassTrack            = snapstate.EnsurePrimarySubclassTrack
	PrimarySubclassTrack                  = snapstate.PrimarySubclassTrack
	AdvancePrimarySubclassTrack           = snapstate.AdvancePrimarySubclassTrack
	AddMulticlassSubclassTrack            = snapstate.AddMulticlassSubclassTrack
	ActiveSubclassTrackFeaturesFromLoader = snapstate.ActiveSubclassTrackFeaturesFromLoader
	ActiveSubclassTrackFeaturesFromStore  = snapstate.ActiveSubclassTrackFeaturesFromStore
	UnlockedSubclassStageFeatures         = snapstate.UnlockedSubclassStageFeatures
	FlattenActiveSubclassFeatures         = snapstate.FlattenActiveSubclassFeatures
	SubclassStatBonusesFromFeatures       = snapstate.SubclassStatBonusesFromFeatures
	ApplySubclassStatBonuses              = snapstate.ApplySubclassStatBonuses
	SummarizeActiveSubclassRules          = snapstate.SummarizeActiveSubclassRules
	NewSnapshotState                      = snapstate.NewSnapshotState
)

// --- Unexported helpers used by root-package files ---

var (
	snapshotOrDefault   = snapstate.SnapshotOrDefault
	assertSnapshotState = snapstate.AssertSnapshotState
	appendUnique        = snapstate.AppendUnique
)

var (
	requiresCompanionSheet               = snapstate.RequiresCompanionSheet
	validateSubclassCreationRequirements = snapstate.ValidateSubclassCreationRequirements
	validateSubclassTracks               = snapstate.ValidateSubclassTracks
	nextSubclassTrackRank                = snapstate.NextSubclassTrackRank
)

var (
	normalizedDiceValues         = snapstate.NormalizedDiceValues
	normalizedDamageDice         = snapstate.NormalizedDamageDice
	normalizedActiveBeastformPtr = snapstate.NormalizedActiveBeastformPtr
	normalizedSubclassStatePtr   = snapstate.NormalizedSubclassStatePtr
	normalizedCompanionStatePtr  = snapstate.NormalizedCompanionStatePtr
)
