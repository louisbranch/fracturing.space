package charactertransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type characterIdentitySnapshot struct {
	avatarSetID   string
	avatarAssetID string
}

func (c characterApplication) CreateCharacter(ctx context.Context, campaignID string, in *campaignv1.CreateCharacterRequest) (storage.CharacterRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CharacterRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CharacterRecord{}, err
	}

	name := strings.TrimSpace(in.GetName())
	if name == "" {
		return storage.CharacterRecord{}, apperrors.New(apperrors.CodeCharacterEmptyName, "character name is required")
	}
	if err := validate.MaxLength(name, "name", validate.MaxNameLen); err != nil {
		return storage.CharacterRecord{}, err
	}
	if err := validate.MaxLength(in.GetNotes(), "notes", validate.MaxNotesLen); err != nil {
		return storage.CharacterRecord{}, err
	}
	kind := KindFromProto(in.GetKind())
	if kind == character.KindUnspecified {
		return storage.CharacterRecord{}, apperrors.New(apperrors.CodeCharacterInvalidKind, "character kind is required")
	}
	notes := strings.TrimSpace(in.GetNotes())
	policyActor, err := authz.RequirePolicyActor(ctx, c.auth, domainauthz.CapabilityMutateCharacters, campaignRecord)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	characterID, err := c.idGenerator()
	if err != nil {
		return storage.CharacterRecord{}, grpcerror.Internal("generate character id", err)
	}

	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		actorID = strings.TrimSpace(policyActor.ID)
	}
	defaultParticipantID := ""
	switch {
	case policyActor.Role == participant.RolePlayer && kind == character.KindPC:
		defaultParticipantID = strings.TrimSpace(policyActor.ID)
	case policyActor.Role == participant.RoleGM && kind == character.KindNPC:
		defaultParticipantID = strings.TrimSpace(policyActor.ID)
	}
	avatarSetID := strings.TrimSpace(in.GetAvatarSetId())
	avatarAssetID := strings.TrimSpace(in.GetAvatarAssetId())
	pronounsInput := in.GetPronouns()
	pronounsProvided := pronounsInput != nil
	pronouns := ""
	if pronounsProvided {
		pronouns = strings.TrimSpace(sharedpronouns.FromProto(pronounsInput))
	}
	if avatarSetID == "" && avatarAssetID == "" {
		identitySnapshot, err := c.resolveCharacterIdentitySnapshot(ctx, campaignID, defaultParticipantID)
		if err != nil {
			return storage.CharacterRecord{}, err
		}
		avatarSetID = identitySnapshot.avatarSetID
		avatarAssetID = identitySnapshot.avatarAssetID
	}
	if avatarSetID == "" && avatarAssetID == "" {
		// Defensive fallback: should be unreachable because unassigned identity
		// snapshots default to blank avatar set.
		avatarSetID = assetcatalog.AvatarSetBlankV1
		avatarAssetID = ""
	}

	createPayload := character.CreatePayload{
		CharacterID:        ids.CharacterID(characterID),
		OwnerParticipantID: ids.ParticipantID(strings.TrimSpace(policyActor.ID)),
		Name:               name,
		Kind:               in.GetKind().String(),
		Notes:              notes,
		AvatarSetID:        avatarSetID,
		AvatarAssetID:      avatarAssetID,
		ParticipantID:      ids.ParticipantID(defaultParticipantID),
		Pronouns:           pronouns,
		Aliases:            append([]string(nil), in.GetAliases()...),
	}
	createPayloadJSON, err := json.Marshal(createPayload)
	if err != nil {
		return storage.CharacterRecord{}, grpcerror.Internal("encode create character payload", err)
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
			Type:         handler.CommandTypeCharacterCreate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  createPayloadJSON,
		}),
		domainwrite.Options{},
	)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	created, err := c.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return storage.CharacterRecord{}, grpcerror.Internal("load character", err)
	}
	return created, nil
}

// resolveCharacterIdentitySnapshot returns avatar defaults for one controller
// participant snapshot. Unassigned controllers intentionally use the blank
// avatar set.
func (c characterApplication) resolveCharacterIdentitySnapshot(
	ctx context.Context,
	campaignID string,
	participantID string,
) (characterIdentitySnapshot, error) {
	snapshot := characterIdentitySnapshot{
		avatarSetID:   assetcatalog.AvatarSetBlankV1,
		avatarAssetID: "",
	}
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return snapshot, nil
	}
	if c.stores.Participant == nil {
		return characterIdentitySnapshot{}, status.Error(codes.Internal, "participant store is not configured")
	}

	participantRecord, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return characterIdentitySnapshot{}, err
	}

	resolvedSetID := strings.TrimSpace(participantRecord.AvatarSetID)
	resolvedAssetID := strings.TrimSpace(participantRecord.AvatarAssetID)
	if resolvedSetID != "" && resolvedAssetID != "" {
		snapshot.avatarSetID = resolvedSetID
		snapshot.avatarAssetID = resolvedAssetID
	}
	return snapshot, nil
}
