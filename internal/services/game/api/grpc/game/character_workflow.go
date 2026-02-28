package game

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// characterCreationStepProgress carries one step's completion state.
type characterCreationStepProgress struct {
	Step     int32
	Key      string
	Complete bool
}

// characterCreationProgress is a system-agnostic workflow progress shape.
type characterCreationProgress struct {
	Steps        []characterCreationStepProgress
	NextStep     int32
	Ready        bool
	UnmetReasons []string
}

// characterCreationWorkflowProvider defines system-specific workflow behavior
// behind a common CharacterService transport contract.
type characterCreationWorkflowProvider interface {
	GetProgress(ctx context.Context, app characterApplication, campaignRecord storage.CampaignRecord, characterID string) (characterCreationProgress, error)
	ApplyStep(ctx context.Context, app characterApplication, campaignRecord storage.CampaignRecord, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, characterCreationProgress, error)
	ApplyWorkflow(ctx context.Context, app characterApplication, campaignRecord storage.CampaignRecord, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, characterCreationProgress, error)
	Reset(ctx context.Context, app characterApplication, campaignRecord storage.CampaignRecord, characterID string) (characterCreationProgress, error)
}

var characterCreationWorkflowProviders = map[commonv1.GameSystem]characterCreationWorkflowProvider{
	commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART: daggerheartCreationWorkflowProvider{},
}

func characterCreationWorkflowProviderForSystem(system commonv1.GameSystem) (characterCreationWorkflowProvider, bool) {
	provider, ok := characterCreationWorkflowProviders[system]
	return provider, ok
}

func characterCreationProgressFromDaggerheart(progress daggerheart.CreationProgress) characterCreationProgress {
	steps := make([]characterCreationStepProgress, 0, len(progress.Steps))
	for _, step := range progress.Steps {
		steps = append(steps, characterCreationStepProgress{
			Step:     step.Step,
			Key:      step.Key,
			Complete: step.Complete,
		})
	}
	return characterCreationProgress{
		Steps:        steps,
		NextStep:     progress.NextStep,
		Ready:        progress.Ready,
		UnmetReasons: append([]string(nil), progress.UnmetReasons...),
	}
}

func (c characterApplication) workflowProviderForCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, characterCreationWorkflowProvider, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, nil, err
	}
	provider, ok := characterCreationWorkflowProviderForSystem(campaignRecord.System)
	if !ok {
		return storage.CampaignRecord{}, nil, status.Errorf(codes.Unimplemented, "character creation workflow is not supported for game system %s", campaignRecord.System.String())
	}
	return campaignRecord, provider, nil
}

func (c characterApplication) GetCharacterCreationProgress(ctx context.Context, campaignID, characterID string) (characterCreationProgress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return characterCreationProgress{}, err
	}
	return provider.GetProgress(ctx, c, campaignRecord, characterID)
}

func (c characterApplication) ApplyCharacterCreationStep(ctx context.Context, campaignID string, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, characterCreationProgress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return nil, characterCreationProgress{}, err
	}
	return provider.ApplyStep(ctx, c, campaignRecord, in)
}

func (c characterApplication) ApplyCharacterCreationWorkflow(ctx context.Context, campaignID string, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, characterCreationProgress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return nil, characterCreationProgress{}, err
	}
	return provider.ApplyWorkflow(ctx, c, campaignRecord, in)
}

func (c characterApplication) ResetCharacterCreationWorkflow(ctx context.Context, campaignID, characterID string) (characterCreationProgress, error) {
	campaignRecord, provider, err := c.workflowProviderForCampaign(ctx, campaignID)
	if err != nil {
		return characterCreationProgress{}, err
	}
	return provider.Reset(ctx, c, campaignRecord, characterID)
}

type daggerheartCreationWorkflowProvider struct{}

func (daggerheartCreationWorkflowProvider) GetProgress(ctx context.Context, app characterApplication, campaignRecord storage.CampaignRecord, characterID string) (characterCreationProgress, error) {
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpRead); err != nil {
		return characterCreationProgress{}, err
	}
	if err := requireReadPolicy(ctx, app.stores, campaignRecord); err != nil {
		return characterCreationProgress{}, err
	}

	if _, err := app.stores.Character.GetCharacter(ctx, campaignRecord.ID, characterID); err != nil {
		return characterCreationProgress{}, err
	}

	profile, err := app.stores.SystemStores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignRecord.ID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return characterCreationProgress{}, status.Errorf(codes.Internal, "get daggerheart profile: %v", err)
		}
		profile = storage.DaggerheartCharacterProfile{CampaignID: campaignRecord.ID, CharacterID: characterID}
	}

	progress := daggerheart.EvaluateCreationProgress(daggerheartCreationProfileFromStorage(profile))
	return characterCreationProgressFromDaggerheart(progress), nil
}

