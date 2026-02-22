package storage

import (
	"context"
	"time"
)

// DaggerheartCharacterProfile is the stored projection of Daggerheart
// character progression and stats for read-heavy operations.
type DaggerheartCharacterProfile struct {
	CampaignID      string
	CharacterID     string
	Level           int
	HpMax           int
	StressMax       int
	Evasion         int
	MajorThreshold  int
	SevereThreshold int
	Proficiency     int
	ArmorScore      int
	ArmorMax        int
	Experiences     []DaggerheartExperience
	// Daggerheart traits
	Agility   int
	Strength  int
	Finesse   int
	Instinct  int
	Presence  int
	Knowledge int
}

// DaggerheartExperience captures character experience modifiers in read form.
type DaggerheartExperience struct {
	Name     string
	Modifier int
}

// DaggerheartCharacterState stores Daggerheart combat state needed by outcome workflows.
type DaggerheartCharacterState struct {
	CampaignID     string
	CharacterID    string
	Hp             int
	Hope           int
	HopeMax        int
	Stress         int
	Armor          int
	Conditions     []string
	TemporaryArmor []DaggerheartTemporaryArmor
	LifeState      string
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
	CampaignID  string
	CountdownID string
	Name        string
	Kind        string
	Current     int
	Max         int
	Direction   string
	Looping     bool
}

