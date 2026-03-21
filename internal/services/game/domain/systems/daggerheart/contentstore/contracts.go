package contentstore

import (
	"context"
	"time"
)

// DaggerheartSubclassCreationRequirement identifies a structured creation-time
// setup requirement exposed by subclass content.
type DaggerheartSubclassCreationRequirement string

const (
	// DaggerheartSubclassCreationRequirementCompanionSheet marks subclasses that
	// require a companion sheet during creation.
	DaggerheartSubclassCreationRequirementCompanionSheet DaggerheartSubclassCreationRequirement = "companion_sheet_required"
)

// DaggerheartFeatureAutomationStatus captures whether a feature has runtime
// automation in the current implementation.
type DaggerheartFeatureAutomationStatus string

const (
	DaggerheartFeatureAutomationStatusUnspecified DaggerheartFeatureAutomationStatus = ""
	DaggerheartFeatureAutomationStatusSupported   DaggerheartFeatureAutomationStatus = "supported"
	DaggerheartFeatureAutomationStatusUnsupported DaggerheartFeatureAutomationStatus = "unsupported"
)

// DaggerheartSubclassFeatureRuleKind identifies the recurring subclass rule a
// feature participates in when automation is supported.
type DaggerheartSubclassFeatureRuleKind string

const (
	DaggerheartSubclassFeatureRuleKindUnspecified                       DaggerheartSubclassFeatureRuleKind = ""
	DaggerheartSubclassFeatureRuleKindThresholdBonus                    DaggerheartSubclassFeatureRuleKind = "threshold_bonus"
	DaggerheartSubclassFeatureRuleKindHPSlotBonus                       DaggerheartSubclassFeatureRuleKind = "hp_slot_bonus"
	DaggerheartSubclassFeatureRuleKindStressSlotBonus                   DaggerheartSubclassFeatureRuleKind = "stress_slot_bonus"
	DaggerheartSubclassFeatureRuleKindEvasionBonus                      DaggerheartSubclassFeatureRuleKind = "evasion_bonus"
	DaggerheartSubclassFeatureRuleKindEvasionBonusWhileHopeAtLeast      DaggerheartSubclassFeatureRuleKind = "evasion_bonus_while_hope_at_least"
	DaggerheartSubclassFeatureRuleKindGainHopeOnFailureWithFear         DaggerheartSubclassFeatureRuleKind = "gain_hope_on_failure_with_fear"
	DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear DaggerheartSubclassFeatureRuleKind = "bonus_magic_damage_on_success_with_fear"
	DaggerheartSubclassFeatureRuleKindBonusDamageWhileVulnerable        DaggerheartSubclassFeatureRuleKind = "bonus_damage_while_vulnerable"
)

// DaggerheartSubclassThresholdScope identifies which threshold a threshold
// bonus modifies.
type DaggerheartSubclassThresholdScope string

const (
	DaggerheartSubclassThresholdScopeUnspecified DaggerheartSubclassThresholdScope = ""
	DaggerheartSubclassThresholdScopeAll         DaggerheartSubclassThresholdScope = "all"
	DaggerheartSubclassThresholdScopeSevereOnly  DaggerheartSubclassThresholdScope = "severe_only"
)

// DaggerheartSubclassFeatureRule stores the typed recurring runtime behavior
// derived from a subclass feature description.
type DaggerheartSubclassFeatureRule struct {
	Kind              DaggerheartSubclassFeatureRuleKind
	Bonus             int
	RequiredHopeMin   int
	DamageDiceCount   int
	DamageDieSides    int
	UseCharacterLevel bool
	ThresholdScope    DaggerheartSubclassThresholdScope
}

// DaggerheartClassFeatureRuleKind identifies recurring runtime behavior
// derived from a class feature description.
type DaggerheartClassFeatureRuleKind string

