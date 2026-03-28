package statmodifiertransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- StatModifierFromProto ---

func TestStatModifierFromProto_NilInput(t *testing.T) {
	_, err := StatModifierFromProto(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestStatModifierFromProto_EmptyID(t *testing.T) {
	_, err := StatModifierFromProto(&pb.DaggerheartStatModifier{
		Id:     "",
		Target: "evasion",
	})
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestStatModifierFromProto_WhitespaceOnlyID(t *testing.T) {
	_, err := StatModifierFromProto(&pb.DaggerheartStatModifier{
		Id:     "   ",
		Target: "evasion",
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only id")
	}
}

func TestStatModifierFromProto_InvalidTarget(t *testing.T) {
	_, err := StatModifierFromProto(&pb.DaggerheartStatModifier{
		Id:     "mod-1",
		Target: "charisma",
	})
	if err == nil {
		t.Fatal("expected error for invalid target")
	}
}

func TestStatModifierFromProto_UnspecifiedClearTrigger(t *testing.T) {
	_, err := StatModifierFromProto(&pb.DaggerheartStatModifier{
		Id:     "mod-1",
		Target: "evasion",
		Delta:  2,
		ClearTriggers: []pb.DaggerheartConditionClearTrigger{
			pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED,
		},
	})
	if err == nil {
		t.Fatal("expected error for unspecified clear trigger")
	}
}

func TestStatModifierFromProto_ValidModifierAllFields(t *testing.T) {
	view, err := StatModifierFromProto(&pb.DaggerheartStatModifier{
		Id:       "mod-1",
		Target:   "evasion",
		Delta:    -2,
		Label:    " Shield Spell ",
		Source:   " spell ",
		SourceId: " spell-42 ",
		ClearTriggers: []pb.DaggerheartConditionClearTrigger{
			pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST,
			pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.ID != "mod-1" {
		t.Fatalf("ID = %q, want %q", view.ID, "mod-1")
	}
	if view.Target != "evasion" {
		t.Fatalf("Target = %q, want %q", view.Target, "evasion")
	}
	if view.Delta != -2 {
		t.Fatalf("Delta = %d, want -2", view.Delta)
	}
	if view.Label != "Shield Spell" {
		t.Fatalf("Label = %q, want %q", view.Label, "Shield Spell")
	}
	if view.Source != "spell" {
		t.Fatalf("Source = %q, want %q", view.Source, "spell")
	}
	if view.SourceID != "spell-42" {
		t.Fatalf("SourceID = %q, want %q", view.SourceID, "spell-42")
	}
	if len(view.ClearTriggers) != 2 {
		t.Fatalf("ClearTriggers len = %d, want 2", len(view.ClearTriggers))
	}
	if view.ClearTriggers[0] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST {
		t.Fatalf("ClearTriggers[0] = %v, want SHORT_REST", view.ClearTriggers[0])
	}
}

func TestStatModifierFromProto_AllValidTargets(t *testing.T) {
	targets := []string{
		"evasion", "major_threshold", "severe_threshold",
		"proficiency", "armor_score",
		"strength", "finesse", "agility", "instinct", "presence", "knowledge",
	}
	for _, target := range targets {
		view, err := StatModifierFromProto(&pb.DaggerheartStatModifier{
			Id:     "mod-" + target,
			Target: target,
			Delta:  1,
		})
		if err != nil {
			t.Fatalf("target %q: unexpected error: %v", target, err)
		}
		if view.Target != target {
			t.Fatalf("target %q: got %q", target, view.Target)
		}
	}
}

// --- StatModifiersFromProto ---

func TestStatModifiersFromProto_NilSlice(t *testing.T) {
	views, err := StatModifiersFromProto(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if views != nil {
		t.Fatalf("expected nil, got %v", views)
	}
}

func TestStatModifiersFromProto_EmptySlice(t *testing.T) {
	views, err := StatModifiersFromProto([]*pb.DaggerheartStatModifier{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if views != nil {
		t.Fatalf("expected nil, got %v", views)
	}
}

func TestStatModifiersFromProto_PropagatesError(t *testing.T) {
	_, err := StatModifiersFromProto([]*pb.DaggerheartStatModifier{
		{Id: "mod-1", Target: "evasion", Delta: 1},
		nil, // triggers error
	})
	if err == nil {
		t.Fatal("expected error for nil element in slice")
	}
}

// --- StatModifierViewsToDomain ---

func TestStatModifierViewsToDomain_NilSlice(t *testing.T) {
	got := StatModifierViewsToDomain(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestStatModifierViewsToDomain_SkipsNilEntries(t *testing.T) {
	got := StatModifierViewsToDomain([]*StatModifierView{
		nil,
		{ID: "mod-1", Target: "evasion", Delta: 1},
		nil,
	})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].ID != "mod-1" {
		t.Fatalf("ID = %q, want %q", got[0].ID, "mod-1")
	}
}

func TestStatModifierViewsToDomain_ConvertsAllFields(t *testing.T) {
	views := []*StatModifierView{
		{
			ID:       "mod-1",
			Target:   "agility",
			Delta:    3,
			Label:    "Haste",
			Source:   "spell",
			SourceID: "sp-1",
			ClearTriggers: []pb.DaggerheartConditionClearTrigger{
				pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST,
				pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN,
			},
		},
	}
	domain := StatModifierViewsToDomain(views)
	if len(domain) != 1 {
		t.Fatalf("len = %d, want 1", len(domain))
	}
	d := domain[0]
	if d.ID != "mod-1" {
		t.Fatalf("ID = %q, want %q", d.ID, "mod-1")
	}
	if d.Target != rules.StatModifierTargetAgility {
		t.Fatalf("Target = %q, want %q", d.Target, rules.StatModifierTargetAgility)
	}
	if d.Delta != 3 {
		t.Fatalf("Delta = %d, want 3", d.Delta)
	}
	if d.Label != "Haste" {
		t.Fatalf("Label = %q, want %q", d.Label, "Haste")
	}
	if d.Source != "spell" {
		t.Fatalf("Source = %q, want %q", d.Source, "spell")
	}
	if d.SourceID != "sp-1" {
		t.Fatalf("SourceID = %q, want %q", d.SourceID, "sp-1")
	}
	if len(d.ClearTriggers) != 2 {
		t.Fatalf("ClearTriggers len = %d, want 2", len(d.ClearTriggers))
	}
	if d.ClearTriggers[0] != rules.ConditionClearTriggerShortRest {
		t.Fatalf("ClearTriggers[0] = %q, want %q", d.ClearTriggers[0], rules.ConditionClearTriggerShortRest)
	}
	if d.ClearTriggers[1] != rules.ConditionClearTriggerDamageTaken {
		t.Fatalf("ClearTriggers[1] = %q, want %q", d.ClearTriggers[1], rules.ConditionClearTriggerDamageTaken)
	}
}

// --- DomainStatModifiersToViews ---

func TestDomainStatModifiersToViews_NilSlice(t *testing.T) {
	got := DomainStatModifiersToViews(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestDomainStatModifiersToViews_ConvertsAllFields(t *testing.T) {
	domain := []rules.StatModifierState{
		{
			ID:            "mod-1",
			Target:        rules.StatModifierTargetEvasion,
			Delta:         -1,
			Label:         "Curse",
			Source:        "enemy",
			SourceID:      "en-1",
			ClearTriggers: []rules.ConditionClearTrigger{rules.ConditionClearTriggerLongRest},
		},
	}
	views := DomainStatModifiersToViews(domain)
	if len(views) != 1 {
		t.Fatalf("len = %d, want 1", len(views))
	}
	v := views[0]
	if v.ID != "mod-1" || v.Target != "evasion" || v.Delta != -1 {
		t.Fatalf("unexpected view: %+v", v)
	}
	if v.Label != "Curse" || v.Source != "enemy" || v.SourceID != "en-1" {
		t.Fatalf("unexpected view metadata: %+v", v)
	}
	if len(v.ClearTriggers) != 1 || v.ClearTriggers[0] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST {
		t.Fatalf("ClearTriggers = %v, want [LONG_REST]", v.ClearTriggers)
	}
}

// --- Proto/domain/view round-trip ---

func TestRoundTrip_ProtoToDomainToView(t *testing.T) {
	original := &pb.DaggerheartStatModifier{
		Id:       "mod-rt",
		Target:   "proficiency",
		Delta:    5,
		Label:    "Bless",
		Source:   "cleric",
		SourceId: "cl-1",
		ClearTriggers: []pb.DaggerheartConditionClearTrigger{
			pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END,
		},
	}

	view, err := StatModifierFromProto(original)
	if err != nil {
		t.Fatalf("StatModifierFromProto: %v", err)
	}

	domain := StatModifierViewsToDomain([]*StatModifierView{view})
	if len(domain) != 1 {
		t.Fatalf("domain len = %d, want 1", len(domain))
	}
	d := domain[0]
	if d.ID != "mod-rt" || d.Target != rules.StatModifierTargetProficiency || d.Delta != 5 {
		t.Fatalf("domain = %+v", d)
	}
	if len(d.ClearTriggers) != 1 || d.ClearTriggers[0] != rules.ConditionClearTriggerSessionEnd {
		t.Fatalf("domain triggers = %v, want [session_end]", d.ClearTriggers)
	}

	backToViews := DomainStatModifiersToViews(domain)
	if len(backToViews) != 1 {
		t.Fatalf("back views len = %d, want 1", len(backToViews))
	}
	bv := backToViews[0]
	if bv.ID != view.ID || bv.Target != view.Target || bv.Delta != view.Delta {
		t.Fatalf("round-trip mismatch: got %+v", bv)
	}

	protos := StatModifierViewsToProto(backToViews)
	if len(protos) != 1 {
		t.Fatalf("proto len = %d, want 1", len(protos))
	}
	p := protos[0]
	if p.GetId() != original.GetId() || p.GetTarget() != original.GetTarget() || p.GetDelta() != original.GetDelta() {
		t.Fatalf("proto round-trip mismatch: %+v", p)
	}
}

// --- StatModifierViewsToProto ---

func TestStatModifierViewsToProto_NilSlice(t *testing.T) {
	got := StatModifierViewsToProto(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestStatModifierViewsToProto_SkipsNilEntries(t *testing.T) {
	got := StatModifierViewsToProto([]*StatModifierView{
		nil,
		{ID: "mod-1", Target: "evasion", Delta: 1},
	})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].GetId() != "mod-1" {
		t.Fatalf("ID = %q, want %q", got[0].GetId(), "mod-1")
	}
}

// --- ProjectionStatModifiersToViews ---

func TestProjectionStatModifiersToViews_NilSlice(t *testing.T) {
	got := ProjectionStatModifiersToViews(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestProjectionStatModifiersToViews_ConvertsFields(t *testing.T) {
	proj := []projectionstore.DaggerheartStatModifier{
		{
			ID:            "mod-p",
			Target:        "strength",
			Delta:         2,
			Label:         "Rage",
			Source:        "ability",
			SourceID:      "ab-1",
			ClearTriggers: []string{"short_rest"},
		},
	}
	views := ProjectionStatModifiersToViews(proj)
	if len(views) != 1 {
		t.Fatalf("len = %d, want 1", len(views))
	}
	v := views[0]
	if v.ID != "mod-p" || v.Target != "strength" || v.Delta != 2 {
		t.Fatalf("unexpected view: %+v", v)
	}
	if v.Label != "Rage" || v.Source != "ability" || v.SourceID != "ab-1" {
		t.Fatalf("unexpected metadata: %+v", v)
	}
	if len(v.ClearTriggers) != 1 || v.ClearTriggers[0] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST {
		t.Fatalf("ClearTriggers = %v, want [SHORT_REST]", v.ClearTriggers)
	}
}

// --- ProjectionStatModifiersToDomain ---

func TestProjectionStatModifiersToDomain_NilSlice(t *testing.T) {
	got := ProjectionStatModifiersToDomain(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestProjectionStatModifiersToDomain_ConvertsFields(t *testing.T) {
	proj := []projectionstore.DaggerheartStatModifier{
		{
			ID:            "mod-p2",
			Target:        "finesse",
			Delta:         -1,
			Label:         "Weaken",
			Source:        "trap",
			SourceID:      "tr-1",
			ClearTriggers: []string{"long_rest", "damage_taken"},
		},
	}
	domain := ProjectionStatModifiersToDomain(proj)
	if len(domain) != 1 {
		t.Fatalf("len = %d, want 1", len(domain))
	}
	d := domain[0]
	if d.ID != "mod-p2" || d.Target != rules.StatModifierTargetFinesse || d.Delta != -1 {
		t.Fatalf("unexpected domain: %+v", d)
	}
	if len(d.ClearTriggers) != 2 {
		t.Fatalf("ClearTriggers len = %d, want 2", len(d.ClearTriggers))
	}
	if d.ClearTriggers[0] != rules.ConditionClearTriggerLongRest {
		t.Fatalf("ClearTriggers[0] = %q, want %q", d.ClearTriggers[0], rules.ConditionClearTriggerLongRest)
	}
	if d.ClearTriggers[1] != rules.ConditionClearTriggerDamageTaken {
		t.Fatalf("ClearTriggers[1] = %q, want %q", d.ClearTriggers[1], rules.ConditionClearTriggerDamageTaken)
	}
}

// --- ProjectionStatModifiersToProto ---

func TestProjectionStatModifiersToProto_NilSlice(t *testing.T) {
	got := ProjectionStatModifiersToProto(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestProjectionStatModifiersToProto_ConvertsThroughViews(t *testing.T) {
	proj := []projectionstore.DaggerheartStatModifier{
		{
			ID:            "mod-pp",
			Target:        "armor_score",
			Delta:         1,
			ClearTriggers: []string{"session_end"},
		},
	}
	protos := ProjectionStatModifiersToProto(proj)
	if len(protos) != 1 {
		t.Fatalf("len = %d, want 1", len(protos))
	}
	if protos[0].GetId() != "mod-pp" {
		t.Fatalf("ID = %q, want %q", protos[0].GetId(), "mod-pp")
	}
	if protos[0].GetTarget() != "armor_score" {
		t.Fatalf("Target = %q, want %q", protos[0].GetTarget(), "armor_score")
	}
	if len(protos[0].GetClearTriggers()) != 1 ||
		protos[0].GetClearTriggers()[0] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END {
		t.Fatalf("ClearTriggers = %v, want [SESSION_END]", protos[0].GetClearTriggers())
	}
}

// --- normalizeRemovalIDs ---

func TestNormalizeRemovalIDs_EmptySlice(t *testing.T) {
	got, err := normalizeRemovalIDs(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestNormalizeRemovalIDs_DeduplicatesIDs(t *testing.T) {
	got, err := normalizeRemovalIDs([]string{"mod-1", "mod-2", "mod-1", "mod-3", "mod-2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	want := map[string]bool{"mod-1": true, "mod-2": true, "mod-3": true}
	for _, id := range got {
		if !want[id] {
			t.Fatalf("unexpected id %q in result", id)
		}
	}
}

func TestNormalizeRemovalIDs_TrimsWhitespace(t *testing.T) {
	got, err := normalizeRemovalIDs([]string{" mod-1 ", "  mod-2  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != "mod-1" || got[1] != "mod-2" {
		t.Fatalf("got %v, want [mod-1 mod-2]", got)
	}
}

func TestNormalizeRemovalIDs_WhitespaceOnlyReturnsError(t *testing.T) {
	_, err := normalizeRemovalIDs([]string{"mod-1", "   "})
	if err == nil {
		t.Fatal("expected error for whitespace-only id")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", st.Code(), codes.InvalidArgument)
	}
}

func TestNormalizeRemovalIDs_EmptyStringReturnsError(t *testing.T) {
	_, err := normalizeRemovalIDs([]string{""})
	if err == nil {
		t.Fatal("expected error for empty string id")
	}
}

func TestNormalizeRemovalIDs_PreservesOrderAfterDedup(t *testing.T) {
	got, err := normalizeRemovalIDs([]string{"c", "a", "b", "a", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0] != "c" || got[1] != "a" || got[2] != "b" {
		t.Fatalf("got %v, want [c a b] (first-seen order)", got)
	}
}

// --- clearTriggerFromProto ---

func TestClearTriggerFromProto_AllKnownTriggers(t *testing.T) {
	tests := []struct {
		proto  pb.DaggerheartConditionClearTrigger
		domain rules.ConditionClearTrigger
	}{
		{pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST, rules.ConditionClearTriggerShortRest},
		{pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST, rules.ConditionClearTriggerLongRest},
		{pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END, rules.ConditionClearTriggerSessionEnd},
		{pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN, rules.ConditionClearTriggerDamageTaken},
	}
	for _, tt := range tests {
		got := clearTriggerFromProto(tt.proto)
		if got != tt.domain {
			t.Fatalf("clearTriggerFromProto(%v) = %q, want %q", tt.proto, got, tt.domain)
		}
	}
}

func TestClearTriggerFromProto_UnspecifiedReturnsEmpty(t *testing.T) {
	got := clearTriggerFromProto(pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED)
	if got != "" {
		t.Fatalf("clearTriggerFromProto(UNSPECIFIED) = %q, want empty", got)
	}
}

func TestClearTriggerFromProto_UnknownValueReturnsEmpty(t *testing.T) {
	got := clearTriggerFromProto(pb.DaggerheartConditionClearTrigger(999))
	if got != "" {
		t.Fatalf("clearTriggerFromProto(999) = %q, want empty", got)
	}
}

// --- domainClearTriggerToProto ---

func TestDomainClearTriggerToProto_AllKnownTriggers(t *testing.T) {
	tests := []struct {
		domain rules.ConditionClearTrigger
		proto  pb.DaggerheartConditionClearTrigger
	}{
		{rules.ConditionClearTriggerShortRest, pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST},
		{rules.ConditionClearTriggerLongRest, pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST},
		{rules.ConditionClearTriggerSessionEnd, pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END},
		{rules.ConditionClearTriggerDamageTaken, pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN},
	}
	for _, tt := range tests {
		got := domainClearTriggerToProto(tt.domain)
		if got != tt.proto {
			t.Fatalf("domainClearTriggerToProto(%q) = %v, want %v", tt.domain, got, tt.proto)
		}
	}
}

func TestDomainClearTriggerToProto_UnknownReturnsUnspecified(t *testing.T) {
	got := domainClearTriggerToProto(rules.ConditionClearTrigger("unknown"))
	if got != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED {
		t.Fatalf("domainClearTriggerToProto(unknown) = %v, want UNSPECIFIED", got)
	}
}

func TestDomainClearTriggerToProto_EmptyReturnsUnspecified(t *testing.T) {
	got := domainClearTriggerToProto("")
	if got != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED {
		t.Fatalf("domainClearTriggerToProto(empty) = %v, want UNSPECIFIED", got)
	}
}

// --- domainClearTriggersToProto ---

func TestDomainClearTriggersToProto_NilSlice(t *testing.T) {
	got := domainClearTriggersToProto(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestDomainClearTriggersToProto_MultipleValues(t *testing.T) {
	got := domainClearTriggersToProto([]rules.ConditionClearTrigger{
		rules.ConditionClearTriggerShortRest,
		rules.ConditionClearTriggerDamageTaken,
	})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST {
		t.Fatalf("got[0] = %v, want SHORT_REST", got[0])
	}
	if got[1] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN {
		t.Fatalf("got[1] = %v, want DAMAGE_TAKEN", got[1])
	}
}

// --- projectionClearTriggersToProto ---

func TestProjectionClearTriggersToProto_NilSlice(t *testing.T) {
	got := projectionClearTriggersToProto(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestProjectionClearTriggersToProto_TrimsWhitespace(t *testing.T) {
	got := projectionClearTriggersToProto([]string{" short_rest ", "  long_rest  "})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST {
		t.Fatalf("got[0] = %v, want SHORT_REST", got[0])
	}
	if got[1] != pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST {
		t.Fatalf("got[1] = %v, want LONG_REST", got[1])
	}
}

// --- requireDependencies ---

func TestRequireDependencies_NilCampaignStore(t *testing.T) {
	h := NewHandler(Dependencies{})
	err := h.requireDependencies()
	if err == nil {
		t.Fatal("expected error for nil campaign store")
	}
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.Internal)
	}
}
