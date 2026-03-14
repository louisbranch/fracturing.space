package daggerheart

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/workflow"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreationWorkflowProvider implements the character creation workflow for
// the Daggerheart game system. It satisfies workflow.Provider.
type CreationWorkflowProvider struct{}

func (CreationWorkflowProvider) GetProgress(ctx context.Context, deps workflow.CreationDeps, campaignContext workflow.CampaignContext, characterID string) (workflow.Progress, error) {
	if err := campaign.ValidateCampaignOperation(campaignContext.Status, campaign.CampaignOpRead); err != nil {
		return workflow.Progress{}, err
	}
	if err := deps.RequireReadPolicy(ctx, campaignContext); err != nil {
		return workflow.Progress{}, err
	}

	if _, err := deps.GetCharacterRecord(ctx, campaignContext.ID, characterID); err != nil {
		return workflow.Progress{}, err
	}

	profile, err := deps.GetCharacterSystemProfile(ctx, campaignContext.ID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return workflow.Progress{}, grpcerror.Internal("get daggerheart profile", err)
		}
		profile = storage.DaggerheartCharacterProfile{CampaignID: campaignContext.ID, CharacterID: characterID}
	}

	progress := daggerheart.EvaluateCreationProgress(creationProfileFromStorage(profile))
	return progressFromDaggerheart(progress), nil
}

func (CreationWorkflowProvider) ApplyStep(ctx context.Context, deps workflow.CreationDeps, campaignContext workflow.CampaignContext, in *campaignv1.ApplyCharacterCreationStepRequest) (*campaignv1.CharacterProfile, workflow.Progress, error) {
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, workflow.Progress{}, err
	}
	stepInput := in.GetDaggerheart()
	if stepInput == nil {
		return nil, workflow.Progress{}, status.Error(codes.InvalidArgument, "daggerheart step payload is required")
	}
	stepNumber, err := creationStepNumber(stepInput)
	if err != nil {
		return nil, workflow.Progress{}, err
	}

	if err := campaign.ValidateCampaignOperation(campaignContext.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, workflow.Progress{}, err
	}

	characterRecord, err := deps.GetCharacterRecord(ctx, campaignContext.ID, characterID)
	if err != nil {
		return nil, workflow.Progress{}, err
	}

	profile, err := deps.GetCharacterSystemProfile(ctx, campaignContext.ID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, workflow.Progress{}, grpcerror.Internal("get daggerheart profile", err)
		}
		profile = defaultProfileForCharacter(campaignContext.ID, characterRecord.Kind)
	} else {
		profile = ensureProfileDefaults(profile, characterRecord.Kind)
	}

	currentProgress := daggerheart.EvaluateCreationProgress(creationProfileFromStorage(profile))
	if currentProgress.Ready {
		return nil, workflow.Progress{}, status.Error(codes.FailedPrecondition, "character creation workflow is already complete")
	}
	if stepNumber != currentProgress.NextStep {
		return nil, workflow.Progress{}, status.Errorf(
			codes.FailedPrecondition,
			"creation step %d is not allowed; expected step %d",
			stepNumber,
			currentProgress.NextStep,
		)
	}

	if err := applyCreationStepInput(ctx, deps.SystemContent(), &profile, stepInput); err != nil {
		return nil, workflow.Progress{}, err
	}

	if err := validateProfile(profile); err != nil {
		return nil, workflow.Progress{}, err
	}

	if err := deps.ExecuteProfileUpdate(ctx, campaignContext, characterID, SystemProfileMap(profile)); err != nil {
		return nil, workflow.Progress{}, err
	}

	nextProgress := daggerheart.EvaluateCreationProgress(creationProfileFromStorage(profile))
	return deps.ProfileToProto(campaignContext.ID, characterID, profile), progressFromDaggerheart(nextProgress), nil
}

