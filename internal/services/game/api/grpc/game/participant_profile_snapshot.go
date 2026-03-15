package game

import (
	"context"
	"encoding/json"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

// applyParticipantProfileSnapshot refreshes a seat's social snapshot after a
// user binding or reassignment so copied seats immediately reflect the caller's
// current name, pronouns, and avatar without duplicating the invite-claim flow.
func applyParticipantProfileSnapshot(
	ctx context.Context,
	write domainwriteexec.WritePath,
	applier projection.Applier,
	socialClient socialv1.SocialServiceClient,
	campaignID string,
	participantID string,
	userID string,
	requestID string,
	invocationID string,
	actorID string,
	actorType command.ActorType,
) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	snapshot := loadSocialProfileSnapshot(ctx, socialClient, userID)
	fields := map[string]string{}
	if snapshot.Name != "" {
		fields["name"] = snapshot.Name
	}
	if snapshot.Pronouns != "" {
		fields["pronouns"] = snapshot.Pronouns
	}
	if snapshot.AvatarSetID != "" {
		fields["avatar_set_id"] = snapshot.AvatarSetID
	}
	if snapshot.AvatarAssetID != "" {
		fields["avatar_asset_id"] = snapshot.AvatarAssetID
	}
	if len(fields) == 0 {
		return
	}

	payloadJSON, err := json.Marshal(participant.UpdatePayload{
		ParticipantID: ids.ParticipantID(participantID),
		Fields:        fields,
	})
	if err != nil {
		return
	}

	_, _ = executeAndApplyDomainCommand(
		ctx,
		write,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeParticipantUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: domainApplyErrorWithCodePreserve("apply participant event"),
		},
	)
}
