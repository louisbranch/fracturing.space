package storage

import (
	"context"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = apperrors.New(apperrors.CodeNotFound, "record not found")

// ErrActiveSessionExists indicates an active session already exists for a campaign.
var ErrActiveSessionExists = apperrors.New(apperrors.CodeActiveSessionExists, "active session already exists for campaign")

// CampaignStore persists campaign metadata records.
type CampaignStore interface {
	Put(ctx context.Context, c campaign.Campaign) error
	Get(ctx context.Context, id string) (campaign.Campaign, error)
	// List returns a page of campaign records starting after the page token.
	List(ctx context.Context, pageSize int, pageToken string) (CampaignPage, error)
}

// CampaignPage describes a page of campaign records.
type CampaignPage struct {
	Campaigns     []campaign.Campaign
	NextPageToken string
}

// ParticipantStore persists participant records.
type ParticipantStore interface {
	PutParticipant(ctx context.Context, p participant.Participant) error
	GetParticipant(ctx context.Context, campaignID, participantID string) (participant.Participant, error)
	DeleteParticipant(ctx context.Context, campaignID, participantID string) error
	// ListParticipantsByCampaign returns all participants for a campaign.
	ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]participant.Participant, error)
	// ListParticipants returns a page of participant records for a campaign starting after the page token.
	ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (ParticipantPage, error)
}

// ParticipantPage describes a page of participant records.
type ParticipantPage struct {
	Participants  []participant.Participant
	NextPageToken string
}

// InviteStore persists campaign invite records.
type InviteStore interface {
	PutInvite(ctx context.Context, inv invite.Invite) error
	GetInvite(ctx context.Context, inviteID string) (invite.Invite, error)
	ListInvites(ctx context.Context, campaignID string, recipientUserID string, status invite.Status, pageSize int, pageToken string) (InvitePage, error)
	ListPendingInvites(ctx context.Context, campaignID string, pageSize int, pageToken string) (InvitePage, error)
	ListPendingInvitesForRecipient(ctx context.Context, userID string, pageSize int, pageToken string) (InvitePage, error)
	UpdateInviteStatus(ctx context.Context, inviteID string, status invite.Status, updatedAt time.Time) error
}

// InvitePage describes a page of invites.
type InvitePage struct {
	Invites       []invite.Invite
	NextPageToken string
}

// CharacterStore persists character records.
type CharacterStore interface {
	PutCharacter(ctx context.Context, c character.Character) error
	GetCharacter(ctx context.Context, campaignID, characterID string) (character.Character, error)
	DeleteCharacter(ctx context.Context, campaignID, characterID string) error
	// ListCharacters returns a page of character records for a campaign starting after the page token.
	ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (CharacterPage, error)
}

// CharacterPage describes a page of character records.
type CharacterPage struct {
	Characters    []character.Character
	NextPageToken string
}

// SessionStore persists session records.
type SessionStore interface {
	// PutSession atomically stores a session and sets it as the active session for the campaign.
	// Returns ErrActiveSessionExists if an active session already exists for the campaign.
	PutSession(ctx context.Context, s session.Session) error
	// EndSession marks a session as ended and clears it as active for the campaign.
	// The boolean return value reports whether the session transitioned to ENDED.
	EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (session.Session, bool, error)
	// GetSession retrieves a session by campaign ID and session ID.
	GetSession(ctx context.Context, campaignID, sessionID string) (session.Session, error)
	// GetActiveSession retrieves the active session for a campaign, if one exists.
	// Returns ErrNotFound if no active session exists.
	GetActiveSession(ctx context.Context, campaignID string) (session.Session, error)
	// ListSessions returns a page of session records for a campaign starting after the page token.
	ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (SessionPage, error)
}

// EventStore persists events to the unified event journal.
type EventStore interface {
	// AppendEvent atomically appends an event and returns it with sequence and hash set.
	AppendEvent(ctx context.Context, evt event.Event) (event.Event, error)
	// GetEventByHash retrieves an event by its content hash.
	GetEventByHash(ctx context.Context, hash string) (event.Event, error)
	// GetEventBySeq retrieves a specific event by sequence number.
	GetEventBySeq(ctx context.Context, campaignID string, seq uint64) (event.Event, error)
	// ListEvents returns events ordered by sequence ascending.
	ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error)
	// ListEventsBySession returns events for a specific session.
	ListEventsBySession(ctx context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error)
	// GetLatestEventSeq returns the latest event sequence number for a campaign.
	// Returns 0 if no events exist.
	GetLatestEventSeq(ctx context.Context, campaignID string) (uint64, error)
	// ListEventsPage returns a paginated, filtered, and sorted list of events.
	ListEventsPage(ctx context.Context, req ListEventsPageRequest) (ListEventsPageResult, error)
}

