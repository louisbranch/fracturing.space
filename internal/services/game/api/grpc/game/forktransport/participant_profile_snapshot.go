package forktransport

import (
	"context"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// applyParticipantProfileSnapshot delegates to the shared handler implementation.
func applyParticipantProfileSnapshot(
	ctx context.Context,
	write domainwrite.WritePath,
	applier projection.Applier,
	participantStore storage.ParticipantStore,
	characterStore storage.CharacterStore,
	socialClient socialv1.SocialServiceClient,
	campaignID string,
	participantID string,
	userID string,
	requestID string,
	invocationID string,
	actorID string,
	actorType command.ActorType,
) {
	handler.ApplyParticipantProfileSnapshot(
		ctx, write, applier,
		participantStore, characterStore, socialClient,
		campaignID, participantID, userID,
		requestID, invocationID, actorID, actorType,
	)
}

// syncControlledCharacterAvatars delegates to the shared handler implementation.
func syncControlledCharacterAvatars(
	ctx context.Context,
	write domainwrite.WritePath,
	applier projection.Applier,
	participantStore storage.ParticipantStore,
	characterStore storage.CharacterStore,
	campaignID string,
	participantID string,
	requestID string,
	invocationID string,
	actorID string,
	actorType command.ActorType,
) {
	handler.SyncControlledCharacterAvatars(
		ctx, write, applier,
		participantStore, characterStore,
		campaignID, participantID,
		requestID, invocationID, actorID, actorType,
	)
}
