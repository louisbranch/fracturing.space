package daggerheart

import (
	"context"
	"errors"
	"testing"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func TestEnsurePrimarySubclassTrack_ReplacesOrAddsPrimaryTrack(t *testing.T) {
	t.Run("adds missing primary", func(t *testing.T) {
		tracks := daggerheartstate.EnsurePrimarySubclassTrack(nil, "class.guardian", "subclass.stalwart")
		if len(tracks) != 1 {
			t.Fatalf("track count = %d, want 1", len(tracks))
		}
		if tracks[0].Origin != daggerheartstate.SubclassTrackOriginPrimary || tracks[0].Rank != daggerheartstate.SubclassTrackRankFoundation {
			t.Fatalf("primary track = %+v", tracks[0])
		}
	})

	t.Run("replaces existing primary and preserves multiclass", func(t *testing.T) {
		tracks := daggerheartstate.EnsurePrimarySubclassTrack([]daggerheartstate.CharacterSubclassTrack{
			{Origin: daggerheartstate.SubclassTrackOriginPrimary, ClassID: "old", SubclassID: "old", Rank: daggerheartstate.SubclassTrackRankMastery},
			{Origin: daggerheartstate.SubclassTrackOriginMulticlass, ClassID: "class.bard", SubclassID: "subclass.wordsmith", Rank: daggerheartstate.SubclassTrackRankFoundation},
		}, "class.guardian", "subclass.stalwart")
		if len(tracks) != 2 {
			t.Fatalf("track count = %d, want 2", len(tracks))
		}
		if tracks[0].ClassID != "class.guardian" || tracks[0].SubclassID != "subclass.stalwart" || tracks[0].Rank != daggerheartstate.SubclassTrackRankFoundation {
			t.Fatalf("replaced primary = %+v", tracks[0])
		}
		if tracks[1].Origin != daggerheartstate.SubclassTrackOriginMulticlass {
			t.Fatalf("multiclass track lost: %+v", tracks[1])
		}
	})
}

func TestAdvancePrimarySubclassTrack_AdvancesAndStopsAtMastery(t *testing.T) {
	tracks, advanced, err := daggerheartstate.AdvancePrimarySubclassTrack([]daggerheartstate.CharacterSubclassTrack{{
		Origin:     daggerheartstate.SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       daggerheartstate.SubclassTrackRankFoundation,
	}})
	if err != nil {
		t.Fatalf("daggerheartstate.AdvancePrimarySubclassTrack returned error: %v", err)
	}
	if advanced.Rank != daggerheartstate.SubclassTrackRankSpecialization || tracks[0].Rank != daggerheartstate.SubclassTrackRankSpecialization {
		t.Fatalf("advanced track = %+v tracks=%+v", advanced, tracks)
	}

	_, _, err = daggerheartstate.AdvancePrimarySubclassTrack([]daggerheartstate.CharacterSubclassTrack{{
		Origin:     daggerheartstate.SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       daggerheartstate.SubclassTrackRankMastery,
	}})
	if err == nil {
		t.Fatal("expected mastery track to reject further advancement")
	}
}

func TestAddMulticlassSubclassTrack_RejectsDuplicate(t *testing.T) {
	_, track, err := daggerheartstate.AddMulticlassSubclassTrack([]daggerheartstate.CharacterSubclassTrack{{
		Origin:     daggerheartstate.SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       daggerheartstate.SubclassTrackRankFoundation,
	}}, "class.bard", "subclass.wordsmith", "domain.codex")
	if err != nil {
		t.Fatalf("daggerheartstate.AddMulticlassSubclassTrack returned error: %v", err)
	}
	if track.Origin != daggerheartstate.SubclassTrackOriginMulticlass || track.Rank != daggerheartstate.SubclassTrackRankFoundation || track.DomainID != "domain.codex" {
		t.Fatalf("multiclass track = %+v", track)
	}

	_, _, err = daggerheartstate.AddMulticlassSubclassTrack([]daggerheartstate.CharacterSubclassTrack{{
		Origin:     daggerheartstate.SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       daggerheartstate.SubclassTrackRankFoundation,
	}, {
		Origin:     daggerheartstate.SubclassTrackOriginMulticlass,
		ClassID:    "class.bard",
		SubclassID: "subclass.wordsmith",
		Rank:       daggerheartstate.SubclassTrackRankFoundation,
	}}, "class.seraph", "subclass.brave", "domain.splendor")
	if err == nil {
		t.Fatal("expected duplicate multiclass track rejection")
	}
}

func TestSubclassTrackHelpers_ErrorAndStageBranches(t *testing.T) {
	t.Run("advance requires primary track", func(t *testing.T) {
		if _, _, err := daggerheartstate.AdvancePrimarySubclassTrack(nil); err == nil {
			t.Fatal("expected missing primary track error")
		}
	})

	t.Run("loader is required when tracks exist", func(t *testing.T) {
		_, err := daggerheartstate.ActiveSubclassTrackFeaturesFromLoader(context.Background(), nil, []daggerheartstate.CharacterSubclassTrack{{
			Origin:     daggerheartstate.SubclassTrackOriginPrimary,
			ClassID:    "class.guardian",
			SubclassID: "subclass.stalwart",
			Rank:       daggerheartstate.SubclassTrackRankFoundation,
		}})
		if err == nil {
			t.Fatal("expected missing loader error")
		}
	})

	t.Run("store is required", func(t *testing.T) {
		_, err := daggerheartstate.ActiveSubclassTrackFeaturesFromStore(context.Background(), nil, []daggerheartstate.CharacterSubclassTrack{{
			Origin:     daggerheartstate.SubclassTrackOriginPrimary,
			ClassID:    "class.guardian",
			SubclassID: "subclass.stalwart",
			Rank:       daggerheartstate.SubclassTrackRankFoundation,
		}})
		if err == nil {
			t.Fatal("expected missing store error")
		}
	})

	t.Run("unlocked features follow requested rank", func(t *testing.T) {
		subclass := contentstore.DaggerheartSubclass{
			FoundationFeatures:     []contentstore.DaggerheartFeature{{ID: "foundation"}},
			SpecializationFeatures: []contentstore.DaggerheartFeature{{ID: "specialization"}},
			MasteryFeatures:        []contentstore.DaggerheartFeature{{ID: "mastery"}},
		}
		if got := daggerheartstate.UnlockedSubclassStageFeatures(subclass, daggerheartstate.SubclassTrackRankFoundation); len(got) != 1 || got[0].ID != "foundation" {
			t.Fatalf("foundation features = %+v", got)
		}
		if got := daggerheartstate.UnlockedSubclassStageFeatures(subclass, daggerheartstate.SubclassTrackRankSpecialization); len(got) != 1 || got[0].ID != "specialization" {
			t.Fatalf("specialization features = %+v", got)
		}
		if got := daggerheartstate.UnlockedSubclassStageFeatures(subclass, daggerheartstate.SubclassTrackRankMastery); len(got) != 1 || got[0].ID != "mastery" {
			t.Fatalf("mastery features = %+v", got)
		}
		if got := daggerheartstate.UnlockedSubclassStageFeatures(subclass, "unknown"); got != nil {
			t.Fatalf("unknown rank features = %+v, want nil", got)
		}
	})

	t.Run("next subclass rank helper", func(t *testing.T) {
		if got, ok := daggerheartstate.NextSubclassTrackRank(daggerheartstate.SubclassTrackRankFoundation); !ok || got != daggerheartstate.SubclassTrackRankSpecialization {
			t.Fatalf("foundation next = %q ok=%v", got, ok)
		}
		if got, ok := daggerheartstate.NextSubclassTrackRank(daggerheartstate.SubclassTrackRankSpecialization); !ok || got != daggerheartstate.SubclassTrackRankMastery {
			t.Fatalf("specialization next = %q ok=%v", got, ok)
		}
		if got, ok := daggerheartstate.NextSubclassTrackRank(daggerheartstate.SubclassTrackRankMastery); ok || got != "" {
			t.Fatalf("mastery next = %q ok=%v, want empty false", got, ok)
		}
	})
}

func TestActiveSubclassTrackFeaturesFromLoaderAndRuleSummaries(t *testing.T) {
	loader := func(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
		switch id {
		case "subclass.stalwart":
			return contentstore.DaggerheartSubclass{
				ID: "subclass.stalwart",
				FoundationFeatures: []contentstore.DaggerheartFeature{{
					ID: "feature.stalwart-unwavering",
					SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
						Kind:  contentstore.DaggerheartSubclassFeatureRuleKindHPSlotBonus,
						Bonus: 1,
					},
				}},
				SpecializationFeatures: []contentstore.DaggerheartFeature{{
					ID: "feature.stalwart-unrelenting",
					SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
						Kind:           contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus,
						Bonus:          1,
						ThresholdScope: contentstore.DaggerheartSubclassThresholdScopeAll,
					},
				}},
				MasteryFeatures: []contentstore.DaggerheartFeature{{
					ID: "feature.stalwart-undaunted",
					SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
						Kind:  contentstore.DaggerheartSubclassFeatureRuleKindStressSlotBonus,
						Bonus: 1,
					},
				}},
			}, nil
		case "subclass.school-war":
			return contentstore.DaggerheartSubclass{
				ID: "subclass.school-war",
				FoundationFeatures: []contentstore.DaggerheartFeature{{
					ID: "feature.school-war-conjure-shield",
					SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
						Kind:            contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonusWhileHopeAtLeast,
						Bonus:           2,
						RequiredHopeMin: 2,
					},
				}},
				SpecializationFeatures: []contentstore.DaggerheartFeature{{
					ID: "feature.school-war-face-your-fear",
					SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
						Kind:            contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear,
						DamageDiceCount: 1,
						DamageDieSides:  8,
					},
				}},
				MasteryFeatures: []contentstore.DaggerheartFeature{{
					ID: "feature.school-war-have-no-fear",
					SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
						Kind:            contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear,
						DamageDiceCount: 2,
						DamageDieSides:  8,
					},
				}},
			}, nil
		default:
			return contentstore.DaggerheartSubclass{}, errors.New("missing subclass")
		}
	}

	sets, err := daggerheartstate.ActiveSubclassTrackFeaturesFromLoader(context.Background(), loader, []daggerheartstate.CharacterSubclassTrack{
		{Origin: daggerheartstate.SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: daggerheartstate.SubclassTrackRankMastery},
		{Origin: daggerheartstate.SubclassTrackOriginMulticlass, ClassID: "class.wizard", SubclassID: "subclass.school-war", Rank: daggerheartstate.SubclassTrackRankSpecialization},
	})
	if err != nil {
		t.Fatalf("daggerheartstate.ActiveSubclassTrackFeaturesFromLoader returned error: %v", err)
	}
	if len(sets) != 2 {
		t.Fatalf("set count = %d, want 2", len(sets))
	}

	active := daggerheartstate.FlattenActiveSubclassFeatures(sets)
	bonuses := daggerheartstate.SubclassStatBonusesFromFeatures(active)
	if bonuses.HpMaxDelta != 1 || bonuses.StressMaxDelta != 1 || bonuses.MajorThresholdDelta != 1 || bonuses.SevereThresholdDelta != 1 {
		t.Fatalf("bonuses = %+v", bonuses)
	}

	profile := &daggerheartstate.CharacterProfile{HpMax: 6, StressMax: 5, MajorThreshold: 3, SevereThreshold: 6}
	daggerheartstate.ApplySubclassStatBonuses(profile, bonuses)
	if profile.HpMax != 7 || profile.StressMax != 6 || profile.MajorThreshold != 4 || profile.SevereThreshold != 7 {
		t.Fatalf("profile after subclass bonuses = %+v", profile)
	}

	summary := daggerheartstate.SummarizeActiveSubclassRules(active)
	if summary.EvasionBonusWhileHopeAtLeast != 2 || summary.EvasionBonusRequiredHopeMin != 2 {
		t.Fatalf("evasion summary = %+v", summary)
	}
	if summary.BonusMagicDamageDiceCount != 1 || summary.BonusMagicDamageDieSides != 8 {
		t.Fatalf("magic damage summary = %+v", summary)
	}
}

