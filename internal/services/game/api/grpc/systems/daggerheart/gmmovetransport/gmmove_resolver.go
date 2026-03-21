package gmmovetransport

import (
	"context"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func gmMoveKindFromProto(kind pb.DaggerheartGmMoveKind) (rules.GMMoveKind, error) {
	switch kind {
	case pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE:
		return rules.GMMoveKindInterruptAndMove, nil
	case pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE:
		return rules.GMMoveKindAdditionalMove, nil
	default:
		return rules.GMMoveKindUnspecified, status.Error(codes.InvalidArgument, "gm move kind is required")
	}
}

func gmMoveShapeFromProto(shape pb.DaggerheartGmMoveShape) (rules.GMMoveShape, error) {
	switch shape {
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHOW_WORLD_REACTION:
		return rules.GMMoveShapeShowWorldReaction, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_REVEAL_DANGER:
		return rules.GMMoveShapeRevealDanger, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_FORCE_SPLIT:
		return rules.GMMoveShapeForceSplit, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_MARK_STRESS:
		return rules.GMMoveShapeMarkStress, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT:
		return rules.GMMoveShapeShiftEnvironment, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SPOTLIGHT_ADVERSARY:
		return rules.GMMoveShapeSpotlightAdversary, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CAPTURE_IMPORTANT_TARGET:
		return rules.GMMoveShapeCaptureImportantTarget, nil
	case pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CUSTOM:
		return rules.GMMoveShapeCustom, nil
	default:
		return rules.GMMoveShapeUnspecified, status.Error(codes.InvalidArgument, "gm move shape is required")
	}
}

func (h *Handler) gmMoveTargetFromProto(ctx context.Context, campaignID, sessionID string, in *pb.DaggerheartApplyGmMoveRequest) (gmMoveResolution, error) {
	switch target := in.GetSpendTarget().(type) {
	case *pb.DaggerheartApplyGmMoveRequest_DirectMove:
		kind, err := gmMoveKindFromProto(target.DirectMove.GetKind())
		if err != nil {
			return gmMoveResolution{}, err
		}
		shape, err := gmMoveShapeFromProto(target.DirectMove.GetShape())
		if err != nil {
			return gmMoveResolution{}, err
		}
		description := strings.TrimSpace(target.DirectMove.GetDescription())
		if shape == rules.GMMoveShapeCustom && description == "" {
			return gmMoveResolution{}, status.Error(codes.InvalidArgument, "gm move description is required for custom shape")
		}
		resolution := gmMoveResolution{
			target: daggerheartpayload.GMMoveTarget{
				Type:        rules.GMMoveTargetTypeDirectMove,
				Kind:        kind,
				Shape:       shape,
				Description: description,
			},
			opensGMConsequence: kind == rules.GMMoveKindInterruptAndMove,
		}
		if shape == rules.GMMoveShapeSpotlightAdversary {
			adversaryID, err := validate.RequiredID(target.DirectMove.GetAdversaryId(), "adversary_id")
			if err != nil {
				return gmMoveResolution{}, err
			}
			adversary, err := h.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
			if err != nil {
				return gmMoveResolution{}, err
			}
			resolution.target.AdversaryID = ids.AdversaryID(adversaryID)
			resolution.spotlightAdversary = &adversary
		}
		return resolution, nil
	case *pb.DaggerheartApplyGmMoveRequest_AdversaryFeature:
		adversaryID, err := validate.RequiredID(target.AdversaryFeature.GetAdversaryId(), "adversary_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		featureID, err := validate.RequiredID(target.AdversaryFeature.GetFeatureId(), "feature_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		adversary, err := h.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
		if err != nil {
			return gmMoveResolution{}, err
		}
		entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, adversary.AdversaryEntryID)
		if err != nil {
			return gmMoveResolution{}, mapContentErr("get adversary entry", err)
		}
		feature, ok := findAdversaryFeature(entry, featureID)
		if !ok {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "adversary feature %q was not found on adversary entry %q", featureID, adversary.AdversaryEntryID)
		}
		if strings.TrimSpace(feature.CostType) != "fear" || feature.Cost <= 0 {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "adversary feature %q is not a fear spend", featureID)
		}
		if int(in.GetFearSpent()) != feature.Cost {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "fear_spent must equal adversary feature cost %d", feature.Cost)
		}
		resolution := gmMoveResolution{target: daggerheartpayload.GMMoveTarget{
			Type:        rules.GMMoveTargetTypeAdversaryFeature,
			AdversaryID: ids.AdversaryID(adversaryID),
			FeatureID:   featureID,
			Description: strings.TrimSpace(target.AdversaryFeature.GetDescription()),
		}}
		if payload := stagedFearFeaturePayload(adversary, feature, ""); payload != nil {
			resolution.adversaryFeaturePayload = payload
		}
		return resolution, nil
	case *pb.DaggerheartApplyGmMoveRequest_EnvironmentFeature:
		environmentEntityID, err := validate.RequiredID(target.EnvironmentFeature.GetEnvironmentEntityId(), "environment_entity_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		featureID, err := validate.RequiredID(target.EnvironmentFeature.GetFeatureId(), "feature_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		environmentEntity, err := h.loadEnvironmentEntityForSession(ctx, campaignID, sessionID, environmentEntityID)
		if err != nil {
			return gmMoveResolution{}, err
		}
		env, err := h.deps.Content.GetDaggerheartEnvironment(ctx, environmentEntity.EnvironmentID)
		if err != nil {
			return gmMoveResolution{}, mapContentErr("get environment", err)
		}
		if _, ok := findEnvironmentFeature(env, featureID); !ok {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "environment feature %q was not found on environment %q", featureID, environmentEntity.EnvironmentID)
		}
		return gmMoveResolution{target: daggerheartpayload.GMMoveTarget{
			Type:                rules.GMMoveTargetTypeEnvironmentFeature,
			EnvironmentEntityID: ids.EnvironmentEntityID(environmentEntityID),
			EnvironmentID:       environmentEntity.EnvironmentID,
			FeatureID:           featureID,
			Description:         strings.TrimSpace(target.EnvironmentFeature.GetDescription()),
		}}, nil
	case *pb.DaggerheartApplyGmMoveRequest_AdversaryExperience:
		adversaryID, err := validate.RequiredID(target.AdversaryExperience.GetAdversaryId(), "adversary_id")
		if err != nil {
			return gmMoveResolution{}, err
		}
		experienceName, err := validate.RequiredID(target.AdversaryExperience.GetExperienceName(), "experience_name")
		if err != nil {
			return gmMoveResolution{}, err
		}
		adversary, err := h.loadAdversaryForSession(ctx, campaignID, sessionID, adversaryID)
		if err != nil {
			return gmMoveResolution{}, err
		}
		entry, err := h.deps.Content.GetDaggerheartAdversaryEntry(ctx, adversary.AdversaryEntryID)
		if err != nil {
			return gmMoveResolution{}, mapContentErr("get adversary entry", err)
		}
		experience, ok := findAdversaryExperience(entry, experienceName)
		if !ok {
			return gmMoveResolution{}, status.Errorf(codes.InvalidArgument, "adversary experience %q was not found on adversary entry %q", experienceName, adversary.AdversaryEntryID)
		}
		if in.GetFearSpent() != 1 {
			return gmMoveResolution{}, status.Error(codes.InvalidArgument, "adversary experience spends must spend exactly 1 fear")
		}
		return gmMoveResolution{target: daggerheartpayload.GMMoveTarget{
			Type:           rules.GMMoveTargetTypeAdversaryExperience,
			AdversaryID:    ids.AdversaryID(adversaryID),
			ExperienceName: experienceName,
			Description:    strings.TrimSpace(target.AdversaryExperience.GetDescription()),
		}, adversaryFeaturePayload: &daggerheartpayload.AdversaryFeatureApplyPayload{
			ActorAdversaryID:        ids.AdversaryID(adversaryID),
			AdversaryID:             ids.AdversaryID(adversaryID),
			FeatureID:               "experience:" + experienceName,
			FeatureStatesBefore:     toBridgeAdversaryFeatureStates(adversary.FeatureStates),
			FeatureStatesAfter:      toBridgeAdversaryFeatureStates(adversary.FeatureStates),
			PendingExperienceBefore: toBridgeAdversaryPendingExperience(adversary.PendingExperience),
			PendingExperienceAfter: &rules.AdversaryPendingExperience{
				Name:     experience.Name,
				Modifier: experience.Modifier,
			},
		}}, nil
	default:
		return gmMoveResolution{}, status.Error(codes.InvalidArgument, "gm move spend_target is required")
	}
}
