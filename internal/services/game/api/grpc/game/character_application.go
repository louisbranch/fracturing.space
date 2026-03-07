package game

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	assetcatalog "github.com/louisbranch/fracturing.space/internal/platform/assets/catalog"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type characterApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

type characterIdentitySnapshot struct {
	avatarSetID   string
	avatarAssetID string
	pronouns      string
}

func newCharacterApplication(service *CharacterService) characterApplication {
	app := characterApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
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
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "generate character id: %v", err)
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

	// Get Daggerheart defaults for the character kind
	kindStr := "PC"
	if kind == character.KindNPC {
		kindStr = "NPC"
	}
	dhDefaults := daggerheartprofile.GetDefaults(kindStr)

	experiencesPayload := make([]map[string]any, 0, len(dhDefaults.Experiences))
	for _, experience := range dhDefaults.Experiences {
		experiencesPayload = append(experiencesPayload, map[string]any{
			"name":     experience.Name,
			"modifier": experience.Modifier,
		})
	}

	systemProfile := map[string]any{
		"daggerheart": map[string]any{
			"level":                   dhDefaults.Level,
			"hp_max":                  dhDefaults.HpMax,
			"stress_max":              dhDefaults.StressMax,
			"evasion":                 dhDefaults.Evasion,
			"major_threshold":         dhDefaults.MajorThreshold,
			"severe_threshold":        dhDefaults.SevereThreshold,
			"proficiency":             dhDefaults.Proficiency,
			"armor_score":             dhDefaults.ArmorScore,
			"armor_max":               dhDefaults.ArmorMax,
			"agility":                 dhDefaults.Traits.Agility,
			"strength":                dhDefaults.Traits.Strength,
			"finesse":                 dhDefaults.Traits.Finesse,
			"instinct":                dhDefaults.Traits.Instinct,
			"presence":                dhDefaults.Traits.Presence,
			"knowledge":               dhDefaults.Traits.Knowledge,
			"experiences":             experiencesPayload,
			"class_id":                "",
			"subclass_id":             "",
			"ancestry_id":             "",
			"community_id":            "",
			"traits_assigned":         false,
			"details_recorded":        false,
			"starting_weapon_ids":     []string{},
			"starting_armor_id":       "",
			"starting_potion_item_id": "",
			"background":              "",
			"domain_card_ids":         []string{},
			"connections":             "",
		},
	}
	workflowPayload := character.CreateWithProfilePayload{
		Create: character.CreatePayload{
			CharacterID:        characterID,
			OwnerParticipantID: strings.TrimSpace(policyActor.ID),
			Name:               name,
			Kind:               in.GetKind().String(),
			Notes:              notes,
			AvatarSetID:        avatarSetID,
			AvatarAssetID:      avatarAssetID,
			ParticipantID:      defaultParticipantID,
			Pronouns:           pronouns,
			Aliases:            append([]string(nil), in.GetAliases()...),
		},
		SystemProfile: systemProfile,
	}
	workflowPayloadJSON, err := json.Marshal(workflowPayload)
	if err != nil {
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "encode create workflow payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores,
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
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "load character: %v", err)
	}
	return created, nil
}