func (CreationWorkflowProvider) ApplyWorkflow(ctx context.Context, deps workflow.CreationDeps, campaignContext workflow.CampaignContext, in *campaignv1.ApplyCharacterCreationWorkflowRequest) (*campaignv1.CharacterProfile, workflow.Progress, error) {
	characterID, err := validate.RequiredID(in.GetCharacterId(), "character id")
	if err != nil {
		return nil, workflow.Progress{}, err
	}
	workflowInput := in.GetDaggerheart()
	if workflowInput == nil {
		return nil, workflow.Progress{}, status.Error(codes.InvalidArgument, "daggerheart workflow payload is required")
	}

	if err := campaign.ValidateCampaignOperation(campaignContext.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, workflow.Progress{}, err
	}

	characterRecord, err := deps.GetCharacterRecord(ctx, campaignContext.ID, characterID)
	if err != nil {
		return nil, workflow.Progress{}, err
	}

	profile, err := deps.GetCharacterSystemProfile(ctx, campaignContext.ID, characterID)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, workflow.Progress{}, grpcerror.Internal("get daggerheart profile", err)
		}
		profile = defaultProfileForCharacter(campaignContext.ID, characterRecord.Kind)
	} else {
		profile = ensureProfileDefaults(profile, characterRecord.Kind)
	}

	profile = resetCreationWorkflowFields(profile)

	steps, err := creationStepSequenceFromWorkflowInput(workflowInput)
	if err != nil {
		return nil, workflow.Progress{}, err
	}

	for idx, stepInput := range steps {
		expectedStep := int32(idx + 1)
		currentProgress := daggerheart.EvaluateCreationProgress(creationProfileFromStorage(profile))
		if currentProgress.Ready {
			return nil, workflow.Progress{}, status.Error(codes.FailedPrecondition, "character creation workflow is already complete")
		}
		if currentProgress.NextStep != expectedStep {
			return nil, workflow.Progress{}, status.Errorf(codes.FailedPrecondition, "creation step %d is not allowed; expected step %d", expectedStep, currentProgress.NextStep)
		}
		stepNumber, err := creationStepNumber(stepInput)
		if err != nil {
			return nil, workflow.Progress{}, err
		}
		if stepNumber != expectedStep {
			return nil, workflow.Progress{}, status.Errorf(codes.InvalidArgument, "daggerheart workflow payload contains out-of-order step %d at position %d", stepNumber, idx+1)
		}
		if err := applyCreationStepInput(ctx, deps.SystemContent(), &profile, stepInput); err != nil {
			return nil, workflow.Progress{}, err
		}
		if err := validateProfile(profile); err != nil {
			return nil, workflow.Progress{}, err
		}
	}

	finalProgress := daggerheart.EvaluateCreationProgress(creationProfileFromStorage(profile))
	if !finalProgress.Ready {
		if len(finalProgress.UnmetReasons) > 0 {
			return nil, workflow.Progress{}, status.Errorf(codes.FailedPrecondition, "character creation workflow is incomplete: %s", finalProgress.UnmetReasons[0])
		}
		return nil, workflow.Progress{}, status.Error(codes.FailedPrecondition, "character creation workflow is incomplete")
	}

	if err := deps.ExecuteProfileUpdate(ctx, campaignContext, characterID, SystemProfileMap(profile)); err != nil {
		return nil, workflow.Progress{}, err
	}

	return deps.ProfileToProto(campaignContext.ID, characterID, profile), progressFromDaggerheart(finalProgress), nil
}

func (CreationWorkflowProvider) Reset(ctx context.Context, deps workflow.CreationDeps, campaignContext workflow.CampaignContext, characterID string) (workflow.Progress, error) {
	if err := campaign.ValidateCampaignOperation(campaignContext.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return workflow.Progress{}, err
	}

	if _, err := deps.GetCharacterRecord(ctx, campaignContext.ID, characterID); err != nil {
		return workflow.Progress{}, err
	}

	if err := deps.ExecuteProfileUpdate(ctx, campaignContext, characterID, map[string]any{
		"daggerheart": map[string]any{"reset": true},
	}); err != nil {
		return workflow.Progress{}, err
	}

	return progressFromDaggerheart(daggerheart.EvaluateCreationProgress(daggerheart.CreationProfile{})), nil
}

// HandleWorkflowError maps domain errors to gRPC status errors for workflow
// endpoints. Unknown errors pass through unchanged.
func HandleWorkflowError(err error) error {
	if err == nil {
		return nil
	}
	if apperrors.GetCode(err) != apperrors.CodeUnknown {
		return handleDomainError(err)
	}
	return err
}

