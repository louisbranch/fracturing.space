package charactertransport

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/characterworkflow"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c characterApplication) PatchCharacterProfile(ctx context.Context, campaignID string, in *campaignv1.PatchCharacterProfileRequest) (string, projectionstore.DaggerheartCharacterProfile, error) {
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return "", projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "character id is required")
	}

	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return "", projectionstore.DaggerheartCharacterProfile{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return "", projectionstore.DaggerheartCharacterProfile{}, err
	}

	dhProfile, err := c.stores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return "", projectionstore.DaggerheartCharacterProfile{}, err
		}
		record, recordErr := c.stores.Character.GetCharacter(ctx, campaignID, characterID)
		if recordErr != nil {
			return "", projectionstore.DaggerheartCharacterProfile{}, recordErr
		}
		dhProfile = defaultDaggerheartStorageProfile(campaignID, characterID, record.Kind)
	}

	dhProfile, err = applyDaggerheartProfilePatch(dhProfile, in.GetDaggerheart())
	if err != nil {
		return "", projectionstore.DaggerheartCharacterProfile{}, err
	}

	if err := c.executeDaggerheartProfileReplace(ctx, characterworkflow.CampaignContext{
		ID:     campaignRecord.ID,
		System: handler.SystemIDFromCampaignRecord(campaignRecord),
		Status: campaignRecord.Status,
	}, characterID, daggerheart.CharacterProfileFromStorage(dhProfile)); err != nil {
		return "", projectionstore.DaggerheartCharacterProfile{}, err
	}

	return characterID, dhProfile, nil
}

func defaultDaggerheartStorageProfile(campaignID, characterID string, kind character.Kind) projectionstore.DaggerheartCharacterProfile {
	kindLabel := "PC"
	if kind == character.KindNPC {
		kindLabel = "NPC"
	}
	defaults := daggerheartprofile.GetDefaults(kindLabel)

	return projectionstore.DaggerheartCharacterProfile{
		CampaignID:        campaignID,
		CharacterID:       characterID,
		Level:             defaults.Level,
		HpMax:             defaults.HpMax,
		StressMax:         defaults.StressMax,
		Evasion:           defaults.Evasion,
		MajorThreshold:    defaults.MajorThreshold,
		SevereThreshold:   defaults.SevereThreshold,
		Proficiency:       defaults.Proficiency,
		ArmorScore:        defaults.ArmorScore,
		ArmorMax:          defaults.ArmorMax,
		Agility:           defaults.Traits.Agility,
		Strength:          defaults.Traits.Strength,
		Finesse:           defaults.Traits.Finesse,
		Instinct:          defaults.Traits.Instinct,
		Presence:          defaults.Traits.Presence,
		Knowledge:         defaults.Traits.Knowledge,
		Experiences:       []projectionstore.DaggerheartExperience{},
		StartingWeaponIDs: []string{},
		DomainCardIDs:     []string{},
	}
}