func (c characterApplication) UpdateCharacter(ctx context.Context, campaignID string, in *campaignv1.UpdateCharacterRequest) (storage.CharacterRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CharacterRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CharacterRecord{}, err
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return storage.CharacterRecord{}, status.Error(codes.InvalidArgument, "character id is required")
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
		ch.Name = trimmed
		fields["name"] = trimmed
	}
	if in.GetKind() != campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED {
		kind := characterKindFromProto(in.GetKind())
		if kind == character.KindUnspecified {
			return storage.CharacterRecord{}, status.Error(codes.InvalidArgument, "character kind is invalid")
		}
		ch.Kind = kind
		fields["kind"] = in.GetKind().String()
	}
	if notes := in.GetNotes(); notes != nil {
		ch.Notes = strings.TrimSpace(notes.GetValue())
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
		policyActor, err = requirePolicyActor(ctx, c.stores, domainauthz.CapabilityTransferCharacterOwnership, campaignRecord)
		if err != nil {
			return storage.CharacterRecord{}, err
		}
	} else {
		policyActor, err = requireCharacterMutationPolicy(
			ctx,
			c.stores,
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
	applier := c.stores.Applier()
	payloadFields := make(map[string]string, len(fields))
	for key, value := range fields {
		stringValue, ok := value.(string)
		if !ok {
			return storage.CharacterRecord{}, status.Errorf(codes.Internal, "character update field %s must be string", key)
		}
		payloadFields[key] = stringValue
	}
	payload := character.UpdatePayload{
		CharacterID: characterID,
		Fields:      payloadFields,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
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
		domainwrite.Options{},
	)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	updated, err := c.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "load character: %v", err)
	}

	return updated, nil
}

func (c characterApplication) DeleteCharacter(ctx context.Context, campaignID string, in *campaignv1.DeleteCharacterRequest) (storage.CharacterRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CharacterRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CharacterRecord{}, err
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return storage.CharacterRecord{}, status.Error(codes.InvalidArgument, "character id is required")
	}

	ch, err := c.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return storage.CharacterRecord{}, err
	}
	policyActor, err := requireCharacterMutationPolicy(
		ctx,
		c.stores,
		campaignRecord,
		characterID,
	)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		actorID = strings.TrimSpace(policyActor.ID)
	}
	reason := strings.TrimSpace(in.GetReason())
	applier := c.stores.Applier()
	payload := character.DeletePayload{
		CharacterID: characterID,
		Reason:      reason,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCharacterDelete,
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
		return storage.CharacterRecord{}, err
	}

	return ch, nil
}

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
		CharacterID: characterID,
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