// TelemetryEvent describes an operational telemetry record.
type TelemetryEvent struct {
	Timestamp      time.Time
	EventName      string
	Severity       string
	CampaignID     string
	SessionID      string
	ActorType      string
	ActorID        string
	RequestID      string
	InvocationID   string
	TraceID        string
	SpanID         string
	Attributes     map[string]any
	AttributesJSON []byte
}

// TelemetryStore persists operational telemetry events.
type TelemetryStore interface {
	AppendTelemetryEvent(ctx context.Context, evt TelemetryEvent) error
}

// GameStatistics contains aggregate counts across the game data set.
type GameStatistics struct {
	CampaignCount    int64
	SessionCount     int64
	CharacterCount   int64
	ParticipantCount int64
}

// StatisticsStore provides aggregate statistics.
type StatisticsStore interface {
	// GetGameStatistics returns aggregate counts.
	// When since is nil, counts are for all time.
	GetGameStatistics(ctx context.Context, since *time.Time) (GameStatistics, error)
}

// ListEventsPageRequest describes the parameters for paginated event listing.
type ListEventsPageRequest struct {
	// CampaignID scopes the query to a specific campaign (required).
	CampaignID string
	// PageSize is the maximum number of events to return (default: 50, max: 200).
	PageSize int
	// CursorSeq is the sequence number to paginate from (0 for first page).
	CursorSeq uint64
	// CursorDir is the pagination direction ("fwd" = seq > cursor, "bwd" = seq < cursor).
	CursorDir string
	// CursorReverse indicates whether to temporarily reverse the sort order.
	// This is used for "previous page" navigation to fetch items nearest to the cursor.
	CursorReverse bool
	// Descending orders results by seq desc (newest first) when true.
	Descending bool
	// FilterClause is an optional SQL WHERE clause fragment.
	FilterClause string
	// FilterParams are the positional parameters for the filter clause.
	FilterParams []any
}

// ListEventsPageResult contains the paginated event results.
type ListEventsPageResult struct {
	// Events are the events matching the request.
	Events []event.Event
	// HasNextPage indicates whether more results exist in the forward direction.
	HasNextPage bool
	// HasPrevPage indicates whether more results exist in the backward direction.
	HasPrevPage bool
	// TotalCount is the total number of events matching the filter.
	TotalCount int
}

// RollOutcomeDelta describes a per-character state change.
type RollOutcomeDelta struct {
	CharacterID string
	HopeDelta   int
	StressDelta int
}

// RollOutcomeApplyInput describes the outcome application request for storage.
type RollOutcomeApplyInput struct {
	CampaignID           string
	SessionID            string
	RollSeq              uint64
	Targets              []string
	RequiresComplication bool
	RequestID            string
	InvocationID         string
	ParticipantID        string
	CharacterID          string
	EventTimestamp       time.Time
	CharacterDeltas      []RollOutcomeDelta
	GMFearDelta          int
}

// RollOutcomeApplyResult describes the outcome application result from storage.
type RollOutcomeApplyResult struct {
	UpdatedCharacterStates []DaggerheartCharacterState
	AppliedChanges         []session.OutcomeAppliedChange
	GMFearChanged          bool
	GMFearBefore           int
	GMFearAfter            int
}

// RollOutcomeStore applies roll outcomes atomically.
type RollOutcomeStore interface {
	ApplyRollOutcome(ctx context.Context, input RollOutcomeApplyInput) (RollOutcomeApplyResult, error)
}

// SessionPage describes a page of session records.
type SessionPage struct {
	Sessions      []session.Session
	NextPageToken string
}

// SessionGate describes an open or resolved session gate.
type SessionGate struct {
	CampaignID          string
	SessionID           string
	GateID              string
	GateType            string
	Status              string
	Reason              string
	CreatedAt           time.Time
	CreatedByActorType  string
	CreatedByActorID    string
	ResolvedAt          *time.Time
	ResolvedByActorType string
	ResolvedByActorID   string
	MetadataJSON        []byte
	ResolutionJSON      []byte
}

