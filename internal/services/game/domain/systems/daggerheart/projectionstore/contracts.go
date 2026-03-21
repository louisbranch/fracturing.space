package projectionstore

import (
	"context"
	"time"
)

// DaggerheartExperience captures character experience modifiers in read form.
type DaggerheartExperience struct {
	Name     string
	Modifier int
}

// DaggerheartSubclassCreationRequirement mirrors the persisted subclass
// creation requirements resolved during workflow execution.
type DaggerheartSubclassCreationRequirement string

const (
	// DaggerheartSubclassCreationRequirementCompanionSheet requires a companion
	// sheet to complete subclass selection.
	DaggerheartSubclassCreationRequirementCompanionSheet DaggerheartSubclassCreationRequirement = "companion_sheet_required"
)

// DaggerheartSubclassTrackOrigin identifies whether a subclass track belongs to
// the primary build or a multiclass expansion.
type DaggerheartSubclassTrackOrigin string

const (
	DaggerheartSubclassTrackOriginPrimary    DaggerheartSubclassTrackOrigin = "primary"
	DaggerheartSubclassTrackOriginMulticlass DaggerheartSubclassTrackOrigin = "multiclass"
)

// DaggerheartSubclassTrackRank captures the highest unlocked stage for one
// subclass track.
type DaggerheartSubclassTrackRank string

const (
	DaggerheartSubclassTrackRankFoundation     DaggerheartSubclassTrackRank = "foundation"
	DaggerheartSubclassTrackRankSpecialization DaggerheartSubclassTrackRank = "specialization"
	DaggerheartSubclassTrackRankMastery        DaggerheartSubclassTrackRank = "mastery"
)

// DaggerheartSubclassTrack stores the unlocked subclass stage for one primary
// or multiclass track.
type DaggerheartSubclassTrack struct {
	Origin     DaggerheartSubclassTrackOrigin
	ClassID    string
	SubclassID string
	Rank       DaggerheartSubclassTrackRank
	DomainID   string
}

// DaggerheartHeritageSelection stores the resolved ancestry/community
// selection used for Daggerheart creation and read surfaces.
type DaggerheartHeritageSelection struct {
	AncestryLabel           string
	FirstFeatureAncestryID  string
	FirstFeatureID          string
	SecondFeatureAncestryID string
	SecondFeatureID         string
	CommunityID             string
}

// DaggerheartCompanionExperience stores one companion experience row.
type DaggerheartCompanionExperience struct {
	ExperienceID string
	Name         string
	Modifier     int
}

// DaggerheartCompanionSheet stores the static companion sheet selected during
// character creation.
type DaggerheartCompanionSheet struct {
	AnimalKind        string
	Name              string
	Evasion           int
	Experiences       []DaggerheartCompanionExperience
	AttackDescription string
	AttackRange       string
	DamageDieSides    int
	DamageType        string
}

// DaggerheartCharacterProfile is the stored projection of Daggerheart
// character progression and stats for read-heavy operations.
type DaggerheartCharacterProfile struct {
	CampaignID                   string
	CharacterID                  string
	Level                        int
	HpMax                        int
	StressMax                    int
	Evasion                      int
	MajorThreshold               int
	SevereThreshold              int
	Proficiency                  int
	ArmorScore                   int
	ArmorMax                     int
	Experiences                  []DaggerheartExperience
	ClassID                      string
	SubclassID                   string
	SubclassTracks               []DaggerheartSubclassTrack
	SubclassCreationRequirements []DaggerheartSubclassCreationRequirement
	Heritage                     DaggerheartHeritageSelection
	CompanionSheet               *DaggerheartCompanionSheet
	EquippedArmorID              string
	SpellcastRollBonus           int
	TraitsAssigned               bool
	DetailsRecorded              bool
	StartingWeaponIDs            []string
	StartingArmorID              string
	StartingPotionItemID         string
	Background                   string
	Description                  string
	DomainCardIDs                []string
	Connections                  string
	GoldHandfuls                 int
	GoldBags                     int
	GoldChests                   int
	Agility                      int
	Strength                     int
	Finesse                      int
	Instinct                     int
	Presence                     int
	Knowledge                    int
}