// --- helper functions ---

func progressFromDaggerheart(progress daggerheart.CreationProgress) workflow.Progress {
	steps := make([]workflow.StepProgress, 0, len(progress.Steps))
	for _, step := range progress.Steps {
		steps = append(steps, workflow.StepProgress{
			Step:     step.Step,
			Key:      step.Key,
			Complete: step.Complete,
		})
	}
	return workflow.Progress{
		Steps:        steps,
		NextStep:     progress.NextStep,
		Ready:        progress.Ready,
		UnmetReasons: append([]string(nil), progress.UnmetReasons...),
	}
}

func creationStepSequenceFromWorkflowInput(input *daggerheartv1.DaggerheartCreationWorkflowInput) ([]*daggerheartv1.DaggerheartCreationStepInput, error) {
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
		{Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{EquipmentInput: input.GetEquipmentInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput{ExperiencesInput: input.GetExperiencesInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{DomainCardsInput: input.GetDomainCardsInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{DetailsInput: input.GetDetailsInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_BackgroundInput{BackgroundInput: input.GetBackgroundInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput{ConnectionsInput: input.GetConnectionsInput()}},
	}, nil
}

func resetCreationWorkflowFields(profile storage.DaggerheartCharacterProfile) storage.DaggerheartCharacterProfile {
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
	profile.Description = ""
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

func applyCreationStepInput(ctx context.Context, content storage.DaggerheartContentReadStore, profile *storage.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepInput) error {
	if content == nil {
		return status.Error(codes.Internal, "daggerheart content store is not configured")
	}

	switch step := input.GetStep().(type) {
	case *daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput:
		classID, err := validate.RequiredID(step.ClassSubclassInput.GetClassId(), "class_id")
		if err != nil {
			return err
		}
		subclassID, err := validate.RequiredID(step.ClassSubclassInput.GetSubclassId(), "subclass_id")
		if err != nil {
			return err
		}
		if _, err := content.GetDaggerheartClass(ctx, classID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "class_id %q is not found", classID)
			}
			return grpcerror.Internal("get class", err)
		}
		subclass, err := content.GetDaggerheartSubclass(ctx, subclassID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "subclass_id %q is not found", subclassID)
			}
			return grpcerror.Internal("get subclass", err)
		}
		if strings.TrimSpace(subclass.ClassID) != classID {
			return status.Errorf(codes.InvalidArgument, "subclass_id %q does not belong to class_id %q", subclassID, classID)
		}
		profile.ClassID = classID
		profile.SubclassID = subclassID
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_HeritageInput:
		ancestryID, err := validate.RequiredID(step.HeritageInput.GetAncestryId(), "ancestry_id")
		if err != nil {
			return err
		}
		communityID, err := validate.RequiredID(step.HeritageInput.GetCommunityId(), "community_id")
		if err != nil {
			return err
		}

		ancestry, err := content.GetDaggerheartHeritage(ctx, ancestryID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "ancestry_id %q is not found", ancestryID)
			}
			return grpcerror.Internal("get ancestry heritage", err)
		}
		if !strings.EqualFold(strings.TrimSpace(ancestry.Kind), "ancestry") {
			return status.Errorf(codes.InvalidArgument, "ancestry_id %q is not an ancestry heritage", ancestryID)
		}

		community, err := content.GetDaggerheartHeritage(ctx, communityID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "community_id %q is not found", communityID)
			}
			return grpcerror.Internal("get community heritage", err)
		}
		if !strings.EqualFold(strings.TrimSpace(community.Kind), "community") {
			return status.Errorf(codes.InvalidArgument, "community_id %q is not a community heritage", communityID)
		}

		profile.AncestryID = ancestryID
		profile.CommunityID = communityID
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_TraitsInput:
		traits := daggerheartprofile.Traits{
			Agility:   int(step.TraitsInput.GetAgility()),
			Strength:  int(step.TraitsInput.GetStrength()),
			Finesse:   int(step.TraitsInput.GetFinesse()),
			Instinct:  int(step.TraitsInput.GetInstinct()),
			Presence:  int(step.TraitsInput.GetPresence()),
			Knowledge: int(step.TraitsInput.GetKnowledge()),
		}
		if err := daggerheartprofile.ValidateTrait("agility", traits.Agility); err != nil {
			return err
		}
		if err := daggerheartprofile.ValidateTrait("strength", traits.Strength); err != nil {
			return err
		}
		if err := daggerheartprofile.ValidateTrait("finesse", traits.Finesse); err != nil {
			return err
		}
		if err := daggerheartprofile.ValidateTrait("instinct", traits.Instinct); err != nil {
			return err
		}
		if err := daggerheartprofile.ValidateTrait("presence", traits.Presence); err != nil {
			return err
		}
		if err := daggerheartprofile.ValidateTrait("knowledge", traits.Knowledge); err != nil {
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
		class, err := content.GetDaggerheartClass(ctx, profile.ClassID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "class_id %q is not found", profile.ClassID)
			}
			return grpcerror.Internal("get class", err)
		}
		if class.StartingHP <= 0 {
			return status.Errorf(codes.InvalidArgument, "class_id %q has invalid starting hp", profile.ClassID)
		}
		if class.StartingEvasion <= 0 {
			return status.Errorf(codes.InvalidArgument, "class_id %q has invalid starting evasion", profile.ClassID)
		}
		profile.Level = daggerheartprofile.PCLevelDefault
		profile.HpMax = class.StartingHP
		profile.StressMax = daggerheartprofile.PCStressMax
		profile.Evasion = class.StartingEvasion
		desc, err := validate.RequiredID(step.DetailsInput.GetDescription(), "description")
		if err != nil {
			return err
		}
		profile.DetailsRecorded = true
		profile.Description = desc
		profile.MajorThreshold, profile.SevereThreshold = daggerheartprofile.DeriveThresholds(
			profile.Level,
			profile.ArmorScore,
			daggerheartprofile.PCMajorThreshold,
			daggerheartprofile.PCSevereThreshold,
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
			weapon, err := content.GetDaggerheartWeapon(ctx, trimmedWeaponID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					return status.Errorf(codes.InvalidArgument, "weapon_id %q is not found", trimmedWeaponID)
				}
				return grpcerror.Internal("get weapon", err)
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

		armorID, err := validate.RequiredID(step.EquipmentInput.GetArmorId(), "armor_id")
		if err != nil {
			return err
		}
		armor, err := content.GetDaggerheartArmor(ctx, armorID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "armor_id %q is not found", armorID)
			}
			return grpcerror.Internal("get armor", err)
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
		if _, err := content.GetDaggerheartItem(ctx, potionItemID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "potion_item_id %q is not found", potionItemID)
			}
			return grpcerror.Internal("get potion item", err)
		}

		profile.StartingWeaponIDs = normalizedWeaponIDs
		profile.StartingArmorID = armorID
		profile.StartingPotionItemID = potionItemID
		profile.Proficiency = daggerheartprofile.PCProficiency
		profile.ArmorScore = armor.ArmorScore
		if profile.Level == 0 {
			profile.Level = daggerheartprofile.PCLevelDefault
		}
		profile.MajorThreshold, profile.SevereThreshold = daggerheartprofile.DeriveThresholds(
			profile.Level,
			profile.ArmorScore,
			armor.BaseMajorThreshold,
			armor.BaseSevereThreshold,
		)
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_BackgroundInput:
		background, err := validate.RequiredID(step.BackgroundInput.GetBackground(), "background")
		if err != nil {
			return err
		}
		profile.Background = background
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput:
		items := step.ExperiencesInput.GetExperiences()
		if len(items) != 2 {
			return status.Error(codes.InvalidArgument, "exactly two experiences are required")
		}
		experiences := make([]storage.DaggerheartExperience, 0, len(items))
		for _, item := range items {
			name, err := validate.RequiredID(item.GetName(), "experience name")
			if err != nil {
				return err
			}
			experiences = append(experiences, storage.DaggerheartExperience{
				Name:     name,
				Modifier: 2,
			})
		}
		profile.Experiences = experiences
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput:
		if strings.TrimSpace(profile.ClassID) == "" {
			return status.Error(codes.FailedPrecondition, "class must be selected before domain cards")
		}
		class, err := content.GetDaggerheartClass(ctx, profile.ClassID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Errorf(codes.InvalidArgument, "class_id %q is not found", profile.ClassID)
			}
			return grpcerror.Internal("get class", err)
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
		if len(domainCardIDs) != 2 {
			return status.Error(codes.InvalidArgument, "exactly two domain cards are required")
		}
		normalizedIDs := make([]string, 0, len(domainCardIDs))
		for _, domainCardID := range domainCardIDs {
			trimmed := strings.TrimSpace(domainCardID)
			if trimmed == "" {
				return status.Error(codes.InvalidArgument, "domain_card_ids must not contain empty values")
			}
			card, err := content.GetDaggerheartDomainCard(ctx, trimmed)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					return status.Errorf(codes.InvalidArgument, "domain_card_id %q is not found", trimmed)
				}
				return grpcerror.Internal("get domain card", err)
			}
			if card.Level != 1 {
				return status.Errorf(codes.InvalidArgument, "domain_card_id %q is level %d, only level 1 cards are allowed at creation", trimmed, card.Level)
			}
			if _, ok := allowedDomains[strings.TrimSpace(card.DomainID)]; !ok {
				return status.Errorf(codes.InvalidArgument, "domain_card_id %q is not in class domains", trimmed)
			}
			normalizedIDs = append(normalizedIDs, trimmed)
		}
		profile.DomainCardIDs = normalizedIDs
		return nil

	case *daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput:
		connections, err := validate.RequiredID(step.ConnectionsInput.GetConnections(), "connections")
		if err != nil {
			return err
		}
		profile.Connections = connections
		return nil

	default:
		return status.Error(codes.InvalidArgument, "daggerheart creation step is required")
	}
}