func (c characterApplication) PatchCharacterProfile(ctx context.Context, campaignID string, in *campaignv1.PatchCharacterProfileRequest) (string, storage.DaggerheartCharacterProfile, error) {
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "character id is required")
	}

	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", storage.DaggerheartCharacterProfile{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return "", storage.DaggerheartCharacterProfile{}, err
	}

	dhProfile, err := c.stores.SystemStores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return "", storage.DaggerheartCharacterProfile{}, err
	}

	// Apply Daggerheart-specific patches (including hp_max)
	if dhPatch := in.GetDaggerheart(); dhPatch != nil {
		if err := rejectDaggerheartCreationWorkflowPatchFields(dhPatch); err != nil {
			return "", storage.DaggerheartCharacterProfile{}, err
		}

		// Validate level (plain int32: 0 is not valid)
		if dhPatch.Level < 0 {
			return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "level must be non-negative")
		}
		if dhPatch.Level > 0 {
			if err := daggerheartprofile.ValidateLevel(int(dhPatch.Level)); err != nil {
				return "", storage.DaggerheartCharacterProfile{}, err
			}
			dhProfile.Level = int(dhPatch.Level)
		}

		// Validate hp_max (plain int32: 0 is not valid)
		if dhPatch.HpMax < 0 {
			return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "hp_max must be non-negative")
		}
		if dhPatch.HpMax > 0 {
			if dhPatch.HpMax > daggerheartprofile.HPMaxCap {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "hp_max must be in range 1..12")
			}
			dhProfile.HpMax = int(dhPatch.HpMax)
		}

		// Validate stress_max (wrapper type: nil means not provided)
		if dhPatch.GetStressMax() != nil {
			val := int(dhPatch.GetStressMax().GetValue())
			if val < 0 {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "stress_max must be non-negative")
			}
			if val > daggerheartprofile.StressMaxCap {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "stress_max must be in range 0..12")
			}
			dhProfile.StressMax = val
		}

		// Validate evasion (wrapper type: nil means not provided)
		if dhPatch.GetEvasion() != nil {
			val := int(dhPatch.GetEvasion().GetValue())
			if val < 0 {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "evasion must be non-negative")
			}
			dhProfile.Evasion = val
		}

		// Validate major_threshold (wrapper type: nil means not provided)
		if dhPatch.GetMajorThreshold() != nil {
			val := int(dhPatch.GetMajorThreshold().GetValue())
			if val < 0 {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "major_threshold must be non-negative")
			}
			dhProfile.MajorThreshold = val
		}

		// Validate severe_threshold (wrapper type: nil means not provided)
		if dhPatch.GetSevereThreshold() != nil {
			val := int(dhPatch.GetSevereThreshold().GetValue())
			if val < 0 {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "severe_threshold must be non-negative")
			}
			dhProfile.SevereThreshold = val
		}

		// Validate proficiency (wrapper type: nil means not provided)
		if dhPatch.GetProficiency() != nil {
			val := int(dhPatch.GetProficiency().GetValue())
			if val < 0 {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "proficiency must be non-negative")
			}
			dhProfile.Proficiency = val
		}

		// Validate armor_score (wrapper type: nil means not provided)
		if dhPatch.GetArmorScore() != nil {
			val := int(dhPatch.GetArmorScore().GetValue())
			if val < 0 {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "armor_score must be non-negative")
			}
			dhProfile.ArmorScore = val
		}

		// Validate armor_max (wrapper type: nil means not provided)
		if dhPatch.GetArmorMax() != nil {
			val := int(dhPatch.GetArmorMax().GetValue())
			if val < 0 || val > daggerheartprofile.ArmorMaxCap {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "armor_max must be in range 0..12")
			}
			dhProfile.ArmorMax = val
		}

		if len(dhPatch.GetExperiences()) > 0 {
			experiences := make([]storage.DaggerheartExperience, 0, len(dhPatch.GetExperiences()))
			for _, experience := range dhPatch.GetExperiences() {
				if strings.TrimSpace(experience.GetName()) == "" {
					return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "experience name is required")
				}
				experiences = append(experiences, storage.DaggerheartExperience{
					Name:     experience.GetName(),
					Modifier: int(experience.GetModifier()),
				})
			}
			dhProfile.Experiences = experiences
		}
		if dhProfile.Level == 0 {
			dhProfile.Level = daggerheartprofile.PCLevelDefault
		}
		dhProfile.MajorThreshold, dhProfile.SevereThreshold = daggerheartprofile.DeriveThresholds(
			dhProfile.Level,
			dhProfile.ArmorScore,
			dhProfile.MajorThreshold,
			dhProfile.SevereThreshold,
		)

		experiences := make([]daggerheartprofile.Experience, 0, len(dhProfile.Experiences))
		for _, experience := range dhProfile.Experiences {
			experiences = append(experiences, daggerheartprofile.Experience{
				Name:     experience.Name,
				Modifier: experience.Modifier,
			})
		}
		if err := daggerheartprofile.Validate(
			dhProfile.Level,
			dhProfile.HpMax,
			dhProfile.StressMax,
			dhProfile.Evasion,
			dhProfile.MajorThreshold,
			dhProfile.SevereThreshold,
			dhProfile.Proficiency,
			dhProfile.ArmorScore,
			dhProfile.ArmorMax,
			daggerheartprofile.Traits{
				Agility:   dhProfile.Agility,
				Strength:  dhProfile.Strength,
				Finesse:   dhProfile.Finesse,
				Instinct:  dhProfile.Instinct,
				Presence:  dhProfile.Presence,
				Knowledge: dhProfile.Knowledge,
			},
			experiences,
		); err != nil {
			return "", storage.DaggerheartCharacterProfile{}, err
		}
	}

	experiencesPayload := make([]map[string]any, 0, len(dhProfile.Experiences))
	for _, experience := range dhProfile.Experiences {
		experiencesPayload = append(experiencesPayload, map[string]any{
			"name":     experience.Name,
			"modifier": experience.Modifier,
		})
	}

	systemProfile := map[string]any{
		"daggerheart": map[string]any{
			"level":                   dhProfile.Level,
			"hp_max":                  dhProfile.HpMax,
			"stress_max":              dhProfile.StressMax,
			"evasion":                 dhProfile.Evasion,
			"major_threshold":         dhProfile.MajorThreshold,
			"severe_threshold":        dhProfile.SevereThreshold,
			"proficiency":             dhProfile.Proficiency,
			"armor_score":             dhProfile.ArmorScore,
			"armor_max":               dhProfile.ArmorMax,
			"agility":                 dhProfile.Agility,
			"strength":                dhProfile.Strength,
			"finesse":                 dhProfile.Finesse,
			"instinct":                dhProfile.Instinct,
			"presence":                dhProfile.Presence,
			"knowledge":               dhProfile.Knowledge,
			"experiences":             experiencesPayload,
			"class_id":                dhProfile.ClassID,
			"subclass_id":             dhProfile.SubclassID,
			"ancestry_id":             dhProfile.AncestryID,
			"community_id":            dhProfile.CommunityID,
			"traits_assigned":         dhProfile.TraitsAssigned,
			"details_recorded":        dhProfile.DetailsRecorded,
			"starting_weapon_ids":     append([]string(nil), dhProfile.StartingWeaponIDs...),
			"starting_armor_id":       dhProfile.StartingArmorID,
			"starting_potion_item_id": dhProfile.StartingPotionItemID,
			"background":              dhProfile.Background,
			"domain_card_ids":         append([]string(nil), dhProfile.DomainCardIDs...),
			"connections":             dhProfile.Connections,
		},
	}
	policyActor, err := requireCharacterMutationPolicy(
		ctx,
		c.stores,
		campaignRecord,
		characterID,
	)
	if err != nil {
		return "", storage.DaggerheartCharacterProfile{}, err
	}
	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		actorID = strings.TrimSpace(policyActor.ID)
	}
	applier := c.stores.Applier()
	commandPayload := character.ProfileUpdatePayload{
		CharacterID:   characterID,
		SystemProfile: systemProfile,
	}
	commandPayloadJSON, err := json.Marshal(commandPayload)
	if err != nil {
		return "", storage.DaggerheartCharacterProfile{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCharacterProfileUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  commandPayloadJSON,
		}),
		domainwrite.Options{},
	)
	if err != nil {
		return "", storage.DaggerheartCharacterProfile{}, err
	}

	return characterID, dhProfile, nil
}

