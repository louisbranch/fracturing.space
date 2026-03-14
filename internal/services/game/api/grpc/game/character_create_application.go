package game

import (
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
	daggerheartgrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
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
	pronouns      string
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
	kind := characterKindFromProto(in.GetKind())
	if kind == character.KindUnspecified {
		return storage.CharacterRecord{}, apperrors.New(apperrors.CodeCharacterInvalidKind, "character kind is required")
	}
	notes := strings.TrimSpace(in.GetNotes())
	policyActor, err := requirePolicyActor(ctx, c.stores, domainauthz.CapabilityMutateCharacters, campaignRecord)
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
	needsIdentitySnapshot := (avatarSetID == "" && avatarAssetID == "") || !pronounsProvided
	if needsIdentitySnapshot {
		identitySnapshot, err := c.resolveCharacterIdentitySnapshot(ctx, campaignID, defaultParticipantID)
		if err != nil {
			return storage.CharacterRecord{}, err
		}
		if avatarSetID == "" && avatarAssetID == "" {
			avatarSetID = identitySnapshot.avatarSetID
			avatarAssetID = identitySnapshot.avatarAssetID
		}
		if !pronounsProvided {
			pronouns = identitySnapshot.pronouns
		}
	}
	if avatarSetID == "" && avatarAssetID == "" {
		// Defensive fallback: should be unreachable because unassigned identity
		// snapshots default to blank avatar set.
		avatarSetID = assetcatalog.AvatarSetBlankV1
		avatarAssetID = ""
	}

	systemProfile := defaultDaggerheartSystemProfile(kind)

	workflowPayload := character.CreateWithProfilePayload{
		Create: character.CreatePayload{
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
		},
		SystemProfile: systemProfile,
	}
	workflowPayloadJSON, err := json.Marshal(workflowPayload)
	if err != nil {
		return storage.CharacterRecord{}, grpcerror.Internal("encode create workflow payload", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Write,
		c.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCharacterCreateWithProfile,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  workflowPayloadJSON,
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

// defaultDaggerheartSystemProfile builds the canonical default system_profile
// payload used by character creation.
func defaultDaggerheartSystemProfile(kind character.Kind) map[string]any {
	kindLabel := "PC"
	if kind == character.KindNPC {
		kindLabel = "NPC"
	}
	dhDefaults := daggerheartprofile.GetDefaults(kindLabel)

	experiences := make([]storage.DaggerheartExperience, 0, len(dhDefaults.Experiences))
	for _, experience := range dhDefaults.Experiences {
		experiences = append(experiences, storage.DaggerheartExperience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}

	return daggerheartgrpc.SystemProfileMap(storage.DaggerheartCharacterProfile{
		Level:             dhDefaults.Level,
		HpMax:             dhDefaults.HpMax,
		StressMax:         dhDefaults.StressMax,
		Evasion:           dhDefaults.Evasion,
		MajorThreshold:    dhDefaults.MajorThreshold,
		SevereThreshold:   dhDefaults.SevereThreshold,
		Proficiency:       dhDefaults.Proficiency,
		ArmorScore:        dhDefaults.ArmorScore,
		ArmorMax:          dhDefaults.ArmorMax,
		Agility:           dhDefaults.Traits.Agility,
		Strength:          dhDefaults.Traits.Strength,
		Finesse:           dhDefaults.Traits.Finesse,
		Instinct:          dhDefaults.Traits.Instinct,
		Presence:          dhDefaults.Traits.Presence,
		Knowledge:         dhDefaults.Traits.Knowledge,
		Experiences:       experiences,
		StartingWeaponIDs: []string{},
		DomainCardIDs:     []string{},
	})
}

// resolveCharacterIdentitySnapshot returns character identity defaults for one
// controller participant snapshot. Unassigned controllers intentionally use the
// blank avatar set and empty pronouns.
func (c characterApplication) resolveCharacterIdentitySnapshot(
	ctx context.Context,
	campaignID string,
	participantID string,
) (characterIdentitySnapshot, error) {
	snapshot := characterIdentitySnapshot{
		avatarSetID:   assetcatalog.AvatarSetBlankV1,
		avatarAssetID: "",
		pronouns:      "",
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
	snapshot.pronouns = strings.TrimSpace(participantRecord.Pronouns)
	return snapshot, nil
}