func (daggerheartCreationWorkflowProvider) ApplyStep(ctx context.Context, app characterApplication, campaignRecord storage.CampaignRecord, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, characterCreationProgress, error) {
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, characterCreationProgress{}, status.Error(codes.InvalidArgument, "character id is required")
	}
	stepInput := in.GetDaggerheart()
	if stepInput == nil {
		return nil, characterCreationProgress{}, status.Error(codes.InvalidArgument, "daggerheart step payload is required")
	}
	stepNumber, err := daggerheartCreationStepNumber(stepInput)
	if err != nil {
		return nil, characterCreationProgress{}, err
	}

	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, characterCreationProgress{}, err
	}

	characterRecord, err := app.stores.Character.GetCharacter(ctx, campaignRecord.ID, characterID)
	if err != nil {
		return nil, characterCreationProgress{}, err
	}

	profile, err := app.stores.SystemStores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignRecord.ID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, characterCreationProgress{}, status.Errorf(codes.Internal, "get daggerheart profile: %v", err)
		}
		profile = defaultDaggerheartProfileForCharacter(campaignRecord.ID, characterRecord)
	} else {
		profile = ensureDaggerheartProfileDefaults(profile, characterRecord.Kind)
	}

	currentProgress := daggerheart.EvaluateCreationProgress(daggerheartCreationProfileFromStorage(profile))
	if currentProgress.Ready {
		return nil, characterCreationProgress{}, status.Error(codes.FailedPrecondition, "character creation workflow is already complete")
	}
	if stepNumber != currentProgress.NextStep {
		return nil, characterCreationProgress{}, status.Errorf(
			codes.FailedPrecondition,
			"creation step %d is not allowed; expected step %d",
			stepNumber,
			currentProgress.NextStep,
		)
	}

	if err := app.applyDaggerheartCreationStepInput(ctx, &profile, stepInput); err != nil {
		return nil, characterCreationProgress{}, err
	}

	if err := validateDaggerheartProfile(profile); err != nil {
		return nil, characterCreationProgress{}, err
	}

	if err := app.executeCharacterProfileUpdate(ctx, campaignRecord, characterID, daggerheartSystemProfileMap(profile)); err != nil {
		return nil, characterCreationProgress{}, err
	}

	nextProgress := daggerheart.EvaluateCreationProgress(daggerheartCreationProfileFromStorage(profile))
	return daggerheartProfileToProto(campaignRecord.ID, characterID, profile), characterCreationProgressFromDaggerheart(nextProgress), nil
}

