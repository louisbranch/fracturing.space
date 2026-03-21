package creationworkflow

import (
	"context"
	"errors"
	"strings"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// applyClassSubclassInput validates the selected class/subclass pair before
// storing the canonical IDs on the creation profile.
func applyClassSubclassInput(ctx context.Context, content contentstore.DaggerheartContentReadStore, profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepClassSubclassInput) error {
	classID, err := validate.RequiredID(input.GetClassId(), "class_id")
	if err != nil {
		return err
	}
	subclassID, err := validate.RequiredID(input.GetSubclassId(), "subclass_id")
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
	requirements := make([]projectionstore.DaggerheartSubclassCreationRequirement, 0, len(subclass.CreationRequirements))
	requiresCompanion := false
	for _, requirement := range subclass.CreationRequirements {
		value := strings.TrimSpace(string(requirement))
		if value != "" {
			typed := projectionstore.DaggerheartSubclassCreationRequirement(value)
			requirements = append(requirements, typed)
			if typed == projectionstore.DaggerheartSubclassCreationRequirementCompanionSheet {
				requiresCompanion = true
			}
		}
	}
	if len(requirements) == 0 {
		profile.SubclassCreationRequirements = nil
	} else {
		profile.SubclassCreationRequirements = requirements
	}
	companion, err := companionSheetFromInput(ctx, content, input.GetCompanion(), requiresCompanion)
	if err != nil {
		return err
	}
	profile.CompanionSheet = companion
	return nil
}

func companionSheetFromInput(ctx context.Context, content contentstore.DaggerheartContentReadStore, input *daggerheartv1.DaggerheartCreationCompanionInput, required bool) (*projectionstore.DaggerheartCompanionSheet, error) {
	if input == nil {
		if required {
			return nil, status.Error(codes.InvalidArgument, "companion is required")
		}
		return nil, nil
	}

	experienceIDs := input.GetExperienceIds()
	if len(experienceIDs) != 2 {
		return nil, status.Error(codes.InvalidArgument, "companion requires exactly two experience_ids")
	}
	seen := make(map[string]struct{}, len(experienceIDs))
	experiences := make([]projectionstore.DaggerheartCompanionExperience, 0, len(experienceIDs))
	for _, rawID := range experienceIDs {
		experienceID, err := validate.RequiredID(rawID, "companion experience_id")
		if err != nil {
			return nil, err
		}
		if _, exists := seen[experienceID]; exists {
			return nil, status.Error(codes.InvalidArgument, "companion experience_ids must be distinct")
		}
		if _, err := content.GetDaggerheartCompanionExperience(ctx, experienceID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return nil, status.Errorf(codes.InvalidArgument, "companion experience_id %q is not found", experienceID)
			}
			return nil, grpcerror.Internal("get companion experience", err)
		}
		seen[experienceID] = struct{}{}
		experiences = append(experiences, projectionstore.DaggerheartCompanionExperience{
			ExperienceID: experienceID,
			Modifier:     daggerheart.CompanionSheetExperienceModifier,
		})
	}

	damageType := strings.TrimSpace(input.GetDamageType())
	if damageType == "" {
		return nil, status.Error(codes.InvalidArgument, "companion damage_type is required")
	}

	return &projectionstore.DaggerheartCompanionSheet{
		AnimalKind:        strings.TrimSpace(input.GetAnimalKind()),
		Name:              strings.TrimSpace(input.GetName()),
		Evasion:           daggerheart.CompanionSheetDefaultEvasion,
		Experiences:       experiences,
		AttackDescription: strings.TrimSpace(input.GetAttackDescription()),
		AttackRange:       daggerheart.CompanionSheetDefaultAttackRange,
		DamageDieSides:    daggerheart.CompanionSheetDefaultDamageDieSides,
		DamageType:        damageType,
	}, nil
}

// applyHeritageInput validates the ancestry/community split before storing the
// selected heritage IDs on the creation profile.
func applyHeritageInput(ctx context.Context, content contentstore.DaggerheartContentReadStore, profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepHeritageInput) error {
	selection := input.GetHeritage()
	if selection == nil {
		return status.Error(codes.InvalidArgument, "heritage is required")
	}
	firstAncestryID, err := validate.RequiredID(selection.GetFirstFeatureAncestryId(), "first_feature_ancestry_id")
	if err != nil {
		return err
	}
	secondAncestryID := strings.TrimSpace(selection.GetSecondFeatureAncestryId())
	if secondAncestryID == "" {
		secondAncestryID = firstAncestryID
	}
	communityID, err := validate.RequiredID(selection.GetCommunityId(), "community_id")
	if err != nil {
		return err
	}

	firstAncestry, err := content.GetDaggerheartHeritage(ctx, firstAncestryID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return status.Errorf(codes.InvalidArgument, "first_feature_ancestry_id %q is not found", firstAncestryID)
		}
		return grpcerror.Internal("get ancestry heritage", err)
	}
	if !strings.EqualFold(strings.TrimSpace(firstAncestry.Kind), "ancestry") {
		return status.Errorf(codes.InvalidArgument, "first_feature_ancestry_id %q is not an ancestry heritage", firstAncestryID)
	}

	secondAncestry, err := content.GetDaggerheartHeritage(ctx, secondAncestryID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return status.Errorf(codes.InvalidArgument, "second_feature_ancestry_id %q is not found", secondAncestryID)
		}
		return grpcerror.Internal("get secondary ancestry heritage", err)
	}
	if !strings.EqualFold(strings.TrimSpace(secondAncestry.Kind), "ancestry") {
		return status.Errorf(codes.InvalidArgument, "second_feature_ancestry_id %q is not an ancestry heritage", secondAncestryID)
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

	firstFeatureID, err := requiredHeritageFeatureID(firstAncestry)
	if err != nil {
		return err
	}
	secondFeatureID, err := requiredHeritageFeatureID(secondAncestry)
	if err != nil {
		return err
	}

	profile.Heritage = projectionstore.DaggerheartHeritageSelection{
		AncestryLabel:           strings.TrimSpace(selection.GetAncestryLabel()),
		FirstFeatureAncestryID:  firstAncestryID,
		FirstFeatureID:          firstFeatureID,
		SecondFeatureAncestryID: secondAncestryID,
		SecondFeatureID:         secondFeatureID,
		CommunityID:             communityID,
	}
	return nil
}

func requiredHeritageFeatureID(heritage contentstore.DaggerheartHeritage) (string, error) {
	if len(heritage.Features) == 0 {
		return "", status.Errorf(codes.InvalidArgument, "heritage %q is missing features", heritage.ID)
	}
	featureID := strings.TrimSpace(heritage.Features[0].ID)
	if featureID == "" {
		return "", status.Errorf(codes.InvalidArgument, "heritage %q has an empty feature id", heritage.ID)
	}
	return featureID, nil
}

// applyTraitsInput validates the creation trait allocation before persisting it
// to the profile.
func applyTraitsInput(profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepTraitsInput) error {
	traits := daggerheartprofile.Traits{
		Agility:   int(input.GetAgility()),
		Strength:  int(input.GetStrength()),
		Finesse:   int(input.GetFinesse()),
		Instinct:  int(input.GetInstinct()),
		Presence:  int(input.GetPresence()),
		Knowledge: int(input.GetKnowledge()),
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
}

// applyDetailsInput records descriptive details and class-derived combat stats
// once class selection is already locked in.
func applyDetailsInput(ctx context.Context, content contentstore.DaggerheartContentReadStore, profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepDetailsInput) error {
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
	desc, err := validate.RequiredID(input.GetDescription(), "description")
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
}
