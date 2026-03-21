package state

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

// SubclassStatBonuses captures permanent stat deltas granted by newly unlocked
// subclass stages.
type SubclassStatBonuses struct {
	HpMaxDelta           int
	StressMaxDelta       int
	EvasionDelta         int
	MajorThresholdDelta  int
	SevereThresholdDelta int
}

// ActiveSubclassTrackFeatures groups the active features derived from one
// subclass track by stage so read surfaces can render unlocked progression.
type ActiveSubclassTrackFeatures struct {
	Track                  CharacterSubclassTrack
	FoundationFeatures     []contentstore.DaggerheartFeature
	SpecializationFeatures []contentstore.DaggerheartFeature
	MasteryFeatures        []contentstore.DaggerheartFeature
}

// ActiveSubclassRuleSummary aggregates the automatic runtime hooks derived from
// the currently active subclass features.
type ActiveSubclassRuleSummary struct {
	GainHopeOnFailureWithFearAmount int
	BonusMagicDamageDiceCount       int
	BonusMagicDamageDieSides        int
	EvasionBonusWhileHopeAtLeast    int
	EvasionBonusRequiredHopeMin     int
	BonusDamageWhileVulnerable      int
	BonusDamageWhileVulnerableLevel bool
}

// PrimarySubclassTrack returns the authoritative primary subclass track.
func PrimarySubclassTrack(tracks []CharacterSubclassTrack) (CharacterSubclassTrack, int, bool) {
	for i, track := range tracks {
		if strings.TrimSpace(track.Origin) == SubclassTrackOriginPrimary {
			return track, i, true
		}
	}
	return CharacterSubclassTrack{}, -1, false
}

// AdvancePrimarySubclassTrack moves the primary track to its next stage.
func AdvancePrimarySubclassTrack(tracks []CharacterSubclassTrack) ([]CharacterSubclassTrack, CharacterSubclassTrack, error) {
	items := append([]CharacterSubclassTrack(nil), tracks...)
	track, idx, ok := PrimarySubclassTrack(items)
	if !ok {
		return nil, CharacterSubclassTrack{}, fmt.Errorf("primary subclass track is required")
	}
	nextRank, ok := nextSubclassTrackRank(track.Rank)
	if !ok {
		return nil, CharacterSubclassTrack{}, fmt.Errorf("primary subclass track has no next rank")
	}
	track.Rank = nextRank
	items[idx] = track
	return items, track, nil
}

// AddMulticlassSubclassTrack adds the secondary subclass foundation track.
func AddMulticlassSubclassTrack(tracks []CharacterSubclassTrack, classID, subclassID, domainID string) ([]CharacterSubclassTrack, CharacterSubclassTrack, error) {
	items := append([]CharacterSubclassTrack(nil), tracks...)
	for _, track := range items {
		if strings.TrimSpace(track.Origin) == SubclassTrackOriginMulticlass {
			return nil, CharacterSubclassTrack{}, fmt.Errorf("multiclass subclass track already exists")
		}
	}
	track := CharacterSubclassTrack{
		Origin:     SubclassTrackOriginMulticlass,
		ClassID:    strings.TrimSpace(classID),
		SubclassID: strings.TrimSpace(subclassID),
		Rank:       SubclassTrackRankFoundation,
		DomainID:   strings.TrimSpace(domainID),
	}
	if err := track.Validate(); err != nil {
		return nil, CharacterSubclassTrack{}, err
	}
	items = append(items, track)
	return items, track, nil
}

// EnsurePrimarySubclassTrack seeds or replaces the primary subclass track with
// the provided foundation-stage identity.
func EnsurePrimarySubclassTrack(tracks []CharacterSubclassTrack, classID, subclassID string) []CharacterSubclassTrack {
	items := make([]CharacterSubclassTrack, 0, len(tracks)+1)
	replaced := false
	for _, track := range tracks {
		if strings.TrimSpace(track.Origin) == SubclassTrackOriginPrimary {
			items = append(items, CharacterSubclassTrack{
				Origin:     SubclassTrackOriginPrimary,
				ClassID:    strings.TrimSpace(classID),
				SubclassID: strings.TrimSpace(subclassID),
				Rank:       SubclassTrackRankFoundation,
			})
			replaced = true
			continue
		}
		items = append(items, track)
	}
	if !replaced {
		items = append(items, CharacterSubclassTrack{
			Origin:     SubclassTrackOriginPrimary,
			ClassID:    strings.TrimSpace(classID),
			SubclassID: strings.TrimSpace(subclassID),
			Rank:       SubclassTrackRankFoundation,
		})
	}
	return items
}

