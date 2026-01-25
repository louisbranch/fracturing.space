package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ActorKind describes the kind of actor in a campaign.
type ActorKind int

const (
	// ActorKindUnspecified represents an invalid actor kind value.
	ActorKindUnspecified ActorKind = iota
	// ActorKindPC indicates a player character.
	ActorKindPC
	// ActorKindNPC indicates a non-player character.
	ActorKindNPC
)

var (
	// ErrEmptyActorName indicates a missing actor name.
	ErrEmptyActorName = errors.New("actor name is required")
	// ErrInvalidActorKind indicates a missing or invalid actor kind.
	ErrInvalidActorKind = errors.New("actor kind is required")
)

// Actor represents an actor (PC or NPC) in a campaign.
type Actor struct {
	ID         string
	CampaignID string
	Name       string
	Kind       ActorKind
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateActorInput describes the metadata needed to create an actor.
type CreateActorInput struct {
	CampaignID string
	Name       string
	Kind       ActorKind
	Notes      string
}

// CreateActor creates a new actor with a generated ID and timestamps.
func CreateActor(input CreateActorInput, now func() time.Time, idGenerator func() (string, error)) (Actor, error) {
	if now == nil {
		now = time.Now
	}
	if idGenerator == nil {
		idGenerator = NewID
	}

	normalized, err := NormalizeCreateActorInput(input)
	if err != nil {
		return Actor{}, err
	}

	actorID, err := idGenerator()
	if err != nil {
		return Actor{}, fmt.Errorf("generate actor id: %w", err)
	}

	createdAt := now().UTC()
	return Actor{
		ID:         actorID,
		CampaignID: normalized.CampaignID,
		Name:       normalized.Name,
		Kind:       normalized.Kind,
		Notes:      normalized.Notes,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}, nil
}

// NormalizeCreateActorInput trims and validates actor input metadata.
func NormalizeCreateActorInput(input CreateActorInput) (CreateActorInput, error) {
	input.CampaignID = strings.TrimSpace(input.CampaignID)
	if input.CampaignID == "" {
		return CreateActorInput{}, ErrEmptyCampaignID
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return CreateActorInput{}, ErrEmptyActorName
	}
	if input.Kind == ActorKindUnspecified {
		return CreateActorInput{}, ErrInvalidActorKind
	}
	input.Notes = strings.TrimSpace(input.Notes)
	return input, nil
}