func (daggerheartCreationWorkflowProvider) ApplyWorkflow(ctx context.Context, app characterApplication, campaignRecord storage.CampaignRecord, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, characterCreationProgress, error) {
	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		return nil, characterCreationProgress{}, status.Error(codes.InvalidArgument, "character id is required")
	}
	workflowInput := in.GetDaggerheart()
	if workflowInput == nil {
		return nil, characterCreationProgress{}, status.Error(codes.InvalidArgument, "daggerheart workflow payload is required")
	}

	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, characterCreationProgress{}, err
	}

	characterRecord, err := app.stores.Character.GetCharacter(ctx, campaignRecord.ID, characterID)
	if err != nil {
		return nil, characterCreationProgress{}, err
	}

	profile, err := app.stores.SystemStores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignRecord.ID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, characterCreationProgress{}, status.Errorf(codes.Internal, "get daggerheart profile: %v", err)
		}
		profile = defaultDaggerheartProfileForCharacter(campaignRecord.ID, characterRecord)
	} else {
		profile = ensureDaggerheartProfileDefaults(profile, characterRecord.Kind)
	}

	profile = resetDaggerheartCreationWorkflowFields(profile)

	steps, err := daggerheartCreationStepSequenceFromWorkflowInput(workflowInput)
	if err != nil {
		return nil, characterCreationProgress{}, err
	}

	for idx, stepInput := range steps {
		expectedStep := int32(idx + 1)
		currentProgress := daggerheart.EvaluateCreationProgress(daggerheartCreationProfileFromStorage(profile))
		if currentProgress.Ready {
			return nil, characterCreationProgress{}, status.Error(codes.FailedPrecondition, "character creation workflow is already complete")
		}
		if currentProgress.NextStep != expectedStep {
			return nil, characterCreationProgress{}, status.Errorf(codes.FailedPrecondition, "creation step %d is not allowed; expected step %d", expectedStep, currentProgress.NextStep)
		}
		stepNumber, err := daggerheartCreationStepNumber(stepInput)
		if err != nil {
			return nil, characterCreationProgress{}, err
		}
		if stepNumber != expectedStep {
			return nil, characterCreationProgress{}, status.Errorf(codes.InvalidArgument, "daggerheart workflow payload contains out-of-order step %d at position %d", stepNumber, idx+1)
		}
		if err := app.applyDaggerheartCreationStepInput(ctx, &profile, stepInput); err != nil {
			return nil, characterCreationProgress{}, err
		}
		if err := validateDaggerheartProfile(profile); err != nil {
			return nil, characterCreationProgress{}, err
		}
	}

	finalProgress := daggerheart.EvaluateCreationProgress(daggerheartCreationProfileFromStorage(profile))
	if !finalProgress.Ready {
		if len(finalProgress.UnmetReasons) > 0 {
			return nil, characterCreationProgress{}, status.Errorf(codes.FailedPrecondition, "character creation workflow is incomplete: %s", finalProgress.UnmetReasons[0])
		}
		return nil, characterCreationProgress{}, status.Error(codes.FailedPrecondition, "character creation workflow is incomplete")
	}

	if err := app.executeCharacterProfileUpdate(ctx, campaignRecord, characterID, daggerheartSystemProfileMap(profile)); err != nil {
		return nil, characterCreationProgress{}, err
	}

	return daggerheartProfileToProto(campaignRecord.ID, characterID, profile), characterCreationProgressFromDaggerheart(finalProgress), nil
}

func (daggerheartCreationWorkflowProvider) Reset(ctx context.Context, app characterApplication, campaignRecord storage.CampaignRecord, characterID string) (characterCreationProgress, error) {
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return characterCreationProgress{}, err
	}

	if _, err := app.stores.Character.GetCharacter(ctx, campaignRecord.ID, characterID); err != nil {
		return characterCreationProgress{}, err
	}

	if err := app.executeCharacterProfileUpdate(ctx, campaignRecord, characterID, map[string]any{
		"daggerheart": map[string]any{"reset": true},
	}); err != nil {
		return characterCreationProgress{}, err
	}

	return characterCreationProgressFromDaggerheart(daggerheart.EvaluateCreationProgress(daggerheart.CreationProfile{})), nil
}