// ActiveSubclassTrackFeaturesFromLoader loads the subclass catalog entries for
// each track and groups the active stage features for rendering and runtime
// rule aggregation.
func ActiveSubclassTrackFeaturesFromLoader(ctx context.Context, loadSubclass func(context.Context, string) (contentstore.DaggerheartSubclass, error), tracks []CharacterSubclassTrack) ([]ActiveSubclassTrackFeatures, error) {
	if len(tracks) == 0 {
		return nil, nil
	}
	if loadSubclass == nil {
		return nil, fmt.Errorf("subclass loader is required")
	}
	result := make([]ActiveSubclassTrackFeatures, 0, len(tracks))
	for _, track := range tracks {
		subclassID := strings.TrimSpace(track.SubclassID)
		if subclassID == "" {
			continue
		}
		subclass, err := loadSubclass(ctx, subclassID)
		if err != nil {
			return nil, err
		}
		features := ActiveSubclassTrackFeatures{
			Track:              track,
			FoundationFeatures: append([]contentstore.DaggerheartFeature(nil), subclass.FoundationFeatures...),
		}
		switch strings.TrimSpace(track.Rank) {
		case SubclassTrackRankSpecialization:
			features.SpecializationFeatures = append([]contentstore.DaggerheartFeature(nil), subclass.SpecializationFeatures...)
		case SubclassTrackRankMastery:
			features.SpecializationFeatures = append([]contentstore.DaggerheartFeature(nil), subclass.SpecializationFeatures...)
			features.MasteryFeatures = append([]contentstore.DaggerheartFeature(nil), subclass.MasteryFeatures...)
		}
		result = append(result, features)
	}
	return result, nil
}

// ActiveSubclassTrackFeaturesFromStore adapts the shared content store to the
// smaller loader consumed by subclass-track derivation helpers.
func ActiveSubclassTrackFeaturesFromStore(ctx context.Context, store contentstore.DaggerheartContentReadStore, tracks []CharacterSubclassTrack) ([]ActiveSubclassTrackFeatures, error) {
	if store == nil {
		return nil, fmt.Errorf("subclass store is required")
	}
	return ActiveSubclassTrackFeaturesFromLoader(ctx, store.GetDaggerheartSubclass, tracks)
}

// UnlockedSubclassStageFeatures returns only the features granted by the newly
// unlocked stage.
func UnlockedSubclassStageFeatures(subclass contentstore.DaggerheartSubclass, rank string) []contentstore.DaggerheartFeature {
	switch strings.TrimSpace(rank) {
	case SubclassTrackRankFoundation:
		return append([]contentstore.DaggerheartFeature(nil), subclass.FoundationFeatures...)
	case SubclassTrackRankSpecialization:
		return append([]contentstore.DaggerheartFeature(nil), subclass.SpecializationFeatures...)
	case SubclassTrackRankMastery:
		return append([]contentstore.DaggerheartFeature(nil), subclass.MasteryFeatures...)
	default:
		return nil
	}
}

// FlattenActiveSubclassFeatures returns every currently active subclass feature.
func FlattenActiveSubclassFeatures(sets []ActiveSubclassTrackFeatures) []contentstore.DaggerheartFeature {
	if len(sets) == 0 {
		return nil
	}
	features := make([]contentstore.DaggerheartFeature, 0)
	for _, set := range sets {
		features = append(features, set.FoundationFeatures...)
		features = append(features, set.SpecializationFeatures...)
		features = append(features, set.MasteryFeatures...)
	}
	return features
}