// DaggerheartCharacterProfilePage describes a page of Daggerheart character
// profiles ordered by stable character ID.
type DaggerheartCharacterProfilePage struct {
	Profiles      []DaggerheartCharacterProfile
	NextPageToken string
}

// DaggerheartStatModifier stores a runtime stat modifier applied to a character.
type DaggerheartStatModifier struct {
	ID            string
	Target        string
	Delta         int
	Label         string
	Source        string
	SourceID      string
	ClearTriggers []string
}

// DaggerheartCharacterState stores Daggerheart combat state needed by outcome workflows.
type DaggerheartCharacterState struct {
	CampaignID                    string
	CharacterID                   string
	Hp                            int
	Hope                          int
	HopeMax                       int
	Stress                        int
	Armor                         int
	Conditions                    []DaggerheartConditionState
	TemporaryArmor                []DaggerheartTemporaryArmor
	LifeState                     string
	ClassState                    DaggerheartClassState
	SubclassState                 *DaggerheartSubclassState
	CompanionState                *DaggerheartCompanionState
	ImpenetrableUsedThisShortRest bool
	StatModifiers                 []DaggerheartStatModifier
}

// DaggerheartCompanionState stores mutable companion-owned runtime state in read form.
type DaggerheartCompanionState struct {
	Status             string
	ActiveExperienceID string
}

// DaggerheartSubclassState stores mutable subclass-owned runtime state in read form.
type DaggerheartSubclassState struct {
	BattleRitualUsedThisLongRest           bool
	GiftedPerformerRelaxingSongUses        int
	GiftedPerformerEpicSongUses            int
	GiftedPerformerHeartbreakingSongUses   int
	ContactsEverywhereUsesThisSession      int
	ContactsEverywhereActionDieBonus       int
	ContactsEverywhereDamageDiceBonusCount int
	SparingTouchUsesThisLongRest           int
	ElementalistActionBonus                int
	ElementalistDamageBonus                int
	TranscendenceActive                    bool
	TranscendenceTraitBonusTarget          string
	TranscendenceTraitBonusValue           int
	TranscendenceProficiencyBonus          int
	TranscendenceEvasionBonus              int
	TranscendenceSevereThresholdBonus      int
	ClarityOfNatureUsedThisLongRest        bool
	ElementalChannel                       string
	NemesisTargetID                        string
	RousingSpeechUsedThisLongRest          bool
	WardensProtectionUsedThisLongRest      bool
}

type DaggerheartConditionState struct {
	ID            string
	Class         string
	Standard      string
	Code          string
	Label         string
	Source        string
	SourceID      string
	ClearTriggers []string
}

// DaggerheartClassState stores mutable class-owned runtime state in read form.
type DaggerheartClassState struct {
	AttackBonusUntilRest            int
	EvasionBonusUntilHitOrRest      int
	DifficultyPenaltyUntilRest      int
	FocusTargetID                   string
	ActiveBeastform                 *DaggerheartActiveBeastformState
	StrangePatternsNumber           int
	RallyDice                       []int
	PrayerDice                      []int
	Unstoppable                     DaggerheartUnstoppableState
	ChannelRawPowerUsedThisLongRest bool
}

type DaggerheartDamageDie struct {
	Count int
	Sides int
}

type DaggerheartActiveBeastformState struct {
	BeastformID            string
	BaseTrait              string
	AttackTrait            string
	TraitBonus             int
	EvasionBonus           int
	AttackRange            string
	DamageDice             []DaggerheartDamageDie
	DamageBonus            int
	DamageType             string
	EvolutionTraitOverride string
	DropOnAnyHPMark        bool
}

// DaggerheartUnstoppableState stores the Guardian unstoppable runtime state.
type DaggerheartUnstoppableState struct {
	Active           bool
	CurrentValue     int
	DieSides         int
	UsedThisLongRest bool
}