func TestSummarizeActiveSubclassRules_PrefersHighestSupportedValues(t *testing.T) {
	features := []contentstore.DaggerheartFeature{
		{
			ID: "feature.low-hope",
			SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
				Kind:  contentstore.DaggerheartSubclassFeatureRuleKindGainHopeOnFailureWithFear,
				Bonus: 1,
			},
		},
		{
			ID: "feature.high-hope",
			SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
				Kind:  contentstore.DaggerheartSubclassFeatureRuleKindGainHopeOnFailureWithFear,
				Bonus: 2,
			},
		},
		{
			ID: "feature.low-evasion",
			SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
				Kind:            contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonusWhileHopeAtLeast,
				Bonus:           1,
				RequiredHopeMin: 1,
			},
		},
		{
			ID: "feature.high-evasion",
			SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
				Kind:            contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonusWhileHopeAtLeast,
				Bonus:           2,
				RequiredHopeMin: 2,
			},
		},
		{
			ID: "feature.fixed-vulnerable",
			SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
				Kind:  contentstore.DaggerheartSubclassFeatureRuleKindBonusDamageWhileVulnerable,
				Bonus: 3,
			},
		},
		{
			ID: "feature.level-vulnerable",
			SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
				Kind:              contentstore.DaggerheartSubclassFeatureRuleKindBonusDamageWhileVulnerable,
				UseCharacterLevel: true,
			},
		},
	}

	summary := daggerheartstate.SummarizeActiveSubclassRules(features)
	if summary.GainHopeOnFailureWithFearAmount != 2 {
		t.Fatalf("gain hope summary = %+v", summary)
	}
	if summary.EvasionBonusWhileHopeAtLeast != 2 || summary.EvasionBonusRequiredHopeMin != 2 {
		t.Fatalf("evasion summary = %+v", summary)
	}
	if !summary.BonusDamageWhileVulnerableLevel || summary.BonusDamageWhileVulnerable != 0 {
		t.Fatalf("vulnerable summary = %+v", summary)
	}
}
