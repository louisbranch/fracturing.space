package invitetransport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// findInviteClaimByJTI scans claim events for a matching JWT ID so duplicate
// grant tokens are detected and handled idempotently.
func findInviteClaimByJTI(ctx context.Context, store storage.EventStore, campaignID, jti string) (*event.Event, error) {
	if strings.TrimSpace(jti) == "" {
		return nil, nil
	}
	if store == nil {
		return nil, fmt.Errorf("event store is not configured")
	}

	var cursor uint64
	for {
		page, err := store.ListEventsPage(ctx, storage.ListEventsPageRequest{
			CampaignID: campaignID,
			PageSize:   200,
			CursorSeq:  cursor,
			CursorDir:  "fwd",
			Descending: false,
			Filter: storage.EventQueryFilter{
				EventType: string(handler.EventTypeInviteClaimed),
			},
		})
		if err != nil {
			return nil, err
		}
		for i := range page.Events {
			evt := page.Events[i]
			if evt.Type != handler.EventTypeInviteClaimed {
				continue
			}
			var payload invite.ClaimPayload
			if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
				return nil, err
			}
			if payload.JWTID == jti {
				return &evt, nil
			}
		}
		if !page.HasNextPage || len(page.Events) == 0 {
			return nil, nil
		}
		cursor = page.Events[len(page.Events)-1].Seq
	}
}

// applyParticipantProfileSnapshot refreshes a seat's social snapshot after a
// user binding so copied seats immediately reflect the caller's current name,
// pronouns, and avatar without duplicating the invite-claim flow.
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
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return
	}

	snapshot := handler.LoadSocialProfileSnapshot(ctx, socialClient, userID)
	fields := map[string]string{}
	avatarApplied := false
	if snapshot.Name != "" {
		fields["name"] = snapshot.Name
	}
	if snapshot.Pronouns != "" {
		fields["pronouns"] = snapshot.Pronouns
	}
	if snapshot.AvatarSetID != "" {
		fields["avatar_set_id"] = snapshot.AvatarSetID
		avatarApplied = true
	}
	if snapshot.AvatarAssetID != "" {
		fields["avatar_asset_id"] = snapshot.AvatarAssetID
		avatarApplied = true
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

	if _, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		write,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeParticipantUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: handler.ApplyErrorWithCodePreserve("apply participant event"),
		},
	); err != nil {
		return
	}
	if !avatarApplied {
		return
	}

	syncControlledCharacterAvatars(
		ctx,
		write,
		applier,
		participantStore,
		characterStore,
		campaignID,
		participantID,
		requestID,
		invocationID,
		actorID,
		actorType,
	)
}

// syncControlledCharacterAvatars best-effort synchronizes controlled character
// avatars after a seat claim updates the participant avatar snapshot.
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
	if participantStore == nil || characterStore == nil {
		return
	}

	participantRecord, err := participantStore.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return
	}
	controlledCharacters, err := characterStore.ListCharactersByControllerParticipant(ctx, campaignID, participantID)
	if err != nil {
		return
	}

	avatarSetID := strings.TrimSpace(participantRecord.AvatarSetID)
	avatarAssetID := strings.TrimSpace(participantRecord.AvatarAssetID)
	for _, controlledCharacter := range controlledCharacters {
		if strings.TrimSpace(controlledCharacter.AvatarSetID) == avatarSetID &&
			strings.TrimSpace(controlledCharacter.AvatarAssetID) == avatarAssetID {
			continue
		}

		payloadJSON, err := json.Marshal(character.UpdatePayload{
			CharacterID: ids.CharacterID(controlledCharacter.ID),
			Fields: map[string]string{
				"avatar_set_id":   avatarSetID,
				"avatar_asset_id": avatarAssetID,
			},
		})
		if err != nil {
			continue
		}
		_, _ = handler.ExecuteAndApplyDomainCommand(
			ctx,
			write,
			applier,
			commandbuild.Core(commandbuild.CoreInput{
				CampaignID:   campaignID,
				Type:         handler.CommandTypeCharacterUpdate,
				ActorType:    actorType,
				ActorID:      actorID,
				RequestID:    requestID,
				InvocationID: invocationID,
				EntityType:   "character",
				EntityID:     controlledCharacter.ID,
				PayloadJSON:  payloadJSON,
			}),
			domainwrite.Options{
				ApplyErr: handler.ApplyErrorWithCodePreserve("apply character avatar event"),
			},
		)
	}
}
