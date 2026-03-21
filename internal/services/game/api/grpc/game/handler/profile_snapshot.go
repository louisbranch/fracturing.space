package handler

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// ApplyParticipantProfileSnapshot refreshes a seat's social snapshot after a
// user binding or reassignment so the seat immediately reflects the caller's
// current name, pronouns, and avatar. Best-effort: errors are silently
// discarded so the caller's primary operation still succeeds.
func ApplyParticipantProfileSnapshot(
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

	snapshot := LoadSocialProfileSnapshot(ctx, socialClient, userID)
	log.Printf("[game] profile snapshot for user %s participant %s: name=%q pronouns=%q avatar_set=%q avatar_asset=%q",
		userID, participantID, snapshot.Name, snapshot.Pronouns, snapshot.AvatarSetID, snapshot.AvatarAssetID)

	// Apply name/pronouns and avatars as separate commands so that avatar
	// resolution failures (e.g. unknown set in the manifest) don't silently
	// reject the name/pronoun update.
	profileFields := map[string]string{}
	if snapshot.Name != "" {
		profileFields["name"] = snapshot.Name
	}
	if snapshot.Pronouns != "" {
		profileFields["pronouns"] = snapshot.Pronouns
	}
	if len(profileFields) > 0 {
		applyParticipantUpdateFields(ctx, write, applier, campaignID, participantID,
			requestID, invocationID, actorID, actorType, profileFields)
	}

	avatarFields := map[string]string{}
	if snapshot.AvatarSetID != "" {
		avatarFields["avatar_set_id"] = snapshot.AvatarSetID
	}
	if snapshot.AvatarAssetID != "" {
		avatarFields["avatar_asset_id"] = snapshot.AvatarAssetID
	}
	if len(avatarFields) > 0 {
		if err := applyParticipantUpdateFields(ctx, write, applier, campaignID, participantID,
			requestID, invocationID, actorID, actorType, avatarFields); err == nil {
			SyncControlledCharacterAvatars(
				ctx, write, applier, participantStore, characterStore,
				campaignID, participantID, requestID, invocationID, actorID, actorType,
			)
		}
	}
}

// applyParticipantUpdateFields issues a single participant.update domain
// command with the given fields map. Returns nil on success.
func applyParticipantUpdateFields(
	ctx context.Context,
	write domainwrite.WritePath,
	applier projection.Applier,
	campaignID string,
	participantID string,
	requestID string,
	invocationID string,
	actorID string,
	actorType command.ActorType,
	fields map[string]string,
) error {
	payloadJSON, err := json.Marshal(participant.UpdatePayload{
		ParticipantID: ids.ParticipantID(participantID),
		Fields:        fields,
	})
	if err != nil {
		return err
	}
	if _, err = ExecuteAndApplyDomainCommand(
		ctx,
		write,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         CommandTypeParticipantUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: ApplyErrorWithCodePreserve("apply participant event"),
		},
	); err != nil {
		log.Printf("[game] apply participant profile snapshot: %v", err)
		return err
	}
	return nil
}

// SyncControlledCharacterAvatars best-effort synchronizes controlled character
// avatars after a seat claim updates the participant avatar snapshot.
func SyncControlledCharacterAvatars(
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
		_, _ = ExecuteAndApplyDomainCommand(
			ctx,
			write,
			applier,
			commandbuild.Core(commandbuild.CoreInput{
				CampaignID:   campaignID,
				Type:         CommandTypeCharacterUpdate,
				ActorType:    actorType,
				ActorID:      actorID,
				RequestID:    requestID,
				InvocationID: invocationID,
				EntityType:   "character",
				EntityID:     controlledCharacter.ID,
				PayloadJSON:  payloadJSON,
			}),
			domainwrite.Options{
				ApplyErr: ApplyErrorWithCodePreserve("apply character avatar event"),
			},
		)
	}
}
