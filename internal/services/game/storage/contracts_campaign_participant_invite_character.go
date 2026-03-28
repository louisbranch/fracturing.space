package storage

import (
	"context"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
)

// CampaignRecord captures the projection-oriented campaign metadata that APIs read.
type CampaignRecord struct {
	ID               string
	Name             string
	Locale           string
	System           bridge.SystemID
	Status           campaign.Status
	GmMode           campaign.GmMode
	Intent           campaign.Intent
	AccessPolicy     campaign.AccessPolicy
	ParticipantCount int
	CharacterCount   int
	ThemePrompt      string
	CoverAssetID     string
	CoverSetID       string
	AIAgentID        string
	AIAuthEpoch      uint64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletedAt      *time.Time
	ArchivedAt       *time.Time
	LatestSessionAt  *time.Time
}

// CampaignReader provides read-only access to campaign projections.
type CampaignReader interface {
	// Get retrieves a campaign by ID. Returns ErrNotFound if the campaign does not exist.
	Get(ctx context.Context, id string) (CampaignRecord, error)
	// List returns a page of campaign records starting after the page token.
	List(ctx context.Context, pageSize int, pageToken string) (CampaignPage, error)
}

// CampaignAIBindingReader provides AI-binding usage lookups for internal guard rails.
type CampaignAIBindingReader interface {
	ListCampaignIDsByAIAgent(ctx context.Context, aiAgentID string) ([]string, error)
}

// CampaignStore owns the campaign-level projection used by list/detail screens and
// status transitions. Projection handlers use the full interface; read-only consumers
// should prefer CampaignReader.
type CampaignStore interface {
	CampaignReader
	Put(ctx context.Context, c CampaignRecord) error
}

// CampaignPage describes a page of campaign records.
type CampaignPage struct {
	Campaigns     []CampaignRecord
	NextPageToken string
}

// ParticipantRecord captures participation state used by campaign membership queries.
type ParticipantRecord struct {
	ID             string
	CampaignID     string
	UserID         string
	Name           string
	Role           participant.Role
	Controller     participant.Controller
	CampaignAccess participant.CampaignAccess
	AvatarSetID    string
	AvatarAssetID  string
	Pronouns       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ParticipantReader provides read-only access to participant projections.
type ParticipantReader interface {
	// GetParticipant retrieves a participant by campaign and participant ID.
	// Returns ErrNotFound if the participant does not exist.
	GetParticipant(ctx context.Context, campaignID, participantID string) (ParticipantRecord, error)
	// CountParticipants returns the number of participants for a campaign.
	CountParticipants(ctx context.Context, campaignID string) (int, error)
	// ListParticipantsByCampaign returns all participants for a campaign.
	ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]ParticipantRecord, error)
	// ListCampaignIDsByUser returns campaign IDs for a participant user.
	ListCampaignIDsByUser(ctx context.Context, userID string) ([]string, error)
	// ListCampaignIDsByParticipant returns campaign IDs for a participant id.
	ListCampaignIDsByParticipant(ctx context.Context, participantID string) ([]string, error)
	// ListParticipants returns a page of participant records for a campaign starting after the page token.
	ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (ParticipantPage, error)
}

// ParticipantStore owns membership read state, including seat ownership and ordering.
// Projection handlers use the full interface; read-only consumers should prefer
// ParticipantReader.
type ParticipantStore interface {
	ParticipantReader
	PutParticipant(ctx context.Context, p ParticipantRecord) error
	DeleteParticipant(ctx context.Context, campaignID, participantID string) error
}

// ParticipantPage describes a page of participant records.
type ParticipantPage struct {
	Participants  []ParticipantRecord
	NextPageToken string
}

// CharacterRecord captures character identity/state metadata for campaign read views.
type CharacterRecord struct {
	ID                 string
	CampaignID         string
	OwnerParticipantID string
	Name               string
	Kind               character.Kind
	Notes              string
	AvatarSetID        string
	AvatarAssetID      string
	Pronouns           string
	Aliases            []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CharacterReader provides read-only access to character projections.
type CharacterReader interface {
	// GetCharacter retrieves a character by campaign and character ID.
	// Returns ErrNotFound if the character does not exist.
	GetCharacter(ctx context.Context, campaignID, characterID string) (CharacterRecord, error)
	// CountCharacters returns the number of characters for a campaign.
	CountCharacters(ctx context.Context, campaignID string) (int, error)
	// ListCharactersByOwnerParticipant returns all characters owned by one participant
	// within a campaign.
	ListCharactersByOwnerParticipant(ctx context.Context, campaignID, participantID string) ([]CharacterRecord, error)
	// ListCharacters returns a page of character records for a campaign starting after the page token.
	ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (CharacterPage, error)
}

// CharacterStore owns character listing and identity metadata for campaign views.
// Projection handlers use the full interface; read-only consumers should prefer
// CharacterReader.
type CharacterStore interface {
	CharacterReader
	PutCharacter(ctx context.Context, c CharacterRecord) error
	DeleteCharacter(ctx context.Context, campaignID, characterID string) error
}

// CharacterPage describes a page of character records.
type CharacterPage struct {
	Characters    []CharacterRecord
	NextPageToken string
}
