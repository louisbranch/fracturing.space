package adversarytransport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/conditiontransport"
	bridgeDaggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type adversaryStatsInput struct {
	HP            *wrapperspb.Int32Value
	HPMax         *wrapperspb.Int32Value
	Stress        *wrapperspb.Int32Value
	StressMax     *wrapperspb.Int32Value
	Evasion       *wrapperspb.Int32Value
	Major         *wrapperspb.Int32Value
	Severe        *wrapperspb.Int32Value
	Armor         *wrapperspb.Int32Value
	RequireFields bool
	Current       *projectionstore.DaggerheartAdversary
}

type adversaryStats struct {
	HP        int
	HPMax     int
	Stress    int
	StressMax int
	Evasion   int
	Major     int
	Severe    int
	Armor     int
}

const (
	defaultAdversaryEvasion = daggerheartprofile.AdversaryDefaultEvasion
	defaultAdversaryMajor   = daggerheartprofile.AdversaryDefaultMajor
	defaultAdversarySevere  = daggerheartprofile.AdversaryDefaultSevere
)

func adversaryToProto(adversary projectionstore.DaggerheartAdversary) *pb.DaggerheartAdversary {
	return &pb.DaggerheartAdversary{
		Id:                adversary.AdversaryID,
		CampaignId:        adversary.CampaignID,
		AdversaryEntryId:  adversary.AdversaryEntryID,
		Name:              adversary.Name,
		Kind:              adversary.Kind,
		SessionId:         adversary.SessionID,
		SceneId:           adversary.SceneID,
		Notes:             adversary.Notes,
		Hp:                int32(adversary.HP),
		HpMax:             int32(adversary.HPMax),
		Stress:            int32(adversary.Stress),
		StressMax:         int32(adversary.StressMax),
		Evasion:           int32(adversary.Evasion),
		MajorThreshold:    int32(adversary.Major),
		SevereThreshold:   int32(adversary.Severe),
		Armor:             int32(adversary.Armor),
		ConditionStates:   conditiontransport.ProjectionConditionStatesToProto(adversary.Conditions),
		SpotlightGateId:   adversary.SpotlightGateID,
		SpotlightCount:    int32(adversary.SpotlightCount),
		CreatedAt:         timestamppb.New(adversary.CreatedAt),
		UpdatedAt:         timestamppb.New(adversary.UpdatedAt),
		FeatureStates:     projectionAdversaryFeatureStatesToProto(adversary.FeatureStates),
		PendingExperience: projectionAdversaryPendingExperienceToProto(adversary.PendingExperience),
	}
}

func projectionAdversaryFeatureStatesToProto(in []projectionstore.DaggerheartAdversaryFeatureState) []*pb.DaggerheartAdversaryFeatureState {
	out := make([]*pb.DaggerheartAdversaryFeatureState, 0, len(in))
	for _, featureState := range in {
		out = append(out, &pb.DaggerheartAdversaryFeatureState{
			FeatureId:       featureState.FeatureID,
			Status:          toProtoAdversaryFeatureStateStatus(featureState.Status),
			FocusedTargetId: featureState.FocusedTargetID,
		})
	}
	return out
}

func projectionAdversaryPendingExperienceToProto(in *projectionstore.DaggerheartAdversaryPendingExperience) *pb.DaggerheartAdversaryPendingExperience {
	if in == nil {
		return nil
	}
	return &pb.DaggerheartAdversaryPendingExperience{
		Name:     in.Name,
		Modifier: int32(in.Modifier),
	}
}

func toProtoAdversaryFeatureStateStatus(value string) pb.DaggerheartAdversaryFeatureStateStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ready":
		return pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_READY
	case "active":
		return pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_ACTIVE
	case "cooldown":
		return pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_COOLDOWN
	case "spent":
		return pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_SPENT
	case "staged":
		return pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_STAGED
	default:
		return pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_UNSPECIFIED
	}
}

func loadAdversaryForSession(ctx context.Context, store DaggerheartStore, campaignID, sessionID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if store == nil {
		return projectionstore.DaggerheartAdversary{}, status.Error(codes.Internal, "daggerheart store is not configured")
	}
	adversary, err := store.GetDaggerheartAdversary(ctx, campaignID, adversaryID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return projectionstore.DaggerheartAdversary{}, status.Error(codes.NotFound, "adversary not found")
		}
		return projectionstore.DaggerheartAdversary{}, grpcerror.Internal("load adversary", err)
	}
	if adversary.SessionID != "" && adversary.SessionID != sessionID {
		return projectionstore.DaggerheartAdversary{}, status.Error(codes.FailedPrecondition, "adversary is not in session")
	}
	return adversary, nil
}

