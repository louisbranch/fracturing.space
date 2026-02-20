package game

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type characterApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
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
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
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

	characterID, err := c.idGenerator()
	if err != nil {
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "generate character id: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return storage.CharacterRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := character.CreatePayload{
		CharacterID: characterID,
		Name:        name,
		Kind:        in.GetKind().String(),
		Notes:       notes,
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
		c.stores.Domain,
		applier,
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("character.create"),
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{},
	)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	created, err := c.stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "load character: %v", err)
	}

	// Get Daggerheart defaults for the character kind
	kindStr := "PC"
	if created.Kind == character.KindNPC {
		kindStr = "NPC"
	}
	dhDefaults := daggerheart.GetProfileDefaults(kindStr)

	reqID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)

	experiencesPayload := make([]map[string]any, 0, len(dhDefaults.Experiences))
	for _, experience := range dhDefaults.Experiences {
		experiencesPayload = append(experiencesPayload, map[string]any{
			"name":     experience.Name,
			"modifier": experience.Modifier,
		})
	}

	systemProfile := map[string]any{
		"daggerheart": map[string]any{
			"level":            dhDefaults.Level,
			"hp_max":           dhDefaults.HpMax,
			"stress_max":       dhDefaults.StressMax,
			"evasion":          dhDefaults.Evasion,
			"major_threshold":  dhDefaults.MajorThreshold,
			"severe_threshold": dhDefaults.SevereThreshold,
			"proficiency":      dhDefaults.Proficiency,
			"armor_score":      dhDefaults.ArmorScore,
			"armor_max":        dhDefaults.ArmorMax,
			"agility":          dhDefaults.Traits.Agility,
			"strength":         dhDefaults.Traits.Strength,
			"finesse":          dhDefaults.Traits.Finesse,
			"instinct":         dhDefaults.Traits.Instinct,
			"presence":         dhDefaults.Traits.Presence,
			"knowledge":        dhDefaults.Traits.Knowledge,
			"experiences":      experiencesPayload,
		},
	}
	projectionApplier := c.stores.Applier()
	commandPayload := character.ProfileUpdatePayload{
		CharacterID:   created.ID,
		SystemProfile: systemProfile,
	}
	commandPayloadJSON, err := json.Marshal(commandPayload)
	if err != nil {
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "encode profile payload: %v", err)
	}

	profileCommandActorType := command.ActorTypeSystem
	if actorID != "" {
		profileCommandActorType = command.ActorTypeGM
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		projectionApplier,
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("character.profile_update"),
			ActorType:    profileCommandActorType,
			ActorID:      actorID,
			RequestID:    reqID,
			InvocationID: invocationID,
			EntityType:   "character",
			EntityID:     created.ID,
			PayloadJSON:  commandPayloadJSON,
		},
		domainCommandApplyOptions{
			applyErrMessage: "apply profile event",
		},
	)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	hpAfter := dhDefaults.HpMax
	hopeAfter := daggerheart.HopeDefault
	hopeMaxAfter := daggerheart.HopeMax
	stressAfter := daggerheart.StressDefault
	armorAfter := 0
	lifeStateAfter := daggerheart.LifeStateAlive
	statePayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:    created.ID,
		HPAfter:        &hpAfter,
		HopeAfter:      &hopeAfter,
		HopeMaxAfter:   &hopeMaxAfter,
		StressAfter:    &stressAfter,
		ArmorAfter:     &armorAfter,
		LifeStateAfter: &lifeStateAfter,
	}
	stateJSON, err := json.Marshal(statePayload)
	if err != nil {
		return storage.CharacterRecord{}, status.Errorf(codes.Internal, "encode state payload: %v", err)
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		c.stores.Applier(),
		command.Command{
			CampaignID:    campaignID,
			Type:          command.Type("sys.daggerheart.character_state.patch"),
			ActorType:     profileCommandActorType,
			ActorID:       actorID,
			SessionID:     grpcmeta.SessionIDFromContext(ctx),
			RequestID:     reqID,
			InvocationID:  invocationID,
			EntityType:    "character",
			EntityID:      created.ID,
			SystemID:      daggerheart.SystemID,
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   stateJSON,
		},
		domainCommandApplyOptions{
			requireEvents:   true,
			missingEventMsg: "character state patch did not emit an event",
			applyErrMessage: "apply state event",
		},
	)
	if err != nil {
		return storage.CharacterRecord{}, err
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
	if len(fields) == 0 {
		return storage.CharacterRecord{}, status.Error(codes.InvalidArgument, "at least one field must be provided")
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return storage.CharacterRecord{}, status.Error(codes.Internal, "domain engine is not configured")
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
		c.stores.Domain,
		applier,
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("character.update"),
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{},
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

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	reason := strings.TrimSpace(in.GetReason())
	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return storage.CharacterRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
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
		c.stores.Domain,
		applier,
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("character.delete"),
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{
			applyErrMessage: "apply event",
		},
	)
	if err != nil {
		return storage.CharacterRecord{}, err
	}

	return ch, nil
}