// DaggerheartTemporaryArmor stores a tracked temporary-armor bucket.
type DaggerheartTemporaryArmor struct {
	Source   string
	Duration string
	SourceID string
	Amount   int
}

// DaggerheartSnapshot stores campaign-level Daggerheart state used during replay.
type DaggerheartSnapshot struct {
	CampaignID            string
	GMFear                int
	ConsecutiveShortRests int
}

// DaggerheartCountdown stores timed countdown state in session read models.
type DaggerheartCountdown struct {
	CampaignID        string
	CountdownID       string
	Name              string
	Kind              string
	Current           int
	Max               int
	Direction         string
	Looping           bool
	Variant           string
	TriggerEventType  string
	LinkedCountdownID string
}

// DaggerheartAdversaryFeatureState stores mutable adversary feature runtime state.
type DaggerheartAdversaryFeatureState struct {
	FeatureID       string
	Status          string
	FocusedTargetID string
}

// DaggerheartAdversaryPendingExperience stores one staged adversary experience modifier.
type DaggerheartAdversaryPendingExperience struct {
	Name     string
	Modifier int
}

// DaggerheartAdversary stores adversary read data used by session renderers.
type DaggerheartAdversary struct {
	CampaignID        string
	AdversaryID       string
	AdversaryEntryID  string
	Name              string
	Kind              string
	SessionID         string
	SceneID           string
	Notes             string
	HP                int
	HPMax             int
	Stress            int
	StressMax         int
	Evasion           int
	Major             int
	Severe            int
	Armor             int
	Conditions        []DaggerheartConditionState
	FeatureStates     []DaggerheartAdversaryFeatureState
	PendingExperience *DaggerheartAdversaryPendingExperience
	SpotlightGateID   string
	SpotlightCount    int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// DaggerheartEnvironmentEntity stores instantiated environment read data used
// by GM Fear workflows and scene renderers.
type DaggerheartEnvironmentEntity struct {
	CampaignID          string
	EnvironmentEntityID string
	EnvironmentID       string
	Name                string
	Type                string
	Tier                int
	Difficulty          int
	SessionID           string
	SceneID             string
	Notes               string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Store provides campaign-scoped Daggerheart projection operations so
// system-specific projection logic stays owned by the Daggerheart system.
type Store interface {
	PutDaggerheartCharacterProfile(ctx context.Context, profile DaggerheartCharacterProfile) error
	GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (DaggerheartCharacterProfile, error)
	ListDaggerheartCharacterProfiles(ctx context.Context, campaignID string, pageSize int, pageToken string) (DaggerheartCharacterProfilePage, error)
	DeleteDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) error

	PutDaggerheartCharacterState(ctx context.Context, state DaggerheartCharacterState) error
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (DaggerheartCharacterState, error)

	PutDaggerheartSnapshot(ctx context.Context, snap DaggerheartSnapshot) error
	GetDaggerheartSnapshot(ctx context.Context, campaignID string) (DaggerheartSnapshot, error)

	PutDaggerheartCountdown(ctx context.Context, countdown DaggerheartCountdown) error
	GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (DaggerheartCountdown, error)
	ListDaggerheartCountdowns(ctx context.Context, campaignID string) ([]DaggerheartCountdown, error)
	DeleteDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) error

	PutDaggerheartAdversary(ctx context.Context, adversary DaggerheartAdversary) error
	GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (DaggerheartAdversary, error)
	ListDaggerheartAdversaries(ctx context.Context, campaignID, sessionID string) ([]DaggerheartAdversary, error)
	DeleteDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) error

	PutDaggerheartEnvironmentEntity(ctx context.Context, environmentEntity DaggerheartEnvironmentEntity) error
	GetDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) (DaggerheartEnvironmentEntity, error)
	ListDaggerheartEnvironmentEntities(ctx context.Context, campaignID, sessionID, sceneID string) ([]DaggerheartEnvironmentEntity, error)
	DeleteDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) error
}
