package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c characterApplication) ClaimCharacterControl(ctx context.Context, campaignID string, in *campaignv1.ClaimCharacterControlRequest) (string, string, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", "", err
	}

	characterRecord, characterID, err := c.loadCharacterForControl(ctx, campaignID, in.GetCharacterId())
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(characterRecord.ParticipantID) != "" {
		return "", "", status.Error(codes.FailedPrecondition, "character already has controller")
	}

	actor, _, err := authz.ResolvePolicyActor(ctx, c.stores.Participant, campaignRecord.ID)
	if err != nil {
		return "", "", err
	}

	participantID := strings.TrimSpace(actor.ID)
	if participantID == "" {
		return "", "", status.Error(codes.PermissionDenied, "missing participant identity")
	}
	if err := c.applyCharacterControlUpdate(ctx, campaignID, characterID, participantID, participantID); err != nil {
		return "", "", err
	}

	return characterID, participantID, nil
}

func (c characterApplication) ReleaseCharacterControl(ctx context.Context, campaignID string, in *campaignv1.ReleaseCharacterControlRequest) (string, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", err
	}

	characterRecord, characterID, err := c.loadCharacterForControl(ctx, campaignID, in.GetCharacterId())
	if err != nil {
		return "", err
	}

	actor, _, err := authz.ResolvePolicyActor(ctx, c.stores.Participant, campaignRecord.ID)
	if err != nil {
		return "", err
	}

	actorID := strings.TrimSpace(actor.ID)
	switch controllerID := strings.TrimSpace(characterRecord.ParticipantID); {
	case controllerID == "":
		return "", status.Error(codes.FailedPrecondition, "character is already unassigned")
	case controllerID != actorID:
		return "", status.Error(codes.PermissionDenied, "character is controlled by another participant")
	}

	if err := c.applyCharacterControlUpdate(ctx, campaignID, characterID, "", actorID); err != nil {
		return "", err
	}

	return characterID, nil
}

func (c characterApplication) SetDefaultControl(ctx context.Context, campaignID string, in *campaignv1.SetDefaultControlRequest) (string, string, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", "", err
	}

	_, characterID, err := c.loadCharacterForControl(ctx, campaignID, in.GetCharacterId())
	if err != nil {
		return "", "", err
	}

	if in.ParticipantId == nil {
		return "", "", status.Error(codes.InvalidArgument, "participant id is required")
	}
	participantID := strings.TrimSpace(in.GetParticipantId().GetValue())
	policyActor, err := authz.RequirePolicyActor(ctx, c.auth, domainauthz.CapabilityManageCharacters, campaignRecord)
	if err != nil {
		return "", "", err
	}
	if err := c.applyCharacterControlUpdate(ctx, campaignID, characterID, participantID, strings.TrimSpace(policyActor.ID)); err != nil {
		return "", "", err
	}

	return characterID, participantID, nil
}

func (c characterApplication) loadCharacterForControl(ctx context.Context, campaignID, characterID string) (storage.CharacterRecord, string, error) {
	resolvedCharacterID, err := validate.RequiredID(characterID, "character id")
	if err != nil {
		return storage.CharacterRecord{}, "", err
	}
	characterRecord, err := c.stores.Character.GetCharacter(ctx, campaignID, resolvedCharacterID)
	if err != nil {
		return storage.CharacterRecord{}, "", err
	}
	return characterRecord, resolvedCharacterID, nil
}

func (c characterApplication) applyCharacterControlUpdate(ctx context.Context, campaignID, characterID, participantID, fallbackActorID string) error {
	identitySnapshot, err := c.resolveCharacterIdentitySnapshot(ctx, campaignID, participantID)
	if err != nil {
		return err
	}

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
		return grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)
	if actorID == "" {
		actorID = strings.TrimSpace(fallbackActorID)
		if actorID != "" {
			actorType = command.ActorTypeParticipant
		}
	}

	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeCharacterUpdate,
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
		return err
	}

	return nil
}
