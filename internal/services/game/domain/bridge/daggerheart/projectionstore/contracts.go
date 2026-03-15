package projectionstore

import (
	"context"
	"time"
)

// DaggerheartCharacterProfile is the stored projection of Daggerheart
// character progression and stats for read-heavy operations.
type DaggerheartCharacterProfile struct {
	CampaignID           string
	CharacterID          string
	Level                int
	HpMax                int
	StressMax            int
	Evasion              int
	MajorThreshold       int
	SevereThreshold      int
	Proficiency          int
	ArmorScore           int
	ArmorMax             int
	Experiences          []DaggerheartExperience
	ClassID              string
	SubclassID           string
	AncestryID           string
	CommunityID          string
	TraitsAssigned       bool
	DetailsRecorded      bool
	StartingWeaponIDs    []string
	StartingArmorID      string
	StartingPotionItemID string
	Background           string
	Description          string
	DomainCardIDs        []string
	Connections          string
	GoldHandfuls         int
	GoldBags             int
	GoldChests           int
	Agility              int
	Strength             int
	Finesse              int
	Instinct             int
	Presence             int
	Knowledge            int
}

// DaggerheartExperience captures character experience modifiers in read form.
type DaggerheartExperience struct {
	Name     string
	Modifier int
}

// DaggerheartCharacterProfilePage describes a page of Daggerheart character
// profiles ordered by stable character ID.
type DaggerheartCharacterProfilePage struct {
	Profiles      []DaggerheartCharacterProfile
	NextPageToken string
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
}
