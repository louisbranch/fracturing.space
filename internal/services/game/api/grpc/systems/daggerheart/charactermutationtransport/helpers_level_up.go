package charactermutationtransport

import (
	"context"
	"strings"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// intPtr returns a pointer to the provided int value.
func intPtr(v int) *int {
	return &v
}

// subclassTracksFromProjection converts projection-layer subclass tracks into
// domain-layer tracks consumed by the level-up pipeline.
func subclassTracksFromProjection(tracks []projectionstore.DaggerheartSubclassTrack) []daggerheartstate.CharacterSubclassTrack {
	if len(tracks) == 0 {
		return nil
	}
	out := make([]daggerheartstate.CharacterSubclassTrack, 0, len(tracks))
	for _, track := range tracks {
		out = append(out, daggerheartstate.CharacterSubclassTrack{
			Origin:     string(track.Origin),
			ClassID:    track.ClassID,
			SubclassID: track.SubclassID,
			Rank:       string(track.Rank),
			DomainID:   track.DomainID,
		})
	}
	return out
}

// deriveLevelUpSubclassProgression resolves the subclass tracks and permanent
// stat bonuses resulting from the level-up advancement choices.  It advances
// the primary track when a "subclass_advance" advancement is present, and adds
// a multiclass track when a "multiclass" advancement is present.
func (h *Handler) deriveLevelUpSubclassProgression(
	ctx context.Context,
	profile projectionstore.DaggerheartCharacterProfile,
	advancements []daggerheartpayload.LevelUpAdvancementPayload,
) ([]daggerheartstate.CharacterSubclassTrack, daggerheartstate.SubclassStatBonuses, error) {
	previousTracks := subclassTracksFromProjection(profile.SubclassTracks)
	tracks := append([]daggerheartstate.CharacterSubclassTrack(nil), previousTracks...)

	for _, adv := range advancements {
		switch strings.TrimSpace(adv.Type) {
		case "upgraded_subclass":
			next, _, err := daggerheartstate.AdvancePrimarySubclassTrack(tracks)
			if err != nil {
				return nil, daggerheartstate.SubclassStatBonuses{}, status.Errorf(codes.FailedPrecondition, "advance primary subclass: %v", err)
			}
			tracks = next

		case "multiclass":
			if adv.Multiclass == nil {
				return nil, daggerheartstate.SubclassStatBonuses{}, status.Error(codes.InvalidArgument, "multiclass advancement requires multiclass payload")
			}
			classID := strings.TrimSpace(adv.Multiclass.SecondaryClassID)
			subclassID := strings.TrimSpace(adv.Multiclass.SecondarySubclassID)
			domainID := strings.TrimSpace(adv.Multiclass.DomainID)
			next, _, err := daggerheartstate.AddMulticlassSubclassTrack(tracks, classID, subclassID, domainID)
			if err != nil {
				return nil, daggerheartstate.SubclassStatBonuses{}, status.Errorf(codes.FailedPrecondition, "add multiclass subclass: %v", err)
			}
			tracks = next
		}
	}

	if h.deps.Content == nil {
		return tracks, daggerheartstate.SubclassStatBonuses{}, nil
	}

	activeFeatureSets, err := daggerheartstate.ActiveSubclassTrackFeaturesFromLoader(
		ctx,
		h.deps.Content.GetDaggerheartSubclass,
		tracks,
	)
	if err != nil {
		return nil, daggerheartstate.SubclassStatBonuses{}, status.Errorf(codes.Internal, "load subclass features: %v", err)
	}

	allFeatures := daggerheartstate.FlattenActiveSubclassFeatures(activeFeatureSets)
	bonuses := daggerheartstate.SubclassStatBonusesFromFeatures(allFeatures)

	// Subtract the bonuses already applied from previous level-ups so we emit
	// only the delta for this level transition.
	previousFeatureSets, err := daggerheartstate.ActiveSubclassTrackFeaturesFromLoader(
		ctx,
		h.deps.Content.GetDaggerheartSubclass,
		previousTracks,
	)
	if err != nil {
		return nil, daggerheartstate.SubclassStatBonuses{}, status.Errorf(codes.Internal, "load previous subclass features: %v", err)
	}
	previousFeatures := daggerheartstate.FlattenActiveSubclassFeatures(previousFeatureSets)
	previousBonuses := daggerheartstate.SubclassStatBonusesFromFeatures(previousFeatures)

	return tracks, daggerheartstate.SubclassStatBonuses{
		HpMaxDelta:           bonuses.HpMaxDelta - previousBonuses.HpMaxDelta,
		StressMaxDelta:       bonuses.StressMaxDelta - previousBonuses.StressMaxDelta,
		EvasionDelta:         bonuses.EvasionDelta - previousBonuses.EvasionDelta,
		MajorThresholdDelta:  bonuses.MajorThresholdDelta - previousBonuses.MajorThresholdDelta,
		SevereThresholdDelta: bonuses.SevereThresholdDelta - previousBonuses.SevereThresholdDelta,
	}, nil
}