func (c characterApplication) SetDefaultControl(ctx context.Context, campaignID string, in *campaignv1.SetDefaultControlRequest) (string, string, error) {
	if _, err := c.stores.Campaign.Get(ctx, campaignID); err != nil {
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
	if participantID != "" {
		if c.stores.Participant == nil {
			return "", "", status.Error(codes.Internal, "participant store is not configured")
		}
		if _, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID); err != nil {
			return "", "", err
		}
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return "", "", status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := character.UpdatePayload{
		CharacterID: characterID,
		Fields: map[string]string{
			"participant_id": participantID,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", "", status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		applier,
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("character.update"),
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{
			applyErrMessage: "apply event",
		},
	)
	if err != nil {
		return "", "", err
	}

	return characterID, participantID, nil
}

func (c characterApplication) PatchCharacterProfile(ctx context.Context, campaignID string, in *campaignv1.PatchCharacterProfileRequest) (string, storage.DaggerheartCharacterProfile, error) {
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "character id is required")
	}

	dhProfile, err := c.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		return "", storage.DaggerheartCharacterProfile{}, err
	}

	// Apply Daggerheart-specific patches (including hp_max)
	if dhPatch := in.GetDaggerheart(); dhPatch != nil {
		// Validate level (plain int32: 0 is not valid)
		if dhPatch.Level < 0 {
			return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "level must be non-negative")
		}
		if dhPatch.Level > 0 {
			if err := daggerheart.ValidateLevel(int(dhPatch.Level)); err != nil {
				return "", storage.DaggerheartCharacterProfile{}, err
			}
			dhProfile.Level = int(dhPatch.Level)
		}

		// Validate hp_max (plain int32: 0 is not valid)
		if dhPatch.HpMax < 0 {
			return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "hp_max must be non-negative")
		}
		if dhPatch.HpMax > 0 {
			if dhPatch.HpMax > daggerheart.HPMaxCap {
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
			if val > daggerheart.StressMaxCap {
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
			if val < 0 || val > daggerheart.ArmorMaxCap {
				return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "armor_max must be in range 0..12")
			}
			dhProfile.ArmorMax = val
		}

		// Validate and apply traits (wrapper types allow nil-checking)
		if dhPatch.GetAgility() != nil {
			if err := daggerheart.ValidateTrait("agility", int(dhPatch.GetAgility().GetValue())); err != nil {
				return "", storage.DaggerheartCharacterProfile{}, err
			}
			dhProfile.Agility = int(dhPatch.GetAgility().GetValue())
		}
		if dhPatch.GetStrength() != nil {
			if err := daggerheart.ValidateTrait("strength", int(dhPatch.GetStrength().GetValue())); err != nil {
				return "", storage.DaggerheartCharacterProfile{}, err
			}
			dhProfile.Strength = int(dhPatch.GetStrength().GetValue())
		}
		if dhPatch.GetFinesse() != nil {
			if err := daggerheart.ValidateTrait("finesse", int(dhPatch.GetFinesse().GetValue())); err != nil {
				return "", storage.DaggerheartCharacterProfile{}, err
			}
			dhProfile.Finesse = int(dhPatch.GetFinesse().GetValue())
		}
		if dhPatch.GetInstinct() != nil {
			if err := daggerheart.ValidateTrait("instinct", int(dhPatch.GetInstinct().GetValue())); err != nil {
				return "", storage.DaggerheartCharacterProfile{}, err
			}
			dhProfile.Instinct = int(dhPatch.GetInstinct().GetValue())
		}
		if dhPatch.GetPresence() != nil {
			if err := daggerheart.ValidateTrait("presence", int(dhPatch.GetPresence().GetValue())); err != nil {
				return "", storage.DaggerheartCharacterProfile{}, err
			}
			dhProfile.Presence = int(dhPatch.GetPresence().GetValue())
		}
		if dhPatch.GetKnowledge() != nil {
			if err := daggerheart.ValidateTrait("knowledge", int(dhPatch.GetKnowledge().GetValue())); err != nil {
				return "", storage.DaggerheartCharacterProfile{}, err
			}
			dhProfile.Knowledge = int(dhPatch.GetKnowledge().GetValue())
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
			dhProfile.Level = daggerheart.PCLevelDefault
		}
		dhProfile.MajorThreshold, dhProfile.SevereThreshold = daggerheart.DeriveThresholds(
			dhProfile.Level,
			dhProfile.ArmorScore,
			dhProfile.MajorThreshold,
			dhProfile.SevereThreshold,
		)

		experiences := make([]daggerheart.Experience, 0, len(dhProfile.Experiences))
		for _, experience := range dhProfile.Experiences {
			experiences = append(experiences, daggerheart.Experience{
				Name:     experience.Name,
				Modifier: experience.Modifier,
			})
		}
		if err := daggerheart.ValidateProfile(
			dhProfile.Level,
			dhProfile.HpMax,
			dhProfile.StressMax,
			dhProfile.Evasion,
			dhProfile.MajorThreshold,
			dhProfile.SevereThreshold,
			dhProfile.Proficiency,
			dhProfile.ArmorScore,
			dhProfile.ArmorMax,
			daggerheart.Traits{
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
			"level":            dhProfile.Level,
			"hp_max":           dhProfile.HpMax,
			"stress_max":       dhProfile.StressMax,
			"evasion":          dhProfile.Evasion,
			"major_threshold":  dhProfile.MajorThreshold,
			"severe_threshold": dhProfile.SevereThreshold,
			"proficiency":      dhProfile.Proficiency,
			"armor_score":      dhProfile.ArmorScore,
			"armor_max":        dhProfile.ArmorMax,
			"agility":          dhProfile.Agility,
			"strength":         dhProfile.Strength,
			"finesse":          dhProfile.Finesse,
			"instinct":         dhProfile.Instinct,
			"presence":         dhProfile.Presence,
			"knowledge":        dhProfile.Knowledge,
			"experiences":      experiencesPayload,
		},
	}
	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return "", storage.DaggerheartCharacterProfile{}, status.Error(codes.Internal, "domain engine is not configured")
	}
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
		c.stores.Domain,
		applier,
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("character.profile_update"),
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  commandPayloadJSON,
		},
		domainCommandApplyOptions{},
	)
	if err != nil {
		return "", storage.DaggerheartCharacterProfile{}, err
	}

	return characterID, dhProfile, nil
}
