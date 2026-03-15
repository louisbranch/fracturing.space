package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/charactertransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c characterApplication) UpdateCharacter(ctx context.Context, campaignID string, in *campaignv1.UpdateCharacterRequest) (storage.CharacterRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CharacterRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CharacterRecord{}, err
	}

	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	ch, err := c.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	fields := make(map[string]any)
	if name := in.GetName(); name != nil {
		trimmed := strings.TrimSpace(name.GetValue())
		if trimmed == "" {
			return storage.CharacterRecord{}, status.Error(codes.InvalidArgument, "name must not be empty")
		}
		if err := validate.MaxLength(trimmed, "name", validate.MaxNameLen); err != nil {
			return storage.CharacterRecord{}, err
		}
		ch.Name = trimmed
		fields["name"] = trimmed
	}
	if in.GetKind() != campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
		kind := charactertransport.KindFromProto(in.GetKind())
		if kind == character.KindUnspecified {
			return storage.CharacterRecord{}, status.Error(codes.InvalidArgument, "character kind is invalid")
		}
		ch.Kind = kind
		fields["kind"] = in.GetKind().String()
	}
	if notes := in.GetNotes(); notes != nil {
		trimmedNotes := strings.TrimSpace(notes.GetValue())
		if err := validate.MaxLength(trimmedNotes, "notes", validate.MaxNotesLen); err != nil {
			return storage.CharacterRecord{}, err
		}
		ch.Notes = trimmedNotes
		fields["notes"] = ch.Notes
	}
	if avatarSetID := in.GetAvatarSetId(); avatarSetID != nil {
		trimmed := strings.TrimSpace(avatarSetID.GetValue())
		ch.AvatarSetID = trimmed
		fields["avatar_set_id"] = trimmed
	}
	if avatarAssetID := in.GetAvatarAssetId(); avatarAssetID != nil {
		trimmed := strings.TrimSpace(avatarAssetID.GetValue())
		ch.AvatarAssetID = trimmed
		fields["avatar_asset_id"] = trimmed
	}
	if pronouns := in.GetPronouns(); pronouns != nil {
		ch.Pronouns = sharedpronouns.FromProto(pronouns)
		fields["pronouns"] = ch.Pronouns
	}
	if in.Aliases != nil {
		aliasesJSON, err := json.Marshal(in.GetAliases())
		if err != nil {
			return storage.CharacterRecord{}, status.Errorf(codes.InvalidArgument, "aliases must be a list of strings: %v", err)
		}
		fields["aliases"] = string(aliasesJSON)
	}
	transferOwnershipRequested := false
	if ownerParticipantID := in.GetOwnerParticipantId(); ownerParticipantID != nil {
		trimmed := strings.TrimSpace(ownerParticipantID.GetValue())
		if trimmed == "" {
			return storage.CharacterRecord{}, status.Error(codes.InvalidArgument, "owner_participant_id must not be empty")
		}
		if c.stores.Participant == nil {
			return storage.CharacterRecord{}, status.Error(codes.Internal, "participant store is not configured")
		}
		if _, err := c.stores.Participant.GetParticipant(ctx, campaignID, trimmed); err != nil {
			return storage.CharacterRecord{}, err
		}
		fields["owner_participant_id"] = trimmed
		transferOwnershipRequested = true
	}
	if len(fields) == 0 {
		return storage.CharacterRecord{}, status.Error(codes.InvalidArgument, "at least one field must be provided")
	}

	var policyActor storage.ParticipantRecord
	if transferOwnershipRequested {
		policyActor, err = authz.RequirePolicyActor(ctx, c.auth, domainauthz.CapabilityTransferCharacterOwnership, campaignRecord)
		if err != nil {
			return storage.CharacterRecord{}, err
		}
	} else {
		policyActor, err = authz.RequireCharacterMutationPolicy(
			ctx,
			c.auth,
			campaignRecord,
			characterID,
		)
		if err != nil {
			return storage.CharacterRecord{}, err
		}
	}

	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		actorID = strings.TrimSpace(policyActor.ID)
	}
	payloadFields := make(map[string]string, len(fields))
	for key, value := range fields {
		stringValue, ok := value.(string)
		if !ok {
			return storage.CharacterRecord{}, status.Errorf(codes.Internal, "character update field %s must be string", key)
		}
		payloadFields[key] = stringValue
	}
	payload := character.UpdatePayload{
		CharacterID: ids.CharacterID(characterID),
		Fields:      payloadFields,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.CharacterRecord{}, grpcerror.Internal("encode payload", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
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
		domainwrite.Options{},
	)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	updated, err := c.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return storage.CharacterRecord{}, grpcerror.Internal("load character", err)
	}

	return updated, nil
}