func creationStepNumber(input *daggerheartv1.DaggerheartCreationStepInput) (int32, error) {
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

func validateProfile(profile storage.DaggerheartCharacterProfile) error {
	experiences := make([]daggerheartprofile.Experience, 0, len(profile.Experiences))
	for _, experience := range profile.Experiences {
		experiences = append(experiences, daggerheartprofile.Experience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}

	return daggerheartprofile.Validate(
		profile.Level,
		profile.HpMax,
		profile.StressMax,
		profile.Evasion,
		profile.MajorThreshold,
		profile.SevereThreshold,
		profile.Proficiency,
		profile.ArmorScore,
		profile.ArmorMax,
		daggerheartprofile.Traits{
			Agility:   profile.Agility,
			Strength:  profile.Strength,
			Finesse:   profile.Finesse,
			Instinct:  profile.Instinct,
			Presence:  profile.Presence,
			Knowledge: profile.Knowledge,
		},
		experiences,
	)
}

func defaultProfileForCharacter(campaignID string, kind character.Kind) storage.DaggerheartCharacterProfile {
	profile := storage.DaggerheartCharacterProfile{
		CampaignID: campaignID,
	}
	return ensureProfileDefaults(profile, kind)
}

func ensureProfileDefaults(profile storage.DaggerheartCharacterProfile, kind character.Kind) storage.DaggerheartCharacterProfile {
	kindLabel := "PC"
	if kind == character.KindNPC {
		kindLabel = "NPC"
	}
	defaults := daggerheartprofile.GetDefaults(kindLabel)

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
		profile.MajorThreshold, profile.SevereThreshold = daggerheartprofile.DeriveThresholds(
			profile.Level,
			profile.ArmorScore,
			defaults.MajorThreshold,
			defaults.SevereThreshold,
		)
	}
	return profile
}

// SystemProfileMap converts a DaggerheartCharacterProfile to the map[string]any
// encoding expected by the character profile update command payload.
func SystemProfileMap(profile storage.DaggerheartCharacterProfile) map[string]any {
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

func creationProfileFromStorage(profile storage.DaggerheartCharacterProfile) daggerheart.CreationProfile {
	experiences := make([]daggerheartprofile.Experience, 0, len(profile.Experiences))
	for _, experience := range profile.Experiences {
		experiences = append(experiences, daggerheartprofile.Experience{
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
		Traits: daggerheartprofile.Traits{
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