// DaggerheartAdversary stores adversary read data used by session renderers.
type DaggerheartAdversary struct {
	CampaignID  string
	AdversaryID string
	Name        string
	Kind        string
	SessionID   string
	Notes       string
	HP          int
	HPMax       int
	Stress      int
	StressMax   int
	Evasion     int
	Major       int
	Severe      int
	Armor       int
	Conditions  []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartFeature captures reusable feature metadata from campaign content.
type DaggerheartFeature struct {
	ID          string
	Name        string
	Description string
	Level       int
}

// DaggerheartHopeFeature captures one class hope feature row for reuse.
type DaggerheartHopeFeature struct {
	Name        string
	Description string
	HopeCost    int
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
	SpellcastTrait         string
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

// DaggerheartDamageDie stores a normalized die specification.
type DaggerheartDamageDie struct {
	Sides int
	Count int
}

// DaggerheartWeapon stores reusable weapon catalog rows.
type DaggerheartWeapon struct {
	ID         string
	Name       string
	Category   string
	Tier       int
	Trait      string
	Range      string
	DamageDice []DaggerheartDamageDie
	DamageType string
	Burden     int
	Feature    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
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
	CreatedAt           time.Time
	UpdatedAt           time.Time
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

// DaggerheartStore provides campaign-scoped Daggerheart extension operations,
// so system-specific projection logic stays isolated from generic projections.
type DaggerheartStore interface {
	// Character Profile Extensions
	PutDaggerheartCharacterProfile(ctx context.Context, profile DaggerheartCharacterProfile) error
	GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (DaggerheartCharacterProfile, error)

	// Character State Extensions
	PutDaggerheartCharacterState(ctx context.Context, state DaggerheartCharacterState) error
	GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (DaggerheartCharacterState, error)

	// Snapshot Extensions
	PutDaggerheartSnapshot(ctx context.Context, snap DaggerheartSnapshot) error
	GetDaggerheartSnapshot(ctx context.Context, campaignID string) (DaggerheartSnapshot, error)

	// Countdown Extensions
	PutDaggerheartCountdown(ctx context.Context, countdown DaggerheartCountdown) error
	GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (DaggerheartCountdown, error)
	ListDaggerheartCountdowns(ctx context.Context, campaignID string) ([]DaggerheartCountdown, error)
	DeleteDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) error

	// Adversary Extensions
	PutDaggerheartAdversary(ctx context.Context, adversary DaggerheartAdversary) error
	GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (DaggerheartAdversary, error)
	ListDaggerheartAdversaries(ctx context.Context, campaignID, sessionID string) ([]DaggerheartAdversary, error)
	DeleteDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) error
}

// DaggerheartContentStore provides read/write access to Daggerheart campaign
// content catalog rows used by bootstrap and content import tooling.
type DaggerheartContentStore interface {
	PutDaggerheartClass(ctx context.Context, class DaggerheartClass) error
	GetDaggerheartClass(ctx context.Context, id string) (DaggerheartClass, error)
	ListDaggerheartClasses(ctx context.Context) ([]DaggerheartClass, error)
	DeleteDaggerheartClass(ctx context.Context, id string) error

	PutDaggerheartSubclass(ctx context.Context, subclass DaggerheartSubclass) error
	GetDaggerheartSubclass(ctx context.Context, id string) (DaggerheartSubclass, error)
	ListDaggerheartSubclasses(ctx context.Context) ([]DaggerheartSubclass, error)
	DeleteDaggerheartSubclass(ctx context.Context, id string) error

	PutDaggerheartHeritage(ctx context.Context, heritage DaggerheartHeritage) error
	GetDaggerheartHeritage(ctx context.Context, id string) (DaggerheartHeritage, error)
	ListDaggerheartHeritages(ctx context.Context) ([]DaggerheartHeritage, error)
	DeleteDaggerheartHeritage(ctx context.Context, id string) error

	PutDaggerheartExperience(ctx context.Context, experience DaggerheartExperienceEntry) error
	GetDaggerheartExperience(ctx context.Context, id string) (DaggerheartExperienceEntry, error)
	ListDaggerheartExperiences(ctx context.Context) ([]DaggerheartExperienceEntry, error)
	DeleteDaggerheartExperience(ctx context.Context, id string) error

	PutDaggerheartAdversaryEntry(ctx context.Context, adversary DaggerheartAdversaryEntry) error
	GetDaggerheartAdversaryEntry(ctx context.Context, id string) (DaggerheartAdversaryEntry, error)
	ListDaggerheartAdversaryEntries(ctx context.Context) ([]DaggerheartAdversaryEntry, error)
	DeleteDaggerheartAdversaryEntry(ctx context.Context, id string) error

	PutDaggerheartBeastform(ctx context.Context, beastform DaggerheartBeastformEntry) error
	GetDaggerheartBeastform(ctx context.Context, id string) (DaggerheartBeastformEntry, error)
	ListDaggerheartBeastforms(ctx context.Context) ([]DaggerheartBeastformEntry, error)
	DeleteDaggerheartBeastform(ctx context.Context, id string) error

	PutDaggerheartCompanionExperience(ctx context.Context, experience DaggerheartCompanionExperienceEntry) error
	GetDaggerheartCompanionExperience(ctx context.Context, id string) (DaggerheartCompanionExperienceEntry, error)
	ListDaggerheartCompanionExperiences(ctx context.Context) ([]DaggerheartCompanionExperienceEntry, error)
	DeleteDaggerheartCompanionExperience(ctx context.Context, id string) error

	PutDaggerheartLootEntry(ctx context.Context, entry DaggerheartLootEntry) error
	GetDaggerheartLootEntry(ctx context.Context, id string) (DaggerheartLootEntry, error)
	ListDaggerheartLootEntries(ctx context.Context) ([]DaggerheartLootEntry, error)
	DeleteDaggerheartLootEntry(ctx context.Context, id string) error

	PutDaggerheartDamageType(ctx context.Context, entry DaggerheartDamageTypeEntry) error
	GetDaggerheartDamageType(ctx context.Context, id string) (DaggerheartDamageTypeEntry, error)
	ListDaggerheartDamageTypes(ctx context.Context) ([]DaggerheartDamageTypeEntry, error)
	DeleteDaggerheartDamageType(ctx context.Context, id string) error

	PutDaggerheartDomain(ctx context.Context, domain DaggerheartDomain) error
	GetDaggerheartDomain(ctx context.Context, id string) (DaggerheartDomain, error)
	ListDaggerheartDomains(ctx context.Context) ([]DaggerheartDomain, error)
	DeleteDaggerheartDomain(ctx context.Context, id string) error

	PutDaggerheartDomainCard(ctx context.Context, card DaggerheartDomainCard) error
	GetDaggerheartDomainCard(ctx context.Context, id string) (DaggerheartDomainCard, error)
	ListDaggerheartDomainCards(ctx context.Context) ([]DaggerheartDomainCard, error)
	ListDaggerheartDomainCardsByDomain(ctx context.Context, domainID string) ([]DaggerheartDomainCard, error)
	DeleteDaggerheartDomainCard(ctx context.Context, id string) error

	PutDaggerheartWeapon(ctx context.Context, weapon DaggerheartWeapon) error
	GetDaggerheartWeapon(ctx context.Context, id string) (DaggerheartWeapon, error)
	ListDaggerheartWeapons(ctx context.Context) ([]DaggerheartWeapon, error)
	DeleteDaggerheartWeapon(ctx context.Context, id string) error

	PutDaggerheartArmor(ctx context.Context, armor DaggerheartArmor) error
	GetDaggerheartArmor(ctx context.Context, id string) (DaggerheartArmor, error)
	ListDaggerheartArmor(ctx context.Context) ([]DaggerheartArmor, error)
	DeleteDaggerheartArmor(ctx context.Context, id string) error

	PutDaggerheartItem(ctx context.Context, item DaggerheartItem) error
	GetDaggerheartItem(ctx context.Context, id string) (DaggerheartItem, error)
	ListDaggerheartItems(ctx context.Context) ([]DaggerheartItem, error)
	DeleteDaggerheartItem(ctx context.Context, id string) error

	PutDaggerheartEnvironment(ctx context.Context, env DaggerheartEnvironment) error
	GetDaggerheartEnvironment(ctx context.Context, id string) (DaggerheartEnvironment, error)
	ListDaggerheartEnvironments(ctx context.Context) ([]DaggerheartEnvironment, error)
	DeleteDaggerheartEnvironment(ctx context.Context, id string) error

	ListDaggerheartContentStrings(ctx context.Context, contentType string, contentIDs []string, locale string) ([]DaggerheartContentString, error)

	PutDaggerheartContentString(ctx context.Context, entry DaggerheartContentString) error
}