// SubclassStatBonusesFromFeatures converts permanent subclass passives into
// concrete profile deltas.
func SubclassStatBonusesFromFeatures(features []contentstore.DaggerheartFeature) SubclassStatBonuses {
	var bonuses SubclassStatBonuses
	for _, feature := range features {
		rule := feature.SubclassRule
		if rule == nil {
			continue
		}
		switch rule.Kind {
		case contentstore.DaggerheartSubclassFeatureRuleKindHPSlotBonus:
			bonuses.HpMaxDelta += rule.Bonus
		case contentstore.DaggerheartSubclassFeatureRuleKindStressSlotBonus:
			bonuses.StressMaxDelta += rule.Bonus
		case contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonus:
			bonuses.EvasionDelta += rule.Bonus
		case contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus:
			switch rule.ThresholdScope {
			case contentstore.DaggerheartSubclassThresholdScopeSevereOnly:
				bonuses.SevereThresholdDelta += rule.Bonus
			default:
				bonuses.MajorThresholdDelta += rule.Bonus
				bonuses.SevereThresholdDelta += rule.Bonus
			}
		}
	}
	return bonuses
}

// ApplySubclassStatBonuses applies permanent subclass deltas directly to a
// profile snapshot.
func ApplySubclassStatBonuses(profile *CharacterProfile, bonuses SubclassStatBonuses) {
	if profile == nil {
		return
	}
	profile.HpMax += bonuses.HpMaxDelta
	profile.StressMax += bonuses.StressMaxDelta
	profile.Evasion += bonuses.EvasionDelta
	profile.MajorThreshold += bonuses.MajorThresholdDelta
	profile.SevereThreshold += bonuses.SevereThresholdDelta
}

// SummarizeActiveSubclassRules reduces the active feature list into the
// runtime-triggered rule set used by roll, attack, and outcome flows.
func SummarizeActiveSubclassRules(features []contentstore.DaggerheartFeature) ActiveSubclassRuleSummary {
	var summary ActiveSubclassRuleSummary
	for _, feature := range features {
		rule := feature.SubclassRule
		if rule == nil {
			continue
		}
		switch rule.Kind {
		case contentstore.DaggerheartSubclassFeatureRuleKindGainHopeOnFailureWithFear:
			if rule.Bonus > summary.GainHopeOnFailureWithFearAmount {
				summary.GainHopeOnFailureWithFearAmount = rule.Bonus
			}
		case contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear:
			if rule.DamageDiceCount > summary.BonusMagicDamageDiceCount ||
				(rule.DamageDiceCount == summary.BonusMagicDamageDiceCount && rule.DamageDieSides > summary.BonusMagicDamageDieSides) {
				summary.BonusMagicDamageDiceCount = rule.DamageDiceCount
				summary.BonusMagicDamageDieSides = rule.DamageDieSides
			}
		case contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonusWhileHopeAtLeast:
			if rule.Bonus > summary.EvasionBonusWhileHopeAtLeast {
				summary.EvasionBonusWhileHopeAtLeast = rule.Bonus
				summary.EvasionBonusRequiredHopeMin = rule.RequiredHopeMin
			}
		case contentstore.DaggerheartSubclassFeatureRuleKindBonusDamageWhileVulnerable:
			if rule.UseCharacterLevel {
				summary.BonusDamageWhileVulnerableLevel = true
				summary.BonusDamageWhileVulnerable = 0
				continue
			}
			if rule.Bonus > summary.BonusDamageWhileVulnerable {
				summary.BonusDamageWhileVulnerable = rule.Bonus
			}
		}
	}
	return summary
}

// NextSubclassTrackRank returns the next subclass track rank, or ("", false)
// if already at mastery.
func NextSubclassTrackRank(rank string) (string, bool) {
	return nextSubclassTrackRank(rank)
}

func nextSubclassTrackRank(rank string) (string, bool) {
	switch strings.TrimSpace(rank) {
	case SubclassTrackRankFoundation:
		return SubclassTrackRankSpecialization, true
	case SubclassTrackRankSpecialization:
		return SubclassTrackRankMastery, true
	default:
		return "", false
	}
}
