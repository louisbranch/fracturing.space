package game

import (
	"context"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func authorizePolicyActor(ctx context.Context, stores Stores, capability domainauthz.Capability, campaignRecord storage.CampaignRecord) (storage.ParticipantRecord, string, error) {
	if overrideReason, overrideRequested := adminOverrideFromContext(ctx); overrideRequested {
		overrideUserID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
		if overrideUserID == "" {
			return storage.ParticipantRecord{}, authzReasonDenyMissingIdentity, status.Error(codes.PermissionDenied, "admin override requires authenticated principal")
		}
		if overrideReason == "" {
			return storage.ParticipantRecord{}, authzReasonDenyOverrideReasonRequired, status.Error(codes.PermissionDenied, "admin override reason is required")
		}
		return storage.ParticipantRecord{
			ID:     strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx)),
			UserID: overrideUserID,
		}, authzReasonAllowAdminOverride, nil
	}

	if stores.Participant == nil {
		return storage.ParticipantRecord{}, authzReasonErrorDependencyUnavailable, status.Error(codes.Internal, "participant store is not configured")
	}
	actor, reasonCode, err := resolvePolicyActor(ctx, stores.Participant, campaignRecord.ID)
	if err != nil {
		return storage.ParticipantRecord{}, reasonCode, err
	}
	decision := domainauthz.CanCampaignAccess(actor.CampaignAccess, capability)
	if !decision.Allowed {
		return storage.ParticipantRecord{}, decision.ReasonCode, status.Error(codes.PermissionDenied, "participant lacks permission")
	}
	return actor, decision.ReasonCode, nil
}

func resolvePolicyActor(ctx context.Context, participants storage.ParticipantStore, campaignID string) (storage.ParticipantRecord, string, error) {
	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID != "" {
		actor, err := participants.GetParticipant(ctx, campaignID, actorID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return storage.ParticipantRecord{}, authzReasonDenyActorNotFound, status.Error(codes.PermissionDenied, "participant lacks permission")
			}
			return storage.ParticipantRecord{}, authzReasonErrorActorLoad, grpcerror.Internal("load participant", err)
		}
		return actor, authzReasonAllowAccessLevel, nil
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return storage.ParticipantRecord{}, authzReasonDenyMissingIdentity, status.Error(codes.PermissionDenied, "missing participant identity")
	}

	campaignParticipants, err := participants.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, authzReasonErrorActorLoad, grpcerror.Internal("load participants", err)
	}
	for _, participantRecord := range campaignParticipants {
		if strings.TrimSpace(participantRecord.UserID) == userID {
			return participantRecord, authzReasonAllowAccessLevel, nil
		}
	}
	return storage.ParticipantRecord{}, authzReasonDenyActorNotFound, status.Error(codes.PermissionDenied, "participant lacks permission")
}

// participantOwnsActiveCharacters reports whether participantID currently owns
// at least one active character in projection-backed read state.
func participantOwnsActiveCharacters(ctx context.Context, characters storage.CharacterStore, campaignID, participantID string) (bool, error) {
	if characters == nil {
		return false, status.Error(codes.Internal, "character store is not configured")
	}
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return false, status.Error(codes.InvalidArgument, "participant id is required")
	}

	const pageSize = 200
	pageToken := ""
	for {
		page, err := characters.ListCharacters(ctx, campaignID, pageSize, pageToken)
		if err != nil {
			return false, grpcerror.Internal("list characters", err)
		}
		for _, characterRecord := range page.Characters {
			if strings.TrimSpace(characterRecord.OwnerParticipantID) == participantID {
				return true, nil
			}
		}
		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" {
			break
		}
		pageToken = nextPageToken
	}

	return false, nil
}

// resolveCharacterMutationOwnerParticipantID resolves the owner participant for
// member-only character mutation checks.
//
// The lookup reads owner directly from character projection state.
func resolveCharacterMutationOwnerParticipantID(
	ctx context.Context,
	stores Stores,
	campaignID string,
	characterID string,
) (string, error) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return "", nil
	}

	if stores.Character == nil {
		return "", status.Error(codes.Internal, "character owner store is not configured")
	}
	characterRecord, err := stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return "", nil
		}
		return "", grpcerror.Internal("load character owner", err)
	}
	return strings.TrimSpace(characterRecord.OwnerParticipantID), nil
}