// SessionGateStore persists session gate projections.
type SessionGateStore interface {
	// PutSessionGate stores a gate record.
	PutSessionGate(ctx context.Context, gate SessionGate) error
	// GetSessionGate retrieves a gate by id.
	GetSessionGate(ctx context.Context, campaignID, sessionID, gateID string) (SessionGate, error)
	// GetOpenSessionGate retrieves the currently open gate for a session.
	GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (SessionGate, error)
}

// SessionSpotlight describes the current session spotlight.
type SessionSpotlight struct {
	CampaignID         string
	SessionID          string
	SpotlightType      string
	CharacterID        string
	UpdatedAt          time.Time
	UpdatedByActorType string
	UpdatedByActorID   string
}

// SessionSpotlightStore persists session spotlight projections.
type SessionSpotlightStore interface {
	// PutSessionSpotlight stores the current spotlight for a session.
	PutSessionSpotlight(ctx context.Context, spotlight SessionSpotlight) error
	// GetSessionSpotlight retrieves the current spotlight for a session.
	GetSessionSpotlight(ctx context.Context, campaignID, sessionID string) (SessionSpotlight, error)
	// ClearSessionSpotlight removes the spotlight for a session.
	ClearSessionSpotlight(ctx context.Context, campaignID, sessionID string) error
}

// Snapshot represents a materialized projection for a campaign as of an event sequence.
// Snapshots are derived from the event journal and are not authoritative.
type Snapshot struct {
	CampaignID          string
	SessionID           string
	EventSeq            uint64
	CharacterStatesJSON []byte
	GMStateJSON         []byte
	SystemStateJSON     []byte
	CreatedAt           time.Time
}

// SnapshotStore persists session snapshots captured at event sequences.
type SnapshotStore interface {
	// PutSnapshot stores a snapshot.
	PutSnapshot(ctx context.Context, snap Snapshot) error
	// GetSnapshot retrieves a snapshot by campaign and session ID.
	GetSnapshot(ctx context.Context, campaignID, sessionID string) (Snapshot, error)
	// GetLatestSnapshot retrieves the most recent snapshot for a campaign.
	GetLatestSnapshot(ctx context.Context, campaignID string) (Snapshot, error)
	// ListSnapshots returns snapshots ordered by event sequence descending.
	ListSnapshots(ctx context.Context, campaignID string, limit int) ([]Snapshot, error)
}

// ParticipantClaim describes a user-to-participant binding in a campaign.
type ParticipantClaim struct {
	CampaignID    string
	UserID        string
	ParticipantID string
	ClaimedAt     time.Time
}

// ClaimIndexStore enforces uniqueness on claimed participants.
type ClaimIndexStore interface {
	// PutParticipantClaim stores a user claim for a participant seat.
	PutParticipantClaim(ctx context.Context, campaignID, userID, participantID string, claimedAt time.Time) error
	// GetParticipantClaim returns the claim for a user in a campaign.
	GetParticipantClaim(ctx context.Context, campaignID, userID string) (ParticipantClaim, error)
	// DeleteParticipantClaim removes a claim by user.
	DeleteParticipantClaim(ctx context.Context, campaignID, userID string) error
}

// ForkMetadata contains fork-related campaign information.
type ForkMetadata struct {
	ParentCampaignID string
	ForkEventSeq     uint64
	OriginCampaignID string
}

// CampaignForkStore provides fork-related campaign operations.
type CampaignForkStore interface {
	// GetCampaignForkMetadata retrieves fork metadata for a campaign.
	GetCampaignForkMetadata(ctx context.Context, campaignID string) (ForkMetadata, error)
	// SetCampaignForkMetadata sets fork metadata for a campaign.
	SetCampaignForkMetadata(ctx context.Context, campaignID string, metadata ForkMetadata) error
}

// ProjectionStore groups projection-related storage interfaces.
type ProjectionStore interface {
	CampaignStore
	ParticipantStore
	ClaimIndexStore
	InviteStore
	CharacterStore
	DaggerheartStore
	SessionStore
	SnapshotStore
	CampaignForkStore
	StatisticsStore
}