// applyDaggerheartProfilePatch validates mutable Daggerheart profile fields and
// applies accepted values to a copied profile.
func applyDaggerheartProfilePatch(current projectionstore.DaggerheartCharacterProfile, patch *daggerheartv1.DaggerheartProfile) (projectionstore.DaggerheartCharacterProfile, error) {
	if patch == nil {
		return current, nil
	}
	if err := rejectDaggerheartCreationWorkflowPatchFields(patch); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, err
	}

	if patch.Level < 0 {
		return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "level must be non-negative")
	}
	if patch.Level > 0 {
		if err := daggerheartprofile.ValidateLevel(int(patch.Level)); err != nil {
			return projectionstore.DaggerheartCharacterProfile{}, err
		}
		current.Level = int(patch.Level)
	}

	if patch.HpMax < 0 {
		return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "hp_max must be non-negative")
	}
	if patch.HpMax > 0 {
		if patch.HpMax > daggerheartprofile.HPMaxCap {
			return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "hp_max must be in range 1..12")
		}
		current.HpMax = int(patch.HpMax)
	}

	if patch.GetStressMax() != nil {
		val := int(patch.GetStressMax().GetValue())
		if val < 0 {
			return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "stress_max must be non-negative")
		}
		if val > daggerheartprofile.StressMaxCap {
			return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "stress_max must be in range 0..12")
		}
		current.StressMax = val
	}

	if patch.GetEvasion() != nil {
		val := int(patch.GetEvasion().GetValue())
		if val < 0 {
			return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "evasion must be non-negative")
		}
		current.Evasion = val
	}

	if patch.GetMajorThreshold() != nil {
		val := int(patch.GetMajorThreshold().GetValue())
		if val < 0 {
			return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "major_threshold must be non-negative")
		}
		current.MajorThreshold = val
	}

	if patch.GetSevereThreshold() != nil {
		val := int(patch.GetSevereThreshold().GetValue())
		if val < 0 {
			return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "severe_threshold must be non-negative")
		}
		current.SevereThreshold = val
	}

	if patch.GetProficiency() != nil {
		val := int(patch.GetProficiency().GetValue())
		if val < 0 {
			return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "proficiency must be non-negative")
		}
		current.Proficiency = val
	}

	if patch.GetArmorScore() != nil {
		val := int(patch.GetArmorScore().GetValue())
		if val < 0 {
			return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "armor_score must be non-negative")
		}
		current.ArmorScore = val
	}

	if patch.GetArmorMax() != nil {
		val := int(patch.GetArmorMax().GetValue())
		if val < 0 || val > daggerheartprofile.ArmorMaxCap {
			return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "armor_max must be in range 0..12")
		}
		current.ArmorMax = val
	}

	if len(patch.GetExperiences()) > 0 {
		experiences := make([]projectionstore.DaggerheartExperience, 0, len(patch.GetExperiences()))
		for _, experience := range patch.GetExperiences() {
			if strings.TrimSpace(experience.GetName()) == "" {
				return projectionstore.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "experience name is required")
			}
			experiences = append(experiences, projectionstore.DaggerheartExperience{
				Name:     experience.GetName(),
				Modifier: int(experience.GetModifier()),
			})
		}
		current.Experiences = experiences
	}

	if current.Level == 0 {
		current.Level = daggerheartprofile.PCLevelDefault
	}
	current.MajorThreshold, current.SevereThreshold = daggerheartprofile.DeriveThresholds(
		current.Level,
		current.ArmorScore,
		current.MajorThreshold,
		current.SevereThreshold,
	)

	if err := validatePatchedDaggerheartProfile(current); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, err
	}
	return current, nil
}

func validatePatchedDaggerheartProfile(current projectionstore.DaggerheartCharacterProfile) error {
	experiences := make([]daggerheartprofile.Experience, 0, len(current.Experiences))
	for _, experience := range current.Experiences {
		experiences = append(experiences, daggerheartprofile.Experience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}

	return daggerheartprofile.Validate(
		current.Level,
		current.HpMax,
		current.StressMax,
		current.Evasion,
		current.MajorThreshold,
		current.SevereThreshold,
		current.Proficiency,
		current.ArmorScore,
		current.ArmorMax,
		daggerheartprofile.Traits{
			Agility:   current.Agility,
			Strength:  current.Strength,
			Finesse:   current.Finesse,
			Instinct:  current.Instinct,
			Presence:  current.Presence,
			Knowledge: current.Knowledge,
		},
		experiences,
	)
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
		len(patch.GetSubclassCreationRequirements()) == 0 && patch.GetHeritage() == nil && patch.GetCompanionSheet() == nil &&
		patch.GetTraitsAssigned() == nil && patch.GetDetailsRecorded() == nil &&
		len(patch.GetStartingWeaponIds()) == 0 && strings.TrimSpace(patch.GetStartingArmorId()) == "" &&
		strings.TrimSpace(patch.GetStartingPotionItemId()) == "" &&
		strings.TrimSpace(patch.GetBackground()) == "" &&
		strings.TrimSpace(patch.GetDescription()) == "" &&
		len(patch.GetExperiences()) == 0 && len(patch.GetDomainCardIds()) == 0 &&
		strings.TrimSpace(patch.GetConnections()) == "" {
		return nil
	}
	return status.Error(codes.InvalidArgument, "daggerheart creation workflow fields must be updated via ApplyCharacterCreationStep or ApplyCharacterCreationWorkflow")
}
