package maintenance

import (
	"flag"
	"reflect"
	"testing"
)

func TestResolveCampaignIDs(t *testing.T) {
	tests := []struct {
		single   string
		list     string
		expected []string
		wantErr  bool
	}{
		{single: "", list: "", wantErr: true},
		{single: "c1", list: "c2", wantErr: true},
		{single: "c1", list: "", expected: []string{"c1"}},
		{single: "", list: "c1, c2", expected: []string{"c1", "c2"}},
		{single: "", list: " , c1 , , c2 ", expected: []string{"c1", "c2"}},
	}

	for _, tc := range tests {
		got, err := resolveCampaignIDs(tc.single, tc.list)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("expected error for %q/%q", tc.single, tc.list)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for %q/%q: %v", tc.single, tc.list, err)
		}
		if !reflect.DeepEqual(got, tc.expected) {
			t.Fatalf("expected %v, got %v", tc.expected, got)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	if got := splitCSV(" a, b ,, "); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("expected trimmed entries, got %v", got)
	}
}

func TestCapWarnings(t *testing.T) {
	warnings := []string{"a", "b", "c"}
	if got, total := capWarnings(warnings, 0); total != 3 || len(got) != 3 {
		t.Fatalf("expected all warnings, got %v (total=%d)", got, total)
	}
	if got, total := capWarnings(warnings, 2); total != 3 || len(got) != 2 {
		t.Fatalf("expected capped warnings, got %v (total=%d)", got, total)
	}
}

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil, func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.EventsDBPath != "data/game-events.db" {
		t.Fatalf("expected default events db path, got %q", cfg.EventsDBPath)
	}
	if cfg.ProjectionsDBPath != "data/game-projections.db" {
		t.Fatalf("expected default projections db path, got %q", cfg.ProjectionsDBPath)
	}
	if cfg.WarningsCap != 25 {
		t.Fatalf("expected warnings cap 25, got %d", cfg.WarningsCap)
	}
}

func TestParseConfigOverrides(t *testing.T) {
	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	lookup := func(key string) (string, bool) {
		switch key {
		case "FRACTURING_SPACE_GAME_EVENTS_DB_PATH":
			return "env-events", true
		case "FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH":
			return "env-projections", true
		default:
			return "", false
		}
	}
	args := []string{
		"-events-db-path", "flag-events",
		"-projections-db-path", "flag-projections",
		"-warnings-cap", "5",
	}
	cfg, err := ParseConfig(fs, args, lookup)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.EventsDBPath != "flag-events" {
		t.Fatalf("expected flag override for events db, got %q", cfg.EventsDBPath)
	}
	if cfg.ProjectionsDBPath != "flag-projections" {
		t.Fatalf("expected flag override for projections db, got %q", cfg.ProjectionsDBPath)
	}
	if cfg.WarningsCap != 5 {
		t.Fatalf("expected warnings cap 5, got %d", cfg.WarningsCap)
	}
}