// Store is a composite interface for all storage concerns.
// Storage backends (BoltDB, SQLite) implement this interface.
type Store interface {
	CampaignStore
	ParticipantStore
	ClaimIndexStore
	CharacterStore
	InviteStore
	DaggerheartStore
	SessionStore
	EventStore
	TelemetryStore
	StatisticsStore
	RollOutcomeStore
	SnapshotStore
	CampaignForkStore
	Close() error
}

// DaggerheartCharacterProfile contains Daggerheart-specific character profile data.
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

// DaggerheartExperience captures experience modifiers.
type DaggerheartExperience struct {
	Name     string
	Modifier int
}

// DaggerheartCharacterState contains Daggerheart-specific character state data.
type DaggerheartCharacterState struct {
	CampaignID  string
	CharacterID string
	Hp          int
	Hope        int
	HopeMax     int
	Stress      int
	Armor       int
	Conditions  []string
	LifeState   string
}

// DaggerheartSnapshot contains Daggerheart-specific campaign-level state.
type DaggerheartSnapshot struct {
	CampaignID            string
	GMFear                int
	ConsecutiveShortRests int
}

// DaggerheartCountdown contains countdown state for a campaign.
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

// DaggerheartAdversary contains adversary metadata for a campaign.
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

// DaggerheartFeature captures a class, subclass, heritage, or environment feature.
type DaggerheartFeature struct {
	ID          string
	Name        string
	Description string
	Level       int
}

// DaggerheartHopeFeature captures a class hope feature.
type DaggerheartHopeFeature struct {
	Name        string
	Description string
	HopeCost    int
}

// DaggerheartClass represents a content catalog class.
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

// DaggerheartSubclass represents a content catalog subclass.
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

// DaggerheartHeritage represents ancestry or community content.
type DaggerheartHeritage struct {
	ID        string
	Name      string
	Kind      string
	Features  []DaggerheartFeature
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DaggerheartExperienceEntry represents an experience catalog entry.
type DaggerheartExperienceEntry struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartAdversaryAttack represents a standard adversary attack.
type DaggerheartAdversaryAttack struct {
	Name        string
	Range       string
	DamageDice  []DaggerheartDamageDie
	DamageBonus int
	DamageType  string
}

// DaggerheartAdversaryExperience represents an adversary experience bonus.
type DaggerheartAdversaryExperience struct {
	Name     string
	Modifier int
}

// DaggerheartAdversaryFeature represents an adversary feature.
type DaggerheartAdversaryFeature struct {
	ID          string
	Name        string
	Kind        string
	Description string
	CostType    string
	Cost        int
}

// DaggerheartAdversaryEntry represents an adversary catalog entry.
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

// DaggerheartBeastformAttack represents a beastform attack profile.
type DaggerheartBeastformAttack struct {
	Range       string
	Trait       string
	DamageDice  []DaggerheartDamageDie
	DamageBonus int
	DamageType  string
}

// DaggerheartBeastformFeature represents a beastform feature.
type DaggerheartBeastformFeature struct {
	ID          string
	Name        string
	Description string
}

// DaggerheartBeastformEntry represents a beastform catalog entry.
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

// DaggerheartCompanionExperienceEntry represents a companion experience catalog entry.
type DaggerheartCompanionExperienceEntry struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartLootEntry represents a loot catalog entry.
type DaggerheartLootEntry struct {
	ID          string
	Name        string
	Roll        int
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartDamageTypeEntry represents a damage type catalog entry.
type DaggerheartDamageTypeEntry struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartDomain represents a domain catalog entry.
type DaggerheartDomain struct {
	ID          string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartDomainCard represents a domain card catalog entry.
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

// DaggerheartDamageDie represents a damage dice spec for content weapons.
type DaggerheartDamageDie struct {
	Sides int
	Count int
}

// DaggerheartWeapon represents a weapon catalog entry.
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

// DaggerheartArmor represents an armor catalog entry.
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

// DaggerheartItem represents an item catalog entry.
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

// DaggerheartEnvironment represents an environment catalog entry.
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

// DaggerheartContentString stores localized content strings.
type DaggerheartContentString struct {
	ContentID   string
	ContentType string
	Field       string
	Locale      string
	Text        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DaggerheartStore provides Daggerheart-specific storage operations.
// This interface is used for the Daggerheart game system extension tables.
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

// DaggerheartContentStore provides access to the Daggerheart content catalog.
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

	PutDaggerheartContentString(ctx context.Context, entry DaggerheartContentString) error
}