const (
	DaggerheartClassFeatureRuleKindUnspecified     DaggerheartClassFeatureRuleKind = ""
	DaggerheartClassFeatureRuleKindUnstoppable     DaggerheartClassFeatureRuleKind = "unstoppable"
	DaggerheartClassFeatureRuleKindRally           DaggerheartClassFeatureRuleKind = "rally"
	DaggerheartClassFeatureRuleKindBeastform       DaggerheartClassFeatureRuleKind = "beastform"
	DaggerheartClassFeatureRuleKindHuntersFocus    DaggerheartClassFeatureRuleKind = "hunters_focus"
	DaggerheartClassFeatureRuleKindCloaked         DaggerheartClassFeatureRuleKind = "cloaked"
	DaggerheartClassFeatureRuleKindSneakAttack     DaggerheartClassFeatureRuleKind = "sneak_attack"
	DaggerheartClassFeatureRuleKindPrayerDice      DaggerheartClassFeatureRuleKind = "prayer_dice"
	DaggerheartClassFeatureRuleKindChannelRawPower DaggerheartClassFeatureRuleKind = "channel_raw_power"
	DaggerheartClassFeatureRuleKindPartingStrike   DaggerheartClassFeatureRuleKind = "parting_strike"
	DaggerheartClassFeatureRuleKindCombatTraining  DaggerheartClassFeatureRuleKind = "combat_training"
	DaggerheartClassFeatureRuleKindStrangePatterns DaggerheartClassFeatureRuleKind = "strange_patterns"
)

// DaggerheartClassFeatureRule stores the typed recurring runtime behavior
// derived from a class feature description.
type DaggerheartClassFeatureRule struct {
	Kind              DaggerheartClassFeatureRuleKind
	Bonus             int
	HopeCost          int
	StressCost        int
	DieSides          int
	UseCharacterLevel bool
}

// DaggerheartHopeFeatureRuleKind identifies recurring runtime behavior for one
// class hope feature.
type DaggerheartHopeFeatureRuleKind string

const (
	DaggerheartHopeFeatureRuleKindUnspecified   DaggerheartHopeFeatureRuleKind = ""
	DaggerheartHopeFeatureRuleKindFrontlineTank DaggerheartHopeFeatureRuleKind = "frontline_tank"
	DaggerheartHopeFeatureRuleKindMakeAScene    DaggerheartHopeFeatureRuleKind = "make_a_scene"
	DaggerheartHopeFeatureRuleKindEvolution     DaggerheartHopeFeatureRuleKind = "evolution"
	DaggerheartHopeFeatureRuleKindHoldThemOff   DaggerheartHopeFeatureRuleKind = "hold_them_off"
	DaggerheartHopeFeatureRuleKindRoguesDodge   DaggerheartHopeFeatureRuleKind = "rogues_dodge"
	DaggerheartHopeFeatureRuleKindLifeSupport   DaggerheartHopeFeatureRuleKind = "life_support"
	DaggerheartHopeFeatureRuleKindVolatileMagic DaggerheartHopeFeatureRuleKind = "volatile_magic"
	DaggerheartHopeFeatureRuleKindNoMercy       DaggerheartHopeFeatureRuleKind = "no_mercy"
	DaggerheartHopeFeatureRuleKindNotThisTime   DaggerheartHopeFeatureRuleKind = "not_this_time"
)

// DaggerheartHopeFeatureRule stores the typed recurring runtime behavior
// derived from a class hope feature description.
type DaggerheartHopeFeatureRule struct {
	Kind     DaggerheartHopeFeatureRuleKind
	Bonus    int
	HopeCost int
}

// DaggerheartFeature captures reusable feature metadata from campaign content.
type DaggerheartFeature struct {
	ID               string
	Name             string
	Description      string
	Level            int
	AutomationStatus DaggerheartFeatureAutomationStatus
	SubclassRule     *DaggerheartSubclassFeatureRule
	ClassRule        *DaggerheartClassFeatureRule
}

// DaggerheartHopeFeature captures one class hope feature row for reuse.
type DaggerheartHopeFeature struct {
	Name             string
	Description      string
	HopeCost         int
	AutomationStatus DaggerheartFeatureAutomationStatus
	HopeFeatureRule  *DaggerheartHopeFeatureRule
}

