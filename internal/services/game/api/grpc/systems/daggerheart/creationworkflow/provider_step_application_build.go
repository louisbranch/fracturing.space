package creationworkflow

import (
	"context"
	"errors"
	"strings"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
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
	return nil
}

// applyHeritageInput validates the ancestry/community split before storing the
// selected heritage IDs on the creation profile.
func applyHeritageInput(ctx context.Context, content contentstore.DaggerheartContentReadStore, profile *projectionstore.DaggerheartCharacterProfile, input *daggerheartv1.DaggerheartCreationStepHeritageInput) error {
	ancestryID, err := validate.RequiredID(input.GetAncestryId(), "ancestry_id")
	if err != nil {
		return err
	}
	communityID, err := validate.RequiredID(input.GetCommunityId(), "community_id")
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
