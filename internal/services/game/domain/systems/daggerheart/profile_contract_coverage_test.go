package daggerheart

import "testing"

func TestCharacterSubclassTrack_Validate_Branches(t *testing.T) {
	// Valid primary track.
	track := CharacterSubclassTrack{
		Origin:     SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       SubclassTrackRankFoundation,
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
	track.Origin = SubclassTrackOriginPrimary
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
	for _, rank := range []string{SubclassTrackRankFoundation, SubclassTrackRankSpecialization, SubclassTrackRankMastery} {
		track.Rank = rank
		if err := track.Validate(); err != nil {
			t.Fatalf("rank %q: %v", rank, err)
		}
	}

	// Multiclass requires domain_id.
	multiTrack := CharacterSubclassTrack{
		Origin:     SubclassTrackOriginMulticlass,
		ClassID:    "class.ranger",
		SubclassID: "subclass.wildbound",
		Rank:       SubclassTrackRankFoundation,
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
	if err := validateSubclassTracks("class.guardian", "subclass.stalwart", nil); err != nil {
		t.Fatalf("empty tracks: %v", err)
	}

	// Duplicate primary rejected.
	tracks := []CharacterSubclassTrack{
		{Origin: SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: SubclassTrackRankFoundation},
		{Origin: SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: SubclassTrackRankSpecialization},
	}
	if err := validateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err == nil {
		t.Fatal("expected duplicate primary error")
	}

	// Primary class mismatch.
	tracks = []CharacterSubclassTrack{
		{Origin: SubclassTrackOriginPrimary, ClassID: "class.ranger", SubclassID: "subclass.stalwart", Rank: SubclassTrackRankFoundation},
	}
	if err := validateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err == nil {
		t.Fatal("expected primary class mismatch error")
	}

	// Primary subclass mismatch.
	tracks = []CharacterSubclassTrack{
		{Origin: SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.wildbound", Rank: SubclassTrackRankFoundation},
	}
	if err := validateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err == nil {
		t.Fatal("expected primary subclass mismatch error")
	}

	// Missing primary when class and subclass are set.
	tracks = []CharacterSubclassTrack{
		{Origin: SubclassTrackOriginMulticlass, ClassID: "class.ranger", SubclassID: "subclass.wildbound", Rank: SubclassTrackRankFoundation, DomainID: "domain.arcana"},
	}
	if err := validateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err == nil {
		t.Fatal("expected missing primary track error")
	}

	// Valid with primary and multiclass.
	tracks = []CharacterSubclassTrack{
		{Origin: SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: SubclassTrackRankFoundation},
		{Origin: SubclassTrackOriginMulticlass, ClassID: "class.ranger", SubclassID: "subclass.wildbound", Rank: SubclassTrackRankFoundation, DomainID: "domain.arcana"},
	}
	if err := validateSubclassTracks("class.guardian", "subclass.stalwart", tracks); err != nil {
		t.Fatalf("valid primary+multiclass tracks: %v", err)
	}
}

func TestCharacterProfile_Validate_SubclassTrackErrors(t *testing.T) {
	// Profile with invalid subclass track rejected.
	profile := validCharacterProfile()
	profile.SubclassTracks = []CharacterSubclassTrack{
		{Origin: "invalid", ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: SubclassTrackRankFoundation},
	}
	if err := profile.Validate(); err == nil {
		t.Fatal("expected subclass track validation error")
	}

	// Profile with valid tracks passes.
	profile.SubclassTracks = []CharacterSubclassTrack{
		{Origin: SubclassTrackOriginPrimary, ClassID: "class.guardian", SubclassID: "subclass.stalwart", Rank: SubclassTrackRankFoundation},
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("valid profile with tracks: %v", err)
	}
}

func TestNormalizedDamageDice(t *testing.T) {
	// nil returns nil.
	if got := normalizedDamageDice(nil); got != nil {
		t.Fatalf("normalizedDamageDice(nil) = %v, want nil", got)
	}
	// All invalid returns nil.
	if got := normalizedDamageDice([]CharacterDamageDie{{Count: 0, Sides: 6}, {Count: 1, Sides: 0}}); got != nil {
		t.Fatalf("normalizedDamageDice(all invalid) = %v, want nil", got)
	}
	// Mixed input filters correctly.
	got := normalizedDamageDice([]CharacterDamageDie{{Count: 2, Sides: 6}, {Count: -1, Sides: 8}, {Count: 1, Sides: 10}})
	if len(got) != 2 || got[0].Count != 2 || got[0].Sides != 6 || got[1].Count != 1 || got[1].Sides != 10 {
		t.Fatalf("normalizedDamageDice(mixed) = %v, want [{2 6} {1 10}]", got)
	}
}

func TestConditionCodes(t *testing.T) {
	states := []ConditionState{
		{ID: "hidden", Code: "hidden", Standard: "hidden"},
		{ID: "custom-1", Code: "burning"},
		{ID: "tag-1", Standard: "special-standard"},
		{ID: "bare-id"},
	}
	codes := ConditionCodes(states)
	if len(codes) != 4 {
		t.Fatalf("ConditionCodes len = %d, want 4", len(codes))
	}
	// First uses Code.
	if codes[0] != "hidden" {
		t.Fatalf("codes[0] = %q, want %q", codes[0], "hidden")
	}
	// Second uses Code.
	if codes[1] != "burning" {
		t.Fatalf("codes[1] = %q, want %q", codes[1], "burning")
	}
	// Third uses Standard (since Code is empty).
	if codes[2] != "special-standard" {
		t.Fatalf("codes[2] = %q, want %q", codes[2], "special-standard")
	}
	// Fourth falls back to ID.
	if codes[3] != "bare-id" {
		t.Fatalf("codes[3] = %q, want %q", codes[3], "bare-id")
	}
	// nil returns nil.
	if got := ConditionCodes(nil); got != nil {
		t.Fatalf("ConditionCodes(nil) = %v, want nil", got)
	}
}

func TestDiffConditionStates_Branches(t *testing.T) {
	before := []ConditionState{
		{ID: "hidden", Code: "hidden"},
		{ID: "restrained", Code: "restrained"},
	}
	after := []ConditionState{
		{ID: "hidden", Code: "hidden"},
		{ID: "vulnerable", Code: "vulnerable"},
	}
	added, removed := DiffConditionStates(before, after)
	if len(added) != 1 || added[0].ID != "vulnerable" {
		t.Fatalf("added = %v, want [vulnerable]", added)
	}
	if len(removed) != 1 || removed[0].ID != "restrained" {
		t.Fatalf("removed = %v, want [restrained]", removed)
	}

	// No changes.
	added, removed = DiffConditionStates(before, before)
	if len(added) != 0 || len(removed) != 0 {
		t.Fatalf("same states: added=%d, removed=%d, want 0, 0", len(added), len(removed))
	}
}

func TestNormalizeConditionStates_DeduplicatesByStandard(t *testing.T) {
	states := []ConditionState{
		{ID: "hidden-1", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden"},
		{ID: "hidden-2", Class: ConditionClassStandard, Standard: "hidden", Code: "hidden", Label: "Hidden"},
		{ID: "custom-1", Class: ConditionClassSpecial, Code: "burning", Label: "Burning"},
	}
	got, err := NormalizeConditionStates(states)
	if err != nil {
		t.Fatalf("NormalizeConditionStates: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (dedup standard hidden)", len(got))
	}
}

func TestConditionStatesEqual_MismatchedTriggers(t *testing.T) {
	a := []ConditionState{
		{ID: "hidden", ClearTriggers: []ConditionClearTrigger{ConditionClearTriggerShortRest}},
	}
	b := []ConditionState{
		{ID: "hidden", ClearTriggers: []ConditionClearTrigger{ConditionClearTriggerLongRest}},
	}
	if ConditionStatesEqual(a, b) {
		t.Fatal("states with different triggers should not be equal")
	}

	// Different trigger count.
	c := []ConditionState{
		{ID: "hidden", ClearTriggers: []ConditionClearTrigger{ConditionClearTriggerShortRest, ConditionClearTriggerLongRest}},
	}
	if ConditionStatesEqual(a, c) {
		t.Fatal("states with different trigger counts should not be equal")
	}
}