func normalizeAdversaryStats(input adversaryStatsInput) (adversaryStats, error) {
	stats := adversaryStats{
		HP:        bridgeDaggerheart.HPDefault,
		HPMax:     bridgeDaggerheart.HPMaxDefault,
		Stress:    bridgeDaggerheart.StressDefault,
		StressMax: bridgeDaggerheart.StressMaxDefault,
		Evasion:   defaultAdversaryEvasion,
		Major:     defaultAdversaryMajor,
		Severe:    defaultAdversarySevere,
		Armor:     bridgeDaggerheart.ArmorDefault,
	}
	if input.Current != nil {
		stats = adversaryStats{
			HP:        input.Current.HP,
			HPMax:     input.Current.HPMax,
			Stress:    input.Current.Stress,
			StressMax: input.Current.StressMax,
			Evasion:   input.Current.Evasion,
			Major:     input.Current.Major,
			Severe:    input.Current.Severe,
			Armor:     input.Current.Armor,
		}
	}
	if input.HPMax != nil {
		stats.HPMax = int(input.HPMax.GetValue())
	}
	if input.HP != nil {
		stats.HP = int(input.HP.GetValue())
	} else if input.HPMax != nil && input.Current == nil {
		stats.HP = stats.HPMax
	} else if input.HPMax != nil && input.Current != nil && stats.HP > stats.HPMax {
		stats.HP = stats.HPMax
	}
	if input.StressMax != nil {
		stats.StressMax = int(input.StressMax.GetValue())
	}
	if input.Stress != nil {
		stats.Stress = int(input.Stress.GetValue())
	} else if input.StressMax != nil && input.Current == nil {
		stats.Stress = stats.StressMax
	} else if input.StressMax != nil && input.Current != nil && stats.Stress > stats.StressMax {
		stats.Stress = stats.StressMax
	}
	if input.Evasion != nil {
		stats.Evasion = int(input.Evasion.GetValue())
	}
	if input.Major != nil {
		stats.Major = int(input.Major.GetValue())
	}
	if input.Severe != nil {
		stats.Severe = int(input.Severe.GetValue())
	}
	if input.Armor != nil {
		stats.Armor = int(input.Armor.GetValue())
	}
	if stats.HPMax <= 0 {
		return adversaryStats{}, fmt.Errorf("hp_max must be positive")
	}
	if stats.HP < 0 || stats.HP > stats.HPMax {
		return adversaryStats{}, fmt.Errorf("hp must be in range 0..%d", stats.HPMax)
	}
	if stats.StressMax < 0 {
		return adversaryStats{}, fmt.Errorf("stress_max must be non-negative")
	}
	if stats.Stress < 0 || stats.Stress > stats.StressMax {
		return adversaryStats{}, fmt.Errorf("stress must be in range 0..%d", stats.StressMax)
	}
	if stats.Evasion < 0 {
		return adversaryStats{}, fmt.Errorf("evasion must be non-negative")
	}
	if stats.Major < 0 || stats.Severe < 0 {
		return adversaryStats{}, fmt.Errorf("thresholds must be non-negative")
	}
	if stats.Severe < stats.Major {
		return adversaryStats{}, fmt.Errorf("severe_threshold must be >= major_threshold")
	}
	if stats.Armor < 0 {
		return adversaryStats{}, fmt.Errorf("armor must be non-negative")
	}
	if input.RequireFields && (input.HP == nil || input.HPMax == nil) {
		return adversaryStats{}, fmt.Errorf("hp and hp_max are required")
	}
	return stats, nil
}

func daggerheartConditionsToProto(in []string) []pb.DaggerheartCondition {
	out := make([]pb.DaggerheartCondition, 0, len(in))
	for _, condition := range in {
		switch strings.ToLower(strings.TrimSpace(condition)) {
		case "hidden":
			out = append(out, pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)
		case "vulnerable":
			out = append(out, pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)
		}
	}
	return out
}