// DaggerheartClass represents a catalog class content row.
type DaggerheartClass struct {
	ID              string
	Name            string
	StartingEvasion int
	StartingHP      int
	StartingItems   []string
	Features        []DaggerheartFeature
	HopeFeature     DaggerheartHopeFeature
	DomainIDs       []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// DaggerheartSubclass represents a catalog subclass content row.
type DaggerheartSubclass struct {
	ID                     string
	Name                   string
	ClassID                string
	SpellcastTrait         string
	CreationRequirements   []DaggerheartSubclassCreationRequirement
	FoundationFeatures     []DaggerheartFeature
	SpecializationFeatures []DaggerheartFeature
	MasteryFeatures        []DaggerheartFeature
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// DaggerheartHeritage stores reusable ancestry/community catalog rows.
type DaggerheartHeritage struct {
	ID        string
	Name      string
	Kind      string
	Features  []DaggerheartFeature
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DaggerheartExperienceEntry stores reusable experience catalog rows.
type DaggerheartExperienceEntry struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartDamageDie stores a normalized die specification.
type DaggerheartDamageDie struct {
	Sides int
	Count int
}

// DaggerheartAdversaryAttack stores base attack schema for adversary projection.
type DaggerheartAdversaryAttack struct {
	Name        string
	Range       string
	DamageDice  []DaggerheartDamageDie
	DamageBonus int
	DamageType  string
}

// DaggerheartAdversaryExperience stores adversary experience modifiers.
type DaggerheartAdversaryExperience struct {
	Name     string
	Modifier int
}

// DaggerheartAdversaryFeature stores adversary feature details.
type DaggerheartAdversaryFeature struct {
	ID          string
	Name        string
	Kind        string
	Description string
	CostType    string
	Cost        int
}

// DaggerheartAdversaryMinionRule stores the derived recurring minion rule for
// one adversary entry.
type DaggerheartAdversaryMinionRule struct {
	SpilloverDamageStep int
}

// DaggerheartAdversaryHordeRule stores the derived recurring horde rule for
// one adversary entry.
type DaggerheartAdversaryHordeRule struct {
	BloodiedAttack DaggerheartAdversaryAttack
}

// DaggerheartAdversaryRelentlessRule stores the derived recurring relentless
// spotlight cap for one adversary entry.
type DaggerheartAdversaryRelentlessRule struct {
	MaxSpotlightsPerGMTurn int
}

// DaggerheartAdversaryEntry stores catalog-grade adversary definitions.
type DaggerheartAdversaryEntry struct {
	ID              string
	Name            string
	Tier            int
	Role            string
	Description     string
	Motives         string
	Difficulty      int
	MajorThreshold  int
	SevereThreshold int
	HP              int
	Stress          int
	Armor           int
	AttackModifier  int
	StandardAttack  DaggerheartAdversaryAttack
	Experiences     []DaggerheartAdversaryExperience
	Features        []DaggerheartAdversaryFeature
	MinionRule      *DaggerheartAdversaryMinionRule
	HordeRule       *DaggerheartAdversaryHordeRule
	RelentlessRule  *DaggerheartAdversaryRelentlessRule
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// DaggerheartBeastformAttack stores beastform attack schema for rendering.
type DaggerheartBeastformAttack struct {
	Range       string
	Trait       string
	DamageDice  []DaggerheartDamageDie
	DamageBonus int
	DamageType  string
}

// DaggerheartBeastformFeature stores reusable beastform feature rows.
type DaggerheartBeastformFeature struct {
	ID          string
	Name        string
	Description string
}

// DaggerheartBeastformEntry stores beastform catalog rows.
type DaggerheartBeastformEntry struct {
	ID           string
	Name         string
	Tier         int
	Examples     string
	Trait        string
	TraitBonus   int
	EvasionBonus int
	Attack       DaggerheartBeastformAttack
	Advantages   []string
	Features     []DaggerheartBeastformFeature
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DaggerheartCompanionExperienceEntry stores reusable companion experience entries.
type DaggerheartCompanionExperienceEntry struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartLootEntry stores loot catalog entries.
type DaggerheartLootEntry struct {
	ID          string
	Name        string
	Roll        int
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartDamageTypeEntry stores reusable damage-type catalog entries.
type DaggerheartDamageTypeEntry struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartDomain stores reusable domain catalog entries.
type DaggerheartDomain struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartDomainCard stores reusable domain card rows.
type DaggerheartDomainCard struct {
	ID          string
	Name        string
	DomainID    string
	Level       int
	Type        string
	RecallCost  int
	UsageLimit  string
	FeatureText string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartWeapon stores reusable weapon catalog rows.
type DaggerheartWeaponDisplayGroup string

const (
	DaggerheartWeaponDisplayGroupPhysical         DaggerheartWeaponDisplayGroup = "physical"
	DaggerheartWeaponDisplayGroupMagic            DaggerheartWeaponDisplayGroup = "magic"
	DaggerheartWeaponDisplayGroupCombatWheelchair DaggerheartWeaponDisplayGroup = "combat_wheelchair"
)

// DaggerheartWeapon stores reusable weapon catalog rows.
type DaggerheartWeapon struct {
	ID           string
	Name         string
	Category     string
	Tier         int
	Trait        string
	Range        string
	DamageDice   []DaggerheartDamageDie
	DamageType   string
	Burden       int
	Feature      string
	DisplayOrder int
	DisplayGroup DaggerheartWeaponDisplayGroup
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DaggerheartArmor stores reusable armor catalog rows.
type DaggerheartArmor struct {
	ID                  string
	Name                string
	Tier                int
	BaseMajorThreshold  int
	BaseSevereThreshold int
	ArmorScore          int
	Feature             string
	Rules               DaggerheartArmorRules
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DaggerheartArmorAutomationStatus captures whether an armor feature is
// automated by the runtime.
type DaggerheartArmorAutomationStatus string

const (
	DaggerheartArmorAutomationStatusSupported   DaggerheartArmorAutomationStatus = "supported"
	DaggerheartArmorAutomationStatusUnsupported DaggerheartArmorAutomationStatus = "unsupported"
)

// DaggerheartArmorMitigationMode captures which damage types an equipped armor
// can reduce by marking armor slots.
type DaggerheartArmorMitigationMode string

const (
	DaggerheartArmorMitigationModeAny          DaggerheartArmorMitigationMode = "any"
	DaggerheartArmorMitigationModePhysicalOnly DaggerheartArmorMitigationMode = "physical_only"
	DaggerheartArmorMitigationModeMagicOnly    DaggerheartArmorMitigationMode = "magic_only"
)

// DaggerheartArmorRules stores the derived recurring runtime behavior for one
// armor entry. Raw feature text remains the reader-facing source, while these
// fields drive automation.
type DaggerheartArmorRules struct {
	AutomationStatus                DaggerheartArmorAutomationStatus
	MitigationMode                  DaggerheartArmorMitigationMode
	EvasionDelta                    int
	AgilityDelta                    int
	PresenceDelta                   int
	SpellcastRollBonus              int
	AllTraitsDelta                  int
	StressOnMark                    bool
	SeverityReductionSteps          int
	ThresholdBonusWhenArmorDepleted int
	WardedMagicReduction            bool
	HopefulReplaceHopeWithArmor     bool
	ResilientDieSides               int
	ResilientSuccessOnOrAbove       int
	ShiftingAttackDisadvantage      int
	TimeslowingEvasionBonusDieSides int
	SharpDamageBonusDieSides        int
	BurningAttackerStress           int
	ImpenetrableStressCost          int
	ImpenetrableUsesPerShortRest    int
	SilentMovementBonus             int
}

// DaggerheartItem stores reusable item catalog rows.
type DaggerheartItem struct {
	ID          string
	Name        string
	Rarity      string
	Kind        string
	StackMax    int
	Description string
	EffectText  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartEnvironment stores reusable environment catalog rows.
type DaggerheartEnvironment struct {
	ID                    string
	Name                  string
	Tier                  int
	Type                  string
	Difficulty            int
	Impulses              []string
	PotentialAdversaryIDs []string
	Features              []DaggerheartFeature
	Prompts               []string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// DaggerheartContentString stores localized content text for the catalog.
type DaggerheartContentString struct {
	ContentID   string
	ContentType string
	Field       string
	Locale      string
	Text        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartCharacterBuildContentReader provides read access to ancestry and
// progression catalog content.
type DaggerheartCharacterBuildContentReader interface {
	GetDaggerheartClass(ctx context.Context, id string) (DaggerheartClass, error)
	ListDaggerheartClasses(ctx context.Context) ([]DaggerheartClass, error)
	GetDaggerheartSubclass(ctx context.Context, id string) (DaggerheartSubclass, error)
	ListDaggerheartSubclasses(ctx context.Context) ([]DaggerheartSubclass, error)
	GetDaggerheartHeritage(ctx context.Context, id string) (DaggerheartHeritage, error)
	ListDaggerheartHeritages(ctx context.Context) ([]DaggerheartHeritage, error)
	GetDaggerheartExperience(ctx context.Context, id string) (DaggerheartExperienceEntry, error)
	ListDaggerheartExperiences(ctx context.Context) ([]DaggerheartExperienceEntry, error)
}

// DaggerheartEncounterContentReader provides read access to encounter-facing
// catalog content.
type DaggerheartEncounterContentReader interface {
	GetDaggerheartAdversaryEntry(ctx context.Context, id string) (DaggerheartAdversaryEntry, error)
	ListDaggerheartAdversaryEntries(ctx context.Context) ([]DaggerheartAdversaryEntry, error)
	GetDaggerheartBeastform(ctx context.Context, id string) (DaggerheartBeastformEntry, error)
	ListDaggerheartBeastforms(ctx context.Context) ([]DaggerheartBeastformEntry, error)
	GetDaggerheartCompanionExperience(ctx context.Context, id string) (DaggerheartCompanionExperienceEntry, error)
	ListDaggerheartCompanionExperiences(ctx context.Context) ([]DaggerheartCompanionExperienceEntry, error)
	GetDaggerheartLootEntry(ctx context.Context, id string) (DaggerheartLootEntry, error)
	ListDaggerheartLootEntries(ctx context.Context) ([]DaggerheartLootEntry, error)
	GetDaggerheartDamageType(ctx context.Context, id string) (DaggerheartDamageTypeEntry, error)
	ListDaggerheartDamageTypes(ctx context.Context) ([]DaggerheartDamageTypeEntry, error)
	GetDaggerheartEnvironment(ctx context.Context, id string) (DaggerheartEnvironment, error)
	ListDaggerheartEnvironments(ctx context.Context) ([]DaggerheartEnvironment, error)
}

// DaggerheartDomainContentReader provides read access to domain and card catalog
// content.
type DaggerheartDomainContentReader interface {
	GetDaggerheartDomain(ctx context.Context, id string) (DaggerheartDomain, error)
	ListDaggerheartDomains(ctx context.Context) ([]DaggerheartDomain, error)
	GetDaggerheartDomainCard(ctx context.Context, id string) (DaggerheartDomainCard, error)
	ListDaggerheartDomainCards(ctx context.Context) ([]DaggerheartDomainCard, error)
	ListDaggerheartDomainCardsByDomain(ctx context.Context, domainID string) ([]DaggerheartDomainCard, error)
}

// DaggerheartEquipmentContentReader provides read access to equipment catalog
// content.
type DaggerheartEquipmentContentReader interface {
	GetDaggerheartWeapon(ctx context.Context, id string) (DaggerheartWeapon, error)
	ListDaggerheartWeapons(ctx context.Context) ([]DaggerheartWeapon, error)
	GetDaggerheartArmor(ctx context.Context, id string) (DaggerheartArmor, error)
	ListDaggerheartArmor(ctx context.Context) ([]DaggerheartArmor, error)
	GetDaggerheartItem(ctx context.Context, id string) (DaggerheartItem, error)
	ListDaggerheartItems(ctx context.Context) ([]DaggerheartItem, error)
}

// DaggerheartContentLocalizationReader provides read access to localized
// content strings.
type DaggerheartContentLocalizationReader interface {
	ListDaggerheartContentStrings(ctx context.Context, contentType string, contentIDs []string, locale string) ([]DaggerheartContentString, error)
}

// DaggerheartContentReadStore provides grouped read access to all Daggerheart
// catalog content used by APIs.
type DaggerheartContentReadStore interface {
	DaggerheartCharacterBuildContentReader
	DaggerheartEncounterContentReader
	DaggerheartDomainContentReader
	DaggerheartEquipmentContentReader
	DaggerheartContentLocalizationReader
}

// DaggerheartCharacterBuildContentWriter provides write access to ancestry and
// progression catalog content.
type DaggerheartCharacterBuildContentWriter interface {
	PutDaggerheartClass(ctx context.Context, class DaggerheartClass) error
	DeleteDaggerheartClass(ctx context.Context, id string) error
	PutDaggerheartSubclass(ctx context.Context, subclass DaggerheartSubclass) error
	DeleteDaggerheartSubclass(ctx context.Context, id string) error
	PutDaggerheartHeritage(ctx context.Context, heritage DaggerheartHeritage) error
	DeleteDaggerheartHeritage(ctx context.Context, id string) error
	PutDaggerheartExperience(ctx context.Context, experience DaggerheartExperienceEntry) error
	DeleteDaggerheartExperience(ctx context.Context, id string) error
}

// DaggerheartEncounterContentWriter provides write access to encounter-facing
// catalog content.
type DaggerheartEncounterContentWriter interface {
	PutDaggerheartAdversaryEntry(ctx context.Context, adversary DaggerheartAdversaryEntry) error
	DeleteDaggerheartAdversaryEntry(ctx context.Context, id string) error
	PutDaggerheartBeastform(ctx context.Context, beastform DaggerheartBeastformEntry) error
	DeleteDaggerheartBeastform(ctx context.Context, id string) error
	PutDaggerheartCompanionExperience(ctx context.Context, experience DaggerheartCompanionExperienceEntry) error
	DeleteDaggerheartCompanionExperience(ctx context.Context, id string) error
	PutDaggerheartLootEntry(ctx context.Context, entry DaggerheartLootEntry) error
	DeleteDaggerheartLootEntry(ctx context.Context, id string) error
	PutDaggerheartDamageType(ctx context.Context, entry DaggerheartDamageTypeEntry) error
	DeleteDaggerheartDamageType(ctx context.Context, id string) error
	PutDaggerheartEnvironment(ctx context.Context, env DaggerheartEnvironment) error
	DeleteDaggerheartEnvironment(ctx context.Context, id string) error
}

// DaggerheartDomainContentWriter provides write access to domain and card
// catalog content.
type DaggerheartDomainContentWriter interface {
	PutDaggerheartDomain(ctx context.Context, domain DaggerheartDomain) error
	DeleteDaggerheartDomain(ctx context.Context, id string) error
	PutDaggerheartDomainCard(ctx context.Context, card DaggerheartDomainCard) error
	DeleteDaggerheartDomainCard(ctx context.Context, id string) error
}

// DaggerheartEquipmentContentWriter provides write access to equipment catalog
// content.
type DaggerheartEquipmentContentWriter interface {
	PutDaggerheartWeapon(ctx context.Context, weapon DaggerheartWeapon) error
	DeleteDaggerheartWeapon(ctx context.Context, id string) error
	PutDaggerheartArmor(ctx context.Context, armor DaggerheartArmor) error
	DeleteDaggerheartArmor(ctx context.Context, id string) error
	PutDaggerheartItem(ctx context.Context, item DaggerheartItem) error
	DeleteDaggerheartItem(ctx context.Context, id string) error
}

// DaggerheartContentLocalizationWriter provides write access to localized
// content strings.
type DaggerheartContentLocalizationWriter interface {
	PutDaggerheartContentString(ctx context.Context, entry DaggerheartContentString) error
}

// DaggerheartContentWriteStore provides grouped write access to all Daggerheart
// catalog content used by import tooling.
type DaggerheartContentWriteStore interface {
	DaggerheartCharacterBuildContentWriter
	DaggerheartEncounterContentWriter
	DaggerheartDomainContentWriter
	DaggerheartEquipmentContentWriter
	DaggerheartContentLocalizationWriter
}

// DaggerheartContentStore provides read/write access to Daggerheart campaign
// content catalog rows used by bootstrap and content import tooling.
type DaggerheartContentStore interface {
	DaggerheartContentReadStore
	DaggerheartContentWriteStore
}