func daggerheartCreationStepSequenceFromWorkflowInput(input *daggerheartv1.DaggerheartCreationWorkflowInput) ([]*daggerheartv1.DaggerheartCreationStepInput, error) {
	if input == nil {
		return nil, status.Error(codes.InvalidArgument, "daggerheart workflow payload is required")
	}
	if input.GetClassSubclassInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "class_subclass_input is required")
	}
	if input.GetHeritageInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "heritage_input is required")
	}
	if input.GetTraitsInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "traits_input is required")
	}
	if input.GetDetailsInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "details_input is required")
	}
	if input.GetEquipmentInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "equipment_input is required")
	}
	if input.GetBackgroundInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "background_input is required")
	}
	if input.GetExperiencesInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "experiences_input is required")
	}
	if input.GetDomainCardsInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "domain_cards_input is required")
	}
	if input.GetConnectionsInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "connections_input is required")
	}
	return []*daggerheartv1.DaggerheartCreationStepInput{
		{Step: &daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput{ClassSubclassInput: input.GetClassSubclassInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{HeritageInput: input.GetHeritageInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_TraitsInput{TraitsInput: input.GetTraitsInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{DetailsInput: input.GetDetailsInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{EquipmentInput: input.GetEquipmentInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_BackgroundInput{BackgroundInput: input.GetBackgroundInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput{ExperiencesInput: input.GetExperiencesInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{DomainCardsInput: input.GetDomainCardsInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput{ConnectionsInput: input.GetConnectionsInput()}},
	}, nil
}

func resetDaggerheartCreationWorkflowFields(profile storage.DaggerheartCharacterProfile) storage.DaggerheartCharacterProfile {
	profile.ClassID = ""
	profile.SubclassID = ""
	profile.AncestryID = ""
	profile.CommunityID = ""
	profile.TraitsAssigned = false
	profile.DetailsRecorded = false
	profile.StartingWeaponIDs = nil
	profile.StartingArmorID = ""
	profile.StartingPotionItemID = ""
	profile.Background = ""
	profile.Experiences = nil
	profile.DomainCardIDs = nil
	profile.Connections = ""
	profile.Agility = 0
	profile.Strength = 0
	profile.Finesse = 0
	profile.Instinct = 0
	profile.Presence = 0
	profile.Knowledge = 0
	return profile
}

func (c characterApplication) applyDaggerheartCreationStepInput(ctx context.Context, profile *storage.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepInput) error {
	if c.stores.DaggerheartContent == nil {
		return status.Error(codes.Internal, "daggerheart content store is not configured")
	}

	switch step := input.GetStep().(type) {
	case *daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput:
		classID := strings.TrimSpace(step.ClassSubclassInput.GetClassId())
		if classID == "" {
			return status.Error(codes.InvalidArgument, "class_id is required")
		}
		subclassID := strings.TrimSpace(step.ClassSubclassInput.GetSubclassId())
		if subclassID == "" {
			return status.Error(codes.InvalidArgument, "subclass_id is required")
		}
		if _, err := c.stores.DaggerheartContent.GetDaggerheartClass(ctx, classID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "class_id %q is not found", classID)
			}
			return status.Errorf(codes.Internal, "get class: %v", err)
		}
		subclass, err := c.stores.DaggerheartContent.GetDaggerheartSubclass(ctx, subclassID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "subclass_id %q is not found", subclassID)
			}
			return status.Errorf(codes.Internal, "get subclass: %v", err)
		}
		if strings.TrimSpace(subclass.ClassID) != classID {
			return status.Errorf(codes.InvalidArgument, "subclass_id %q does not belong to class_id %q", subclassID, classID)
		}
		profile.ClassID = classID
		profile.SubclassID = subclassID
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_HeritageInput:
		ancestryID := strings.TrimSpace(step.HeritageInput.GetAncestryId())
		if ancestryID == "" {
			return status.Error(codes.InvalidArgument, "ancestry_id is required")
		}
		communityID := strings.TrimSpace(step.HeritageInput.GetCommunityId())
		if communityID == "" {
			return status.Error(codes.InvalidArgument, "community_id is required")
		}

		ancestry, err := c.stores.DaggerheartContent.GetDaggerheartHeritage(ctx, ancestryID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "ancestry_id %q is not found", ancestryID)
			}
			return status.Errorf(codes.Internal, "get ancestry heritage: %v", err)
		}
		if !strings.EqualFold(strings.TrimSpace(ancestry.Kind), "ancestry") {
			return status.Errorf(codes.InvalidArgument, "ancestry_id %q is not an ancestry heritage", ancestryID)
		}

		community, err := c.stores.DaggerheartContent.GetDaggerheartHeritage(ctx, communityID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "community_id %q is not found", communityID)
			}
			return status.Errorf(codes.Internal, "get community heritage: %v", err)
		}
		if !strings.EqualFold(strings.TrimSpace(community.Kind), "community") {
			return status.Errorf(codes.InvalidArgument, "community_id %q is not a community heritage", communityID)
		}

		profile.AncestryID = ancestryID
		profile.CommunityID = communityID
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_TraitsInput:
		traits := daggerheart.Traits{
			Agility:   int(step.TraitsInput.GetAgility()),
			Strength:  int(step.TraitsInput.GetStrength()),
			Finesse:   int(step.TraitsInput.GetFinesse()),
			Instinct:  int(step.TraitsInput.GetInstinct()),
			Presence:  int(step.TraitsInput.GetPresence()),
			Knowledge: int(step.TraitsInput.GetKnowledge()),
		}
		if err := daggerheart.ValidateTrait("agility", traits.Agility); err != nil {
			return err
		}
		if err := daggerheart.ValidateTrait("strength", traits.Strength); err != nil {
			return err
		}
		if err := daggerheart.ValidateTrait("finesse", traits.Finesse); err != nil {
			return err
		}
		if err := daggerheart.ValidateTrait("instinct", traits.Instinct); err != nil {
			return err
		}
		if err := daggerheart.ValidateTrait("presence", traits.Presence); err != nil {
			return err
		}
		if err := daggerheart.ValidateTrait("knowledge", traits.Knowledge); err != nil {
			return err
		}
		if err := daggerheart.ValidateCreationTraitDistribution(traits); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		profile.Agility = traits.Agility
		profile.Strength = traits.Strength
		profile.Finesse = traits.Finesse
		profile.Instinct = traits.Instinct
		profile.Presence = traits.Presence
		profile.Knowledge = traits.Knowledge
		profile.TraitsAssigned = true
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_DetailsInput:
		if strings.TrimSpace(profile.ClassID) == "" {
			return status.Error(codes.FailedPrecondition, "class must be selected before details")
		}
		class, err := c.stores.DaggerheartContent.GetDaggerheartClass(ctx, profile.ClassID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "class_id %q is not found", profile.ClassID)
			}
			return status.Errorf(codes.Internal, "get class: %v", err)
		}
		if class.StartingHP <= 0 {
			return status.Errorf(codes.InvalidArgument, "class_id %q has invalid starting hp", profile.ClassID)
		}
		if class.StartingEvasion <= 0 {
			return status.Errorf(codes.InvalidArgument, "class_id %q has invalid starting evasion", profile.ClassID)
		}
		profile.Level = daggerheart.PCLevelDefault
		profile.HpMax = class.StartingHP
		profile.StressMax = daggerheart.PCStressMax
		profile.Evasion = class.StartingEvasion
		profile.DetailsRecorded = true
		profile.MajorThreshold, profile.SevereThreshold = daggerheart.DeriveThresholds(
			profile.Level,
			profile.ArmorScore,
			daggerheart.PCMajorThreshold,
			daggerheart.PCSevereThreshold,
		)
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_EquipmentInput:
		weaponIDs := step.EquipmentInput.GetWeaponIds()
		if len(weaponIDs) == 0 {
			return status.Error(codes.InvalidArgument, "at least one weapon_id is required")
		}
		if len(weaponIDs) > 2 {
			return status.Error(codes.InvalidArgument, "at most two weapon_ids are allowed")
		}
		seenWeaponIDs := make(map[string]struct{}, len(weaponIDs))
		normalizedWeaponIDs := make([]string, 0, len(weaponIDs))
		primaryCount := 0
		secondaryCount := 0
		for _, weaponID := range weaponIDs {
			trimmedWeaponID := strings.TrimSpace(weaponID)
			if trimmedWeaponID == "" {
				return status.Error(codes.InvalidArgument, "weapon_ids must not contain empty values")
			}
			if _, seen := seenWeaponIDs[trimmedWeaponID]; seen {
				return status.Errorf(codes.InvalidArgument, "weapon_id %q is duplicated", trimmedWeaponID)
			}
			seenWeaponIDs[trimmedWeaponID] = struct{}{}
			weapon, err := c.stores.DaggerheartContent.GetDaggerheartWeapon(ctx, trimmedWeaponID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					return status.Errorf(codes.InvalidArgument, "weapon_id %q is not found", trimmedWeaponID)
				}
				return status.Errorf(codes.Internal, "get weapon: %v", err)
			}
			if weapon.Tier != 1 {
				return status.Errorf(codes.InvalidArgument, "weapon_id %q must be tier 1", trimmedWeaponID)
			}
			switch strings.ToLower(strings.TrimSpace(weapon.Category)) {
			case "primary":
				primaryCount++
			case "secondary":
				secondaryCount++
			default:
				return status.Errorf(codes.InvalidArgument, "weapon_id %q has unsupported category %q", trimmedWeaponID, weapon.Category)
			}
			normalizedWeaponIDs = append(normalizedWeaponIDs, trimmedWeaponID)
		}
		if primaryCount != 1 {
			return status.Error(codes.InvalidArgument, "starting equipment must include exactly one primary weapon")
		}
		if len(normalizedWeaponIDs) == 2 && secondaryCount != 1 {
			return status.Error(codes.InvalidArgument, "two-weapon loadouts must include exactly one secondary weapon")
		}
		if len(normalizedWeaponIDs) == 1 && secondaryCount != 0 {
			return status.Error(codes.InvalidArgument, "single-weapon loadouts cannot use a secondary weapon")
		}

		armorID := strings.TrimSpace(step.EquipmentInput.GetArmorId())
		if armorID == "" {
			return status.Error(codes.InvalidArgument, "armor_id is required")
		}
		armor, err := c.stores.DaggerheartContent.GetDaggerheartArmor(ctx, armorID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "armor_id %q is not found", armorID)
			}
			return status.Errorf(codes.Internal, "get armor: %v", err)
		}
		if armor.Tier != 1 {
			return status.Errorf(codes.InvalidArgument, "armor_id %q must be tier 1", armorID)
		}

		potionItemID := strings.TrimSpace(step.EquipmentInput.GetPotionItemId())
		if !daggerheart.IsValidStartingPotionItemID(potionItemID) {
			return status.Errorf(
				codes.InvalidArgument,
				"potion_item_id must be %q or %q",
				daggerheart.StartingPotionMinorHealthID,
				daggerheart.StartingPotionMinorStaminaID,
			)
		}
		if _, err := c.stores.DaggerheartContent.GetDaggerheartItem(ctx, potionItemID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "potion_item_id %q is not found", potionItemID)
			}
			return status.Errorf(codes.Internal, "get potion item: %v", err)
		}

		profile.StartingWeaponIDs = normalizedWeaponIDs
		profile.StartingArmorID = armorID
		profile.StartingPotionItemID = potionItemID
		profile.Proficiency = daggerheart.PCProficiency
		profile.ArmorScore = armor.ArmorScore
		if profile.Level == 0 {
			profile.Level = daggerheart.PCLevelDefault
		}
		profile.MajorThreshold, profile.SevereThreshold = daggerheart.DeriveThresholds(
			profile.Level,
			profile.ArmorScore,
			armor.BaseMajorThreshold,
			armor.BaseSevereThreshold,
		)
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_BackgroundInput:
		background := strings.TrimSpace(step.BackgroundInput.GetBackground())
		if background == "" {
			return status.Error(codes.InvalidArgument, "background is required")
		}
		profile.Background = background
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput:
		items := step.ExperiencesInput.GetExperiences()
		if len(items) == 0 {
			return status.Error(codes.InvalidArgument, "at least one experience is required")
		}
		experiences := make([]storage.DaggerheartExperience, 0, len(items))
		for _, item := range items {
			name := strings.TrimSpace(item.GetName())
			if name == "" {
				return status.Error(codes.InvalidArgument, "experience name is required")
			}
			experiences = append(experiences, storage.DaggerheartExperience{
				Name:     name,
				Modifier: int(item.GetModifier()),
			})
		}
		profile.Experiences = experiences
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput:
		if strings.TrimSpace(profile.ClassID) == "" {
			return status.Error(codes.FailedPrecondition, "class must be selected before domain cards")
		}
		class, err := c.stores.DaggerheartContent.GetDaggerheartClass(ctx, profile.ClassID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "class_id %q is not found", profile.ClassID)
			}
			return status.Errorf(codes.Internal, "get class: %v", err)
		}
		allowedDomains := make(map[string]struct{}, len(class.DomainIDs))
		for _, domainID := range class.DomainIDs {
			trimmedDomainID := strings.TrimSpace(domainID)
			if trimmedDomainID == "" {
				continue
			}
			allowedDomains[trimmedDomainID] = struct{}{}
		}
		if len(allowedDomains) == 0 {
			return status.Errorf(codes.InvalidArgument, "class_id %q has no configured domains", profile.ClassID)
		}

		domainCardIDs := step.DomainCardsInput.GetDomainCardIds()
		if len(domainCardIDs) == 0 {
			return status.Error(codes.InvalidArgument, "at least one domain_card_id is required")
		}
		normalizedIDs := make([]string, 0, len(domainCardIDs))
		for _, domainCardID := range domainCardIDs {
			trimmed := strings.TrimSpace(domainCardID)
			if trimmed == "" {
				return status.Error(codes.InvalidArgument, "domain_card_ids must not contain empty values")
			}
			card, err := c.stores.DaggerheartContent.GetDaggerheartDomainCard(ctx, trimmed)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					return status.Errorf(codes.InvalidArgument, "domain_card_id %q is not found", trimmed)
				}
				return status.Errorf(codes.Internal, "get domain card: %v", err)
			}
			if _, ok := allowedDomains[strings.TrimSpace(card.DomainID)]; !ok {
				return status.Errorf(codes.InvalidArgument, "domain_card_id %q is not in class domains", trimmed)
			}
			normalizedIDs = append(normalizedIDs, trimmed)
		}
		profile.DomainCardIDs = normalizedIDs
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput:
		connections := strings.TrimSpace(step.ConnectionsInput.GetConnections())
		if connections == "" {
			return status.Error(codes.InvalidArgument, "connections are required")
		}
		profile.Connections = connections
		return nil

	default:
		return status.Error(codes.InvalidArgument, "daggerheart creation step is required")
	}
}

