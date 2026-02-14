package game

import (
	"context"
	"errors"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/policy"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type authzParticipantStore struct {
	get func(ctx context.Context, campaignID, participantID string) (participant.Participant, error)
}

func (f authzParticipantStore) PutParticipant(ctx context.Context, p participant.Participant) error {
	return nil
}

func (f authzParticipantStore) GetParticipant(ctx context.Context, campaignID, participantID string) (participant.Participant, error) {
	if f.get == nil {
		return participant.Participant{}, errors.New("missing handler")
	}
	return f.get(ctx, campaignID, participantID)
}

func (f authzParticipantStore) DeleteParticipant(ctx context.Context, campaignID, participantID string) error {
	return nil
}

func (f authzParticipantStore) ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]participant.Participant, error) {
	return nil, nil
}

func (f authzParticipantStore) ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	return storage.ParticipantPage{}, nil
}

func TestRequirePolicyMissingActor(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{}}
	err := requirePolicy(context.Background(), stores, policy.ActionManageParticipants, campaign.Campaign{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyNotFound(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (participant.Participant, error) {
		return participant.Participant{}, storage.ErrNotFound
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policy.ActionManageParticipants, campaign.Campaign{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyLoadError(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (participant.Participant, error) {
		return participant.Participant{}, errors.New("boom")
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policy.ActionManageParticipants, campaign.Campaign{ID: "camp"})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
}

func TestRequirePolicyDenied(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (participant.Participant, error) {
		return participant.Participant{CampaignAccess: participant.CampaignAccessMember}, nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policy.ActionManageParticipants, campaign.Campaign{ID: "camp"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
}

func TestRequirePolicyAllowed(t *testing.T) {
	stores := Stores{Participant: authzParticipantStore{get: func(ctx context.Context, campaignID, participantID string) (participant.Participant, error) {
		return participant.Participant{CampaignAccess: participant.CampaignAccessOwner}, nil
	}}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "participant"))

	err := requirePolicy(ctx, stores, policy.ActionManageParticipants, campaign.Campaign{ID: "camp"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