// rejectDaggerheartCreationWorkflowPatchFields enforces the single creation
// pipeline policy by preventing workflow-field mutation through profile patch.
func rejectDaggerheartCreationWorkflowPatchFields(patch *daggerheartv1.DaggerheartProfile) error {
	if patch == nil {
		return nil
	}
	if patch.GetAgility() == nil && patch.GetStrength() == nil && patch.GetFinesse() == nil &&
		patch.GetInstinct() == nil && patch.GetPresence() == nil && patch.GetKnowledge() == nil &&
		strings.TrimSpace(patch.GetClassId()) == "" && strings.TrimSpace(patch.GetSubclassId()) == "" &&
		strings.TrimSpace(patch.GetAncestryId()) == "" && strings.TrimSpace(patch.GetCommunityId()) == "" &&
		patch.GetTraitsAssigned() == nil && patch.GetDetailsRecorded() == nil &&
		len(patch.GetStartingWeaponIds()) == 0 && strings.TrimSpace(patch.GetStartingArmorId()) == "" &&
		strings.TrimSpace(patch.GetStartingPotionItemId()) == "" &&
		strings.TrimSpace(patch.GetBackground()) == "" &&
		len(patch.GetExperiences()) == 0 && len(patch.GetDomainCardIds()) == 0 &&
		strings.TrimSpace(patch.GetConnections()) == "" {
		return nil
	}
	return status.Error(codes.InvalidArgument, "daggerheart creation workflow fields must be updated via ApplyCharacterCreationStep or ApplyCharacterCreationWorkflow")
}