func (c characterApplication) executeCharacterProfileUpdate(ctx context.Context, campaignRecord storage.CampaignRecord, characterID string, systemProfile map[string]any) error {
	policyActor, err := requireCharacterMutationPolicy(ctx, c.stores, campaignRecord, characterID)
	if err != nil {
		return err
	}

	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID == "" {
		actorID = strings.TrimSpace(policyActor.ID)
	}
	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return status.Error(codes.Internal, "domain engine is not configured")
	}

	commandPayload := character.ProfileUpdatePayload{
		CharacterID:   characterID,
		SystemProfile: systemProfile,
	}
	commandPayloadJSON, err := json.Marshal(commandPayload)
	if err != nil {
		return status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignRecord.ID,
			Type:         commandTypeCharacterProfileUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "character",
			EntityID:     characterID,
			PayloadJSON:  commandPayloadJSON,
		}),
		domainCommandApplyOptions{},
	)
	return err
}

func daggerheartCreationStepNumber(input *daggerheartv1.DaggerheartCreationStepInput) (int32, error) {
	if input == nil {
		return 0, status.Error(codes.InvalidArgument, "daggerheart step payload is required")
	}
	switch input.GetStep().(type) {
	case *daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput:
		return daggerheart.CreationStepClassSubclass, nil
	case *daggerheartv1.DaggerheartCreationStepInput_HeritageInput:
		return daggerheart.CreationStepHeritage, nil
	case *daggerheartv1.DaggerheartCreationStepInput_TraitsInput:
		return daggerheart.CreationStepTraits, nil
	case *daggerheartv1.DaggerheartCreationStepInput_DetailsInput:
		return daggerheart.CreationStepDetails, nil
	case *daggerheartv1.DaggerheartCreationStepInput_EquipmentInput:
		return daggerheart.CreationStepEquipment, nil
	case *daggerheartv1.DaggerheartCreationStepInput_BackgroundInput:
		return daggerheart.CreationStepBackground, nil
	case *daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput:
		return daggerheart.CreationStepExperiences, nil
	case *daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput:
		return daggerheart.CreationStepDomainCards, nil
	case *daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput:
		return daggerheart.CreationStepConnections, nil
	default:
		return 0, status.Error(codes.InvalidArgument, "daggerheart creation step is required")
	}
}

