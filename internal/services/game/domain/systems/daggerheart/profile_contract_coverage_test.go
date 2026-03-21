package daggerheart

import (
	"testing"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestCharacterSubclassTrack_Validate_Branches(t *testing.T) {
	// Valid primary track.
	track := daggerheartstate.CharacterSubclassTrack{
		Origin:     daggerheartstate.SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       daggerheartstate.SubclassTrackRankFoundation,
	}
	if err := track.Validate(); err != nil {
		t.Fatalf("valid primary track: %v", err)
	}

	// Invalid origin.
	track.Origin = "unknown"
	if err := track.Validate(); err == nil {
		t.Fatal("expected invalid origin error")
	}

	// Missing class_id.
	track.Origin = daggerheartstate.SubclassTrackOriginPrimary
	track.ClassID = ""
	if err := track.Validate(); err == nil {
		t.Fatal("expected missing class_id error")
	}

	// Missing subclass_id.
	track.ClassID = "class.guardian"
	track.SubclassID = ""
	if err := track.Validate(); err == nil {
		t.Fatal("expected missing subclass_id error")
	}

	// Invalid rank.
	track.SubclassID = "subclass.stalwart"
	track.Rank = "invalid"
	if err := track.Validate(); err == nil {
		t.Fatal("expected invalid rank error")
	}

	// All valid ranks.
	for _, rank := range []string{daggerheartstate.SubclassTrackRankFoundation, daggerheartstate.SubclassTrackRankSpecialization, daggerheartstate.SubclassTrackRankMastery} {
		track.Rank = rank
		if err := track.Validate(); err != nil {
			t.Fatalf("rank %q: %v", rank, err)
		}
	}

	// Multiclass requires domain_id.
	multiTrack := daggerheartstate.CharacterSubclassTrack{
		Origin:     daggerheartstate.SubclassTrackOriginMulticlass,
		ClassID:    "class.ranger",
		SubclassID: "subclass.wildbound",
		Rank:       daggerheartstate.SubclassTrackRankFoundation,
		DomainID:   "",
	}
	if err := multiTrack.Validate(); err == nil {
		t.Fatal("expected multiclass missing domain_id error")
	}
	multiTrack.DomainID = "domain.arcana"
	if err := multiTrack.Validate(); err != nil {
		t.Fatalf("valid multiclass track: %v", err)
	}
}

func TestValidateSubclassTracks_Branches(t *testing.T) {
	// Empty tracks is valid.
	if err := daggerheartstate.ValidateSubclassTracks("class.guardian", "subclass.stalwart", nil); err != nil {
		t.Fatalf("empty tracks: %v", err)
	}

	// Duplicate primary rejected.
	tracks := []daggerheartstate.CharacterSubclassTrack{
		{Origin: daggerheartstate.SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: daggerheartstate.SubclassTrackRankFoundation},
		{Origin: daggerheartstate.SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: daggerheartstate.SubclassTrackRankSpecialization},
	}
	if err := daggerheartstate.ValidateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err == nil {
		t.Fatal("expected duplicate primary error")
	}

	// Primary class mismatch.
	tracks = []daggerheartstate.CharacterSubclassTrack{
		{Origin: daggerheartstate.SubclassTrackOriginPrimary, ClassID: "class.ranger", SubclassID: "subclass.stalwart", Rank: daggerheartstate.SubclassTrackRankFoundation},
	}
	if err := daggerheartstate.ValidateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err == nil {
		t.Fatal("expected primary class mismatch error")
	}

	// Primary subclass mismatch.
	tracks = []daggerheartstate.CharacterSubclassTrack{
		{Origin: daggerheartstate.SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.wildbound", Rank: daggerheartstate.SubclassTrackRankFoundation},
	}
	if err := daggerheartstate.ValidateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err == nil {
		t.Fatal("expected primary subclass mismatch error")
	}

	// Missing primary when class and subclass are set.
	tracks = []daggerheartstate.CharacterSubclassTrack{
		{Origin: daggerheartstate.SubclassTrackOriginMulticlass, ClassID: "class.ranger", SubclassID: "subclass.wildbound", Rank: daggerheartstate.SubclassTrackRankFoundation, DomainID: "domain.arcana"},
	}
	if err := daggerheartstate.ValidateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err == nil {
		t.Fatal("expected missing primary track error")
	}

	// Valid with primary and multiclass.
	tracks = []daggerheartstate.CharacterSubclassTrack{
		{Origin: daggerheartstate.SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: daggerheartstate.SubclassTrackRankFoundation},
		{Origin: daggerheartstate.SubclassTrackOriginMulticlass, ClassID: "class.ranger", SubclassID: "subclass.wildbound", Rank: daggerheartstate.SubclassTrackRankFoundation, DomainID: "domain.arcana"},
	}
	if err := daggerheartstate.ValidateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err != nil {
		t.Fatalf("valid primary+multiclass tracks: %v", err)
	}
}

func TestCharacterProfile_Validate_SubclassTrackErrors(t *testing.T) {
	// Profile with invalid subclass track rejected.
	profile := validCharacterProfile()
	profile.SubclassTracks = []daggerheartstate.CharacterSubclassTrack{
		{Origin: "invalid", ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: daggerheartstate.SubclassTrackRankFoundation},
	}
	if err := profile.Validate(); err == nil {
		t.Fatal("expected subclass track validation error")
	}

	// Profile with valid tracks passes.
	profile.SubclassTracks = []daggerheartstate.CharacterSubclassTrack{
		{Origin: daggerheartstate.SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: daggerheartstate.SubclassTrackRankFoundation},
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("valid profile with tracks: %v", err)
	}
}

func TestNormalizedDamageDice(t *testing.T) {
	// nil returns nil.
	if got := daggerheartstate.NormalizedDamageDice(nil); got != nil {
		t.Fatalf("daggerheartstate.NormalizedDamageDice(nil) = %v, want nil", got)
	}
	// All invalid returns nil.
	if got := daggerheartstate.NormalizedDamageDice([]daggerheartstate.CharacterDamageDie{{Count: 0, Sides: 6}, {Count: 1, Sides: 0}}); got != nil {
		t.Fatalf("daggerheartstate.NormalizedDamageDice(all invalid) = %v, want nil", got)
	}
	// Mixed input filters correctly.
	got := daggerheartstate.NormalizedDamageDice([]daggerheartstate.CharacterDamageDie{{Count: 2, Sides: 6}, {Count: -1, Sides: 8}, {Count: 1, Sides: 10}})
	if len(got) != 2 || got[0].Count != 2 || got[0].Sides != 6 || got[1].Count != 1 || got[1].Sides != 10 {
		t.Fatalf("daggerheartstate.NormalizedDamageDice(mixed) = %v, want [{2 6} {1 10}]", got)
	}
}
