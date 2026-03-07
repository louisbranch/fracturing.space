package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartgrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

	dhProfile, err = applyDaggerheartProfilePatch(dhProfile, in.GetDaggerheart())
	if err != nil {
		return "", storage.DaggerheartCharacterProfile{}, err
	}

	if err := c.executeCharacterProfileUpdate(ctx, campaignRecord, characterID, daggerheartgrpc.SystemProfileMap(dhProfile)); err != nil {
		return "", storage.DaggerheartCharacterProfile{}, err
	}

	return characterID, dhProfile, nil
}

// applyDaggerheartProfilePatch validates mutable Daggerheart profile fields and
// applies accepted values to a copied profile.
func applyDaggerheartProfilePatch(current storage.DaggerheartCharacterProfile, patch *daggerheartv1.DaggerheartProfile) (storage.DaggerheartCharacterProfile, error) {
	if patch == nil {
		return current, nil
	}
	if err := rejectDaggerheartCreationWorkflowPatchFields(patch); err != nil {
		return storage.DaggerheartCharacterProfile{}, err
	}

	if patch.Level < 0 {
		return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "level must be non-negative")
	}
	if patch.Level > 0 {
		if err := daggerheartprofile.ValidateLevel(int(patch.Level)); err != nil {
			return storage.DaggerheartCharacterProfile{}, err
		}
		current.Level = int(patch.Level)
	}

	if patch.HpMax < 0 {
		return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "hp_max must be non-negative")
	}
	if patch.HpMax > 0 {
		if patch.HpMax > daggerheartprofile.HPMaxCap {
			return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "hp_max must be in range 1..12")
		}
		current.HpMax = int(patch.HpMax)
	}

	if patch.GetStressMax() != nil {
		val := int(patch.GetStressMax().GetValue())
		if val < 0 {
			return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "stress_max must be non-negative")
		}
		if val > daggerheartprofile.StressMaxCap {
			return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "stress_max must be in range 0..12")
		}
		current.StressMax = val
	}

	if patch.GetEvasion() != nil {
		val := int(patch.GetEvasion().GetValue())
		if val < 0 {
			return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "evasion must be non-negative")
		}
		current.Evasion = val
	}

	if patch.GetMajorThreshold() != nil {
		val := int(patch.GetMajorThreshold().GetValue())
		if val < 0 {
			return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "major_threshold must be non-negative")
		}
		current.MajorThreshold = val
	}

	if patch.GetSevereThreshold() != nil {
		val := int(patch.GetSevereThreshold().GetValue())
		if val < 0 {
			return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "severe_threshold must be non-negative")
		}
		current.SevereThreshold = val
	}

	if patch.GetProficiency() != nil {
		val := int(patch.GetProficiency().GetValue())
		if val < 0 {
			return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "proficiency must be non-negative")
		}
		current.Proficiency = val
	}

	if patch.GetArmorScore() != nil {
		val := int(patch.GetArmorScore().GetValue())
		if val < 0 {
			return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "armor_score must be non-negative")
		}
		current.ArmorScore = val
	}

	if patch.GetArmorMax() != nil {
		val := int(patch.GetArmorMax().GetValue())
		if val < 0 || val > daggerheartprofile.ArmorMaxCap {
			return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "armor_max must be in range 0..12")
		}
		current.ArmorMax = val
	}

	if len(patch.GetExperiences()) > 0 {
		experiences := make([]storage.DaggerheartExperience, 0, len(patch.GetExperiences()))
		for _, experience := range patch.GetExperiences() {
			if strings.TrimSpace(experience.GetName()) == "" {
				return storage.DaggerheartCharacterProfile{}, status.Error(codes.InvalidArgument, "experience name is required")
			}
			experiences = append(experiences, storage.DaggerheartExperience{
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
		return storage.DaggerheartCharacterProfile{}, err
	}
	return current, nil
}

func validatePatchedDaggerheartProfile(current storage.DaggerheartCharacterProfile) error {
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
