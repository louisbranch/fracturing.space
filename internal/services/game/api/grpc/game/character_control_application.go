package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c characterApplication) SetDefaultControl(ctx context.Context, campaignID string, in *campaignv1.SetDefaultControlRequest) (string, string, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", "", err
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return "", "", status.Error(codes.InvalidArgument, "character id is required")
	}
	if _, err := c.stores.Character.GetCharacter(ctx, campaignID, characterID); err != nil {
		return "", "", err
	}

	if in.ParticipantId == nil {
		return "", "", status.Error(codes.InvalidArgument, "participant id is required")
	}
	participantID := strings.TrimSpace(in.GetParticipantId().GetValue())
	identitySnapshot, err := c.resolveCharacterIdentitySnapshot(ctx, campaignID, participantID)
	if err != nil {
		return "", "", err
	}
	if err := requirePolicy(ctx, c.stores, domainauthz.CapabilityManageCharacters, campaignRecord); err != nil {
		return "", "", err
	}

	applier := c.stores.Applier()
	payload := character.UpdatePayload{
		CharacterID: ids.CharacterID(characterID),
		Fields: map[string]string{
			"participant_id":  participantID,
			"avatar_set_id":   identitySnapshot.avatarSetID,
			"avatar_asset_id": identitySnapshot.avatarAssetID,
			"pronouns":        identitySnapshot.pronouns,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", "", status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCharacterUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErrMessage: "apply event",
		},
	)
	if err != nil {
		return "", "", err
	}

	return characterID, participantID, nil
}