func validateDaggerheartProfile(profile storage.DaggerheartCharacterProfile) error {
	experiences := make([]daggerheart.Experience, 0, len(profile.Experiences))
	for _, experience := range profile.Experiences {
		experiences = append(experiences, daggerheart.Experience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}

	if err := daggerheart.ValidateProfile(
		profile.Level,
		profile.HpMax,
		profile.StressMax,
		profile.Evasion,
		profile.MajorThreshold,
		profile.SevereThreshold,
		profile.Proficiency,
		profile.ArmorScore,
		profile.ArmorMax,
		daggerheart.Traits{
			Agility:   profile.Agility,
			Strength:  profile.Strength,
			Finesse:   profile.Finesse,
			Instinct:  profile.Instinct,
			Presence:  profile.Presence,
			Knowledge: profile.Knowledge,
		},
		experiences,
	); err != nil {
		return err
	}
	return nil
}

func defaultDaggerheartProfileForCharacter(campaignID string, characterRecord storage.CharacterRecord) storage.DaggerheartCharacterProfile {
	profile := storage.DaggerheartCharacterProfile{
		CampaignID:  campaignID,
		CharacterID: characterRecord.ID,
	}
	return ensureDaggerheartProfileDefaults(profile, characterRecord.Kind)
}

func ensureDaggerheartProfileDefaults(profile storage.DaggerheartCharacterProfile, kind character.Kind) storage.DaggerheartCharacterProfile {
	kindLabel := "PC"
	if kind == character.KindNPC {
		kindLabel = "NPC"
	}
	defaults := daggerheart.GetProfileDefaults(kindLabel)

	if profile.Level == 0 {
		profile.Level = defaults.Level
	}
	if profile.HpMax == 0 {
		profile.HpMax = defaults.HpMax
	}
	if profile.StressMax == 0 {
		profile.StressMax = defaults.StressMax
	}
	if profile.Evasion == 0 {
		profile.Evasion = defaults.Evasion
	}
	if profile.Proficiency == 0 {
		profile.Proficiency = defaults.Proficiency
	}
	if profile.ArmorMax == 0 {
		profile.ArmorMax = defaults.ArmorMax
	}
	if profile.MajorThreshold == 0 && profile.SevereThreshold == 0 {
		profile.MajorThreshold, profile.SevereThreshold = daggerheart.DeriveThresholds(
			profile.Level,
			profile.ArmorScore,
			defaults.MajorThreshold,
			defaults.SevereThreshold,
		)
	}
	return profile
}

func daggerheartSystemProfileMap(profile storage.DaggerheartCharacterProfile) map[string]any {
	experiencesPayload := make([]map[string]any, 0, len(profile.Experiences))
	for _, experience := range profile.Experiences {
		experiencesPayload = append(experiencesPayload, map[string]any{
			"name":     experience.Name,
			"modifier": experience.Modifier,
		})
	}

	return map[string]any{
		"daggerheart": map[string]any{
			"level":                   profile.Level,
			"hp_max":                  profile.HpMax,
			"stress_max":              profile.StressMax,
			"evasion":                 profile.Evasion,
			"major_threshold":         profile.MajorThreshold,
			"severe_threshold":        profile.SevereThreshold,
			"proficiency":             profile.Proficiency,
			"armor_score":             profile.ArmorScore,
			"armor_max":               profile.ArmorMax,
			"agility":                 profile.Agility,
			"strength":                profile.Strength,
			"finesse":                 profile.Finesse,
			"instinct":                profile.Instinct,
			"presence":                profile.Presence,
			"knowledge":               profile.Knowledge,
			"experiences":             experiencesPayload,
			"class_id":                profile.ClassID,
			"subclass_id":             profile.SubclassID,
			"ancestry_id":             profile.AncestryID,
			"community_id":            profile.CommunityID,
			"traits_assigned":         profile.TraitsAssigned,
			"details_recorded":        profile.DetailsRecorded,
			"starting_weapon_ids":     append([]string(nil), profile.StartingWeaponIDs...),
			"starting_armor_id":       profile.StartingArmorID,
			"starting_potion_item_id": profile.StartingPotionItemID,
			"background":              profile.Background,
			"domain_card_ids":         append([]string(nil), profile.DomainCardIDs...),
			"connections":             profile.Connections,
		},
	}
}

func daggerheartCreationProfileFromStorage(profile storage.DaggerheartCharacterProfile) daggerheart.CreationProfile {
	experiences := make([]daggerheart.Experience, 0, len(profile.Experiences))
	for _, experience := range profile.Experiences {
		experiences = append(experiences, daggerheart.Experience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}

	return daggerheart.CreationProfile{
		ClassID:        profile.ClassID,
		SubclassID:     profile.SubclassID,
		AncestryID:     profile.AncestryID,
		CommunityID:    profile.CommunityID,
		TraitsAssigned: profile.TraitsAssigned,
		Traits: daggerheart.Traits{
			Agility:   profile.Agility,
			Strength:  profile.Strength,
			Finesse:   profile.Finesse,
			Instinct:  profile.Instinct,
			Presence:  profile.Presence,
			Knowledge: profile.Knowledge,
		},
		DetailsRecorded:      profile.DetailsRecorded,
		Level:                profile.Level,
		HpMax:                profile.HpMax,
		StressMax:            profile.StressMax,
		Evasion:              profile.Evasion,
		StartingWeaponIDs:    append([]string(nil), profile.StartingWeaponIDs...),
		StartingArmorID:      profile.StartingArmorID,
		StartingPotionItemID: profile.StartingPotionItemID,
		Background:           profile.Background,
		Experiences:          experiences,
		DomainCardIDs:        append([]string(nil), profile.DomainCardIDs...),
		Connections:          profile.Connections,
	}
}

func handleWorkflowError(err error) error {
	if err == nil {
		return nil
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		return handleDomainError(err)
	}
	return err
}
