package storage

import (
	"context"
	"errors"

	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	sessiondomain "github.com/louisbranch/duality-engine/internal/session/domain"
)

// ErrNotFound indicates a requested record is missing.
var ErrNotFound = errors.New("record not found")

// ErrActiveSessionExists indicates an active session already exists for a campaign.
var ErrActiveSessionExists = errors.New("active session already exists for campaign")

// CampaignStore persists campaign metadata records.
type CampaignStore interface {
	Put(ctx context.Context, campaign domain.Campaign) error
	Get(ctx context.Context, id string) (domain.Campaign, error)
	// List returns a page of campaign records starting after the page token.
	List(ctx context.Context, pageSize int, pageToken string) (CampaignPage, error)
}

// CampaignPage describes a page of campaign records.
type CampaignPage struct {
	Campaigns     []domain.Campaign
	NextPageToken string
}

// ParticipantStore persists participant records.
type ParticipantStore interface {
	PutParticipant(ctx context.Context, participant domain.Participant) error
	GetParticipant(ctx context.Context, campaignID, participantID string) (domain.Participant, error)
	// ListParticipantsByCampaign returns all participants for a campaign.
	ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]domain.Participant, error)
	// ListParticipants returns a page of participant records for a campaign starting after the page token.
	ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (ParticipantPage, error)
}

// ParticipantPage describes a page of participant records.
type ParticipantPage struct {
	Participants  []domain.Participant
	NextPageToken string
}

// ActorStore persists actor records.
type ActorStore interface {
	PutActor(ctx context.Context, actor domain.Actor) error
	GetActor(ctx context.Context, campaignID, actorID string) (domain.Actor, error)
	// ListActors returns a page of actor records for a campaign starting after the page token.
	ListActors(ctx context.Context, campaignID string, pageSize int, pageToken string) (ActorPage, error)
}

// ActorPage describes a page of actor records.
type ActorPage struct {
	Actors        []domain.Actor
	NextPageToken string
}

// ControlDefaultStore persists default controller assignments for actors.
type ControlDefaultStore interface {
	// PutControlDefault sets the default controller for an actor in a campaign.
	// Overwrites any existing controller for the same (campaign_id, actor_id) pair.
	PutControlDefault(ctx context.Context, campaignID, actorID string, controller domain.ActorController) error
}

// SessionStore persists session records.
type SessionStore interface {
	// PutSession stores a session record.
	PutSession(ctx context.Context, session sessiondomain.Session) error
	// GetSession retrieves a session by campaign ID and session ID.
	GetSession(ctx context.Context, campaignID, sessionID string) (sessiondomain.Session, error)
	// GetActiveSession retrieves the active session for a campaign, if one exists.
	// Returns ErrNotFound if no active session exists.
	GetActiveSession(ctx context.Context, campaignID string) (sessiondomain.Session, error)
	// PutSessionWithActivePointer atomically stores a session and sets it as the active session for the campaign.
	// Returns an error if an active session already exists for the campaign.
	PutSessionWithActivePointer(ctx context.Context, session sessiondomain.Session) error
}
