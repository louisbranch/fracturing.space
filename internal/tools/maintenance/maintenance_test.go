package maintenance

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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

func TestEnvOrDefault(t *testing.T) {
	t.Run("nil lookup returns fallback", func(t *testing.T) {
		got := envOrDefault(nil, []string{"KEY"}, "fb")
		if got != "fb" {
			t.Errorf("expected %q, got %q", "fb", got)
		}
	})

	t.Run("key found", func(t *testing.T) {
		lookup := func(key string) (string, bool) {
			if key == "A" {
				return "found", true
			}
			return "", false
		}
		got := envOrDefault(lookup, []string{"A"}, "fb")
		if got != "found" {
			t.Errorf("expected %q, got %q", "found", got)
		}
	})

	t.Run("whitespace value falls through", func(t *testing.T) {
		lookup := func(key string) (string, bool) {
			if key == "A" {
				return "  ", true
			}
			return "", false
		}
		got := envOrDefault(lookup, []string{"A"}, "fb")
		if got != "fb" {
			t.Errorf("expected %q, got %q", "fb", got)
		}
	})

	t.Run("first matching key wins", func(t *testing.T) {
		lookup := func(key string) (string, bool) {
			switch key {
			case "A":
				return "", false
			case "B":
				return "b-val", true
			default:
				return "", false
			}
		}
		got := envOrDefault(lookup, []string{"A", "B"}, "fb")
		if got != "b-val" {
			t.Errorf("expected %q, got %q", "b-val", got)
		}
	})

	t.Run("no keys returns fallback", func(t *testing.T) {
		lookup := func(string) (string, bool) { return "", false }
		got := envOrDefault(lookup, nil, "fb")
		if got != "fb" {
			t.Errorf("expected %q, got %q", "fb", got)
		}
	})
}

func TestDefaultEventsDBPath(t *testing.T) {
	t.Run("no env", func(t *testing.T) {
		lookup := func(string) (string, bool) { return "", false }
		got := defaultEventsDBPath(lookup)
		if got != "data/game-events.db" {
			t.Errorf("expected default path, got %q", got)
		}
	})

	t.Run("env set", func(t *testing.T) {
		lookup := func(key string) (string, bool) {
			if key == "FRACTURING_SPACE_GAME_EVENTS_DB_PATH" {
				return "/custom/events.db", true
			}
			return "", false
		}
		got := defaultEventsDBPath(lookup)
		if got != "/custom/events.db" {
			t.Errorf("expected %q, got %q", "/custom/events.db", got)
		}
	})
}

func TestDefaultProjectionsDBPath(t *testing.T) {
	t.Run("no env", func(t *testing.T) {
		lookup := func(string) (string, bool) { return "", false }
		got := defaultProjectionsDBPath(lookup)
		if got != "data/game-projections.db" {
			t.Errorf("expected default path, got %q", got)
		}
	})

	t.Run("env set", func(t *testing.T) {
		lookup := func(key string) (string, bool) {
			if key == "FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH" {
				return "/custom/proj.db", true
			}
			return "", false
		}
		got := defaultProjectionsDBPath(lookup)
		if got != "/custom/proj.db" {
			t.Errorf("expected %q, got %q", "/custom/proj.db", got)
		}
	})
}

func TestOutputJSON(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{
			CampaignID: "c1",
			Mode:       "scan",
		}
		outputJSON(&out, &errOut, result)
		if errOut.Len() != 0 {
			t.Errorf("unexpected error output: %s", errOut.String())
		}
		var decoded runResult
		if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
			t.Fatalf("invalid JSON output: %v", err)
		}
		if decoded.CampaignID != "c1" {
			t.Errorf("campaign_id = %q, want %q", decoded.CampaignID, "c1")
		}
	})

	t.Run("with warnings", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{
			CampaignID:    "c2",
			Mode:          "validate",
			Warnings:      []string{"warn1"},
			WarningsTotal: 5,
		}
		outputJSON(&out, &errOut, result)
		if !strings.Contains(out.String(), `"warnings_total":5`) {
			t.Errorf("expected warnings_total in output: %s", out.String())
		}
	})
}

func TestPrintResult(t *testing.T) {
	t.Run("error output", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{Error: "something failed"}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(errOut.String(), "Error: something failed") {
			t.Errorf("expected error in errOut: %s", errOut.String())
		}
	})

	t.Run("warnings output", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{Warnings: []string{"w1", "w2"}, WarningsTotal: 5}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(errOut.String(), "Warning: w1") {
			t.Errorf("expected warning w1: %s", errOut.String())
		}
		if !strings.Contains(errOut.String(), "3 more warnings suppressed") {
			t.Errorf("expected suppressed warning count: %s", errOut.String())
		}
	})

	t.Run("integrity report", func(t *testing.T) {
		var out, errOut bytes.Buffer
		report := integrityReport{
			LastSeq:             100,
			CharacterMismatches: 2,
			MissingStates:       1,
			GmFearMatch:         true,
			GmFearSource:        5,
			GmFearReplay:        5,
		}
		reportJSON, _ := json.Marshal(report)
		result := runResult{
			CampaignID: "c1",
			Mode:       "integrity",
			Report:     reportJSON,
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(out.String(), "Integrity check") {
			t.Errorf("expected integrity output: %s", out.String())
		}
		if !strings.Contains(out.String(), "GM fear match: true") {
			t.Errorf("expected GM fear match: %s", out.String())
		}
		if !strings.Contains(out.String(), "Character state mismatches: 2") {
			t.Errorf("expected character mismatches: %s", out.String())
		}
	})

	t.Run("scan report", func(t *testing.T) {
		var out, errOut bytes.Buffer
		report := snapshotScanReport{
			LastSeq:        50,
			TotalEvents:    100,
			SnapshotEvents: 10,
		}
		reportJSON, _ := json.Marshal(report)
		result := runResult{
			CampaignID: "c1",
			Mode:       "scan",
			Report:     reportJSON,
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(out.String(), "Scanned snapshot-related events") {
			t.Errorf("expected scan output: %s", out.String())
		}
	})

	t.Run("validate report", func(t *testing.T) {
		var out, errOut bytes.Buffer
		report := snapshotScanReport{
			LastSeq:        50,
			TotalEvents:    100,
			SnapshotEvents: 10,
			InvalidEvents:  3,
		}
		reportJSON, _ := json.Marshal(report)
		result := runResult{
			CampaignID: "c1",
			Mode:       "validate",
			Report:     reportJSON,
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(out.String(), "Validated snapshot-related events") {
			t.Errorf("expected validate output: %s", out.String())
		}
	})

	t.Run("replay report", func(t *testing.T) {
		var out, errOut bytes.Buffer
		report := snapshotScanReport{LastSeq: 50}
		reportJSON, _ := json.Marshal(report)
		result := runResult{
			CampaignID: "c1",
			Mode:       "replay",
			Report:     reportJSON,
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(out.String(), "Replayed snapshot-related events") {
			t.Errorf("expected replay output: %s", out.String())
		}
	})

	t.Run("prefix applied", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{Error: "oops"}
		printResult(&out, &errOut, result, "[c1] ")
		if !strings.Contains(errOut.String(), "[c1] Error: oops") {
			t.Errorf("expected prefix in output: %s", errOut.String())
		}
	})

	t.Run("empty report returns early", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{CampaignID: "c1", Mode: "scan"}
		printResult(&out, &errOut, result, "")
		if out.Len() != 0 {
			t.Errorf("expected no output for empty report: %s", out.String())
		}
	})

	t.Run("invalid integrity JSON", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{
			CampaignID: "c1",
			Mode:       "integrity",
			Report:     json.RawMessage(`{invalid`),
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(errOut.String(), "decode report") {
			t.Errorf("expected decode error: %s", errOut.String())
		}
	})

	t.Run("invalid scan JSON", func(t *testing.T) {
		var out, errOut bytes.Buffer
		result := runResult{
			CampaignID: "c1",
			Mode:       "scan",
			Report:     json.RawMessage(`{invalid`),
		}
		printResult(&out, &errOut, result, "")
		if !strings.Contains(errOut.String(), "decode report") {
			t.Errorf("expected decode error: %s", errOut.String())
		}
	})
}

func TestRunValidationErrors(t *testing.T) {
	t.Run("integrity with dry-run", func(t *testing.T) {
		cfg := Config{
			CampaignID: "c1",
			Integrity:  true,
			DryRun:     true,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-integrity cannot be combined") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("integrity with validate", func(t *testing.T) {
		cfg := Config{
			CampaignID: "c1",
			Integrity:  true,
			Validate:   true,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-integrity cannot be combined") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("integrity with after-seq", func(t *testing.T) {
		cfg := Config{
			CampaignID: "c1",
			Integrity:  true,
			AfterSeq:   10,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "-integrity does not support -after-seq") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("no campaign IDs", func(t *testing.T) {
		cfg := Config{}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil {
			t.Fatal("expected error for no campaign IDs")
		}
	})

	t.Run("negative warnings cap", func(t *testing.T) {
		cfg := Config{
			CampaignID:  "c1",
			WarningsCap: -1,
		}
		err := Run(t.Context(), cfg, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "warnings-cap") {
			t.Fatalf("expected warnings-cap error, got %v", err)
		}
	})
}

// --- fake event store ---

// fakeEventStore implements storage.EventStore with canned events.
type fakeEventStore struct {
	events  map[string][]event.Event // keyed by campaignID
	listErr error
}

func (f *fakeEventStore) AppendEvent(_ context.Context, _ event.Event) (event.Event, error) {
	return event.Event{}, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) GetEventByHash(_ context.Context, _ string) (event.Event, error) {
	return event.Event{}, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) GetEventBySeq(_ context.Context, _ string, _ uint64) (event.Event, error) {
	return event.Event{}, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	all := f.events[campaignID]
	var result []event.Event
	for _, evt := range all {
		if evt.Seq > afterSeq {
			result = append(result, evt)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (f *fakeEventStore) ListEventsBySession(_ context.Context, _, _ string, _ uint64, _ int) ([]event.Event, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) GetLatestEventSeq(_ context.Context, _ string) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (f *fakeEventStore) ListEventsPage(_ context.Context, _ storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	return storage.ListEventsPageResult{}, fmt.Errorf("not implemented")
}

// --- scanSnapshotEvents tests ---

func TestScanSnapshotEventsEmpty(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{}}
	report, warnings, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalEvents != 0 {
		t.Fatalf("expected 0 events, got %d", report.TotalEvents)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
}

func TestScanSnapshotEventsCountsSnapshotEvents(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.TypeCampaignCreated},
			{Seq: 2, Type: daggerheart.EventTypeCharacterStatePatched, SystemID: "daggerheart"},
			{Seq: 3, Type: event.TypeCharacterCreated},
			{Seq: 4, Type: daggerheart.EventTypeGMFearChanged, SystemID: "daggerheart"},
		},
	}}
	report, _, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalEvents != 4 {
		t.Fatalf("expected 4 total events, got %d", report.TotalEvents)
	}
	if report.SnapshotEvents != 2 {
		t.Fatalf("expected 2 snapshot events, got %d", report.SnapshotEvents)
	}
	if report.LastSeq != 4 {
		t.Fatalf("expected last seq 4, got %d", report.LastSeq)
	}
}

func TestScanSnapshotEventsWithValidation(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: daggerheart.EventTypeCharacterStatePatched, SystemID: "dh", PayloadJSON: []byte(`{"character_id":"ch1"}`)},
			{Seq: 2, Type: daggerheart.EventTypeCharacterStatePatched, SystemID: "dh", PayloadJSON: []byte(`{invalid`)},
		},
	}}
	report, warnings, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 0, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.InvalidEvents != 1 {
		t.Fatalf("expected 1 invalid event, got %d", report.InvalidEvents)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestScanSnapshotEventsUntilSeq(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.TypeCampaignCreated},
			{Seq: 2, Type: event.TypeCharacterCreated},
			{Seq: 3, Type: event.TypeCharacterCreated},
		},
	}}
	report, _, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalEvents != 2 {
		t.Fatalf("expected 2 events, got %d", report.TotalEvents)
	}
}

func TestScanSnapshotEventsAfterSeq(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.TypeCampaignCreated},
			{Seq: 2, Type: event.TypeCharacterCreated},
			{Seq: 3, Type: event.TypeCharacterCreated},
		},
	}}
	report, _, err := scanSnapshotEvents(t.Context(), store, "c1", 2, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.TotalEvents != 1 {
		t.Fatalf("expected 1 event after seq 2, got %d", report.TotalEvents)
	}
}

func TestScanSnapshotEventsNilStore(t *testing.T) {
	_, _, err := scanSnapshotEvents(t.Context(), nil, "c1", 0, 0, false)
	if err == nil || !strings.Contains(err.Error(), "event store is not configured") {
		t.Fatalf("expected nil store error, got %v", err)
	}
}

func TestScanSnapshotEventsEmptyCampaign(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{}}
	_, _, err := scanSnapshotEvents(t.Context(), store, "", 0, 0, false)
	if err == nil || !strings.Contains(err.Error(), "campaign id is required") {
		t.Fatalf("expected campaign id error, got %v", err)
	}
}

func TestScanSnapshotEventsListError(t *testing.T) {
	store := &fakeEventStore{listErr: fmt.Errorf("db gone")}
	_, _, err := scanSnapshotEvents(t.Context(), store, "c1", 0, 0, false)
	if err == nil || !strings.Contains(err.Error(), "db gone") {
		t.Fatalf("expected list error, got %v", err)
	}
}

// --- validateSnapshotEvent tests ---

func TestValidateSnapshotEventCharacterStatePatched(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{"valid", `{"character_id":"ch1"}`, false},
		{"invalid json", `{bad`, true},
		{"missing character_id", `{}`, true},
		{"hp out of range", `{"character_id":"ch1","hp_after":9999}`, true},
		{"hope_max out of range", `{"character_id":"ch1","hope_max_after":9999}`, true},
		{"hope out of range", `{"character_id":"ch1","hope_after":9999}`, true},
		{"stress out of range", `{"character_id":"ch1","stress_after":9999}`, true},
		{"armor out of range", `{"character_id":"ch1","armor_after":9999}`, true},
		{"invalid life_state", `{"character_id":"ch1","life_state_after":"bogus"}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := event.Event{Type: daggerheart.EventTypeCharacterStatePatched, PayloadJSON: []byte(tt.payload)}
			err := validateSnapshotEvent(evt)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSnapshotEventDeathMoveResolved(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{"valid", `{"character_id":"ch1","move":"risk_it_all","life_state_after":"dead"}`, false},
		{"invalid json", `{bad`, true},
		{"missing character_id", `{"move":"risk_it_all","life_state_after":"dead"}`, true},
		{"invalid move", `{"character_id":"ch1","move":"bogus","life_state_after":"dead"}`, true},
		{"missing life_state_after", `{"character_id":"ch1","move":"risk_it_all"}`, true},
		{"invalid life_state_after", `{"character_id":"ch1","move":"risk_it_all","life_state_after":"bogus"}`, true},
		{"hope_die out of range", `{"character_id":"ch1","move":"risk_it_all","life_state_after":"dead","hope_die":99}`, true},
		{"fear_die out of range", `{"character_id":"ch1","move":"risk_it_all","life_state_after":"dead","fear_die":99}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := event.Event{Type: daggerheart.EventTypeDeathMoveResolved, PayloadJSON: []byte(tt.payload)}
			err := validateSnapshotEvent(evt)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSnapshotEventBlazeOfGlory(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{"valid", `{"character_id":"ch1","life_state_after":"dead"}`, false},
		{"invalid json", `{bad`, true},
		{"missing character_id", `{"life_state_after":"dead"}`, true},
		{"missing life_state_after", `{"character_id":"ch1"}`, true},
		{"invalid life_state_after", `{"character_id":"ch1","life_state_after":"bogus"}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := event.Event{Type: daggerheart.EventTypeBlazeOfGloryResolved, PayloadJSON: []byte(tt.payload)}
			err := validateSnapshotEvent(evt)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSnapshotEventAttackResolved(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{"valid", `{"character_id":"ch1","roll_seq":1,"targets":["t1"],"outcome":"hit"}`, false},
		{"invalid json", `{bad`, true},
		{"missing character_id", `{"roll_seq":1,"targets":["t1"],"outcome":"hit"}`, true},
		{"missing roll_seq", `{"character_id":"ch1","targets":["t1"],"outcome":"hit"}`, true},
		{"missing targets", `{"character_id":"ch1","roll_seq":1,"outcome":"hit"}`, true},
		{"empty target", `{"character_id":"ch1","roll_seq":1,"targets":[""],"outcome":"hit"}`, true},
		{"missing outcome", `{"character_id":"ch1","roll_seq":1,"targets":["t1"]}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := event.Event{Type: daggerheart.EventTypeAttackResolved, PayloadJSON: []byte(tt.payload)}
			err := validateSnapshotEvent(evt)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSnapshotEventReactionResolved(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{"valid", `{"character_id":"ch1","roll_seq":1,"outcome":"success"}`, false},
		{"invalid json", `{bad`, true},
		{"missing character_id", `{"roll_seq":1,"outcome":"success"}`, true},
		{"missing roll_seq", `{"character_id":"ch1","outcome":"success"}`, true},
		{"missing outcome", `{"character_id":"ch1","roll_seq":1}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := event.Event{Type: daggerheart.EventTypeReactionResolved, PayloadJSON: []byte(tt.payload)}
			err := validateSnapshotEvent(evt)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSnapshotEventDamageRollResolved(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{"valid", `{"character_id":"ch1","roll_seq":1,"rolls":[{"value":3}]}`, false},
		{"invalid json", `{bad`, true},
		{"missing character_id", `{"roll_seq":1,"rolls":[{"value":3}]}`, true},
		{"missing roll_seq", `{"character_id":"ch1","rolls":[{"value":3}]}`, true},
		{"missing rolls", `{"character_id":"ch1","roll_seq":1}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := event.Event{Type: daggerheart.EventTypeDamageRollResolved, PayloadJSON: []byte(tt.payload)}
			err := validateSnapshotEvent(evt)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSnapshotEventGMFearChanged(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
	}{
		{"valid", `{"after":3}`, false},
		{"invalid json", `{bad`, true},
		{"out of range", `{"after":9999}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := event.Event{Type: daggerheart.EventTypeGMFearChanged, PayloadJSON: []byte(tt.payload)}
			err := validateSnapshotEvent(evt)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSnapshotEventUnknownType(t *testing.T) {
	evt := event.Event{Type: "unknown.type", PayloadJSON: []byte(`{}`)}
	if err := validateSnapshotEvent(evt); err != nil {
		t.Fatalf("unknown types should pass validation, got %v", err)
	}
}

// --- runCampaign tests ---

func TestRunCampaignDryRunScan(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: event.TypeCampaignCreated},
			{Seq: 2, Type: daggerheart.EventTypeGMFearChanged, SystemID: "dh"},
		},
	}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
	if result.Mode != "scan" {
		t.Fatalf("expected mode scan, got %s", result.Mode)
	}
	var report snapshotScanReport
	if err := json.Unmarshal(result.Report, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.TotalEvents != 2 {
		t.Fatalf("expected 2 total events, got %d", report.TotalEvents)
	}
	if report.SnapshotEvents != 1 {
		t.Fatalf("expected 1 snapshot event, got %d", report.SnapshotEvents)
	}
}

func TestRunCampaignValidateMode(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: daggerheart.EventTypeCharacterStatePatched, SystemID: "dh", PayloadJSON: []byte(`{invalid`)},
		},
	}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, Validate: true, WarningsCap: 25}, io.Discard)
	if result.Mode != "validate" {
		t.Fatalf("expected mode validate, got %s", result.Mode)
	}
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1 for invalid events, got %d", result.ExitCode)
	}
}

func TestRunCampaignReplayNilProjStore(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "projection store is not configured") {
		t.Fatalf("expected projection store error, got %s", result.Error)
	}
}

func TestRunCampaignScanError(t *testing.T) {
	store := &fakeEventStore{listErr: fmt.Errorf("db error")}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "scan snapshot-related events") {
		t.Fatalf("expected scan error, got %s", result.Error)
	}
}

func TestRunCampaignIntegrityNilStores(t *testing.T) {
	result := runCampaign(t.Context(), nil, nil, "c1", runOptions{Integrity: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "event store is not configured") {
		t.Fatalf("expected event store error, got %s", result.Error)
	}
}

func TestRunCampaignIntegrityNilProjStore(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{Integrity: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "projection store is not configured") {
		t.Fatalf("expected projection store error, got %s", result.Error)
	}
}

// checkSnapshotIntegrity empty-campaign requires a non-nil ProjectionStore,
// which is a large composite interface. We test the nil-store branches above
// and leave deeper integrity testing to integration tests.

func TestRunCampaignValidateNoInvalid(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {
			{Seq: 1, Type: daggerheart.EventTypeGMFearChanged, SystemID: "dh", PayloadJSON: []byte(`{"after":3}`)},
		},
	}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, Validate: true, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
}

func TestRunCampaignDryRunWithJSONOutput(t *testing.T) {
	store := &fakeEventStore{events: map[string][]event.Event{
		"c1": {{Seq: 1, Type: event.TypeCampaignCreated}},
	}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, WarningsCap: 25, JSONOutput: true}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
	// Verify JSON output works
	var out bytes.Buffer
	outputJSON(&out, io.Discard, result)
	if !strings.Contains(out.String(), `"campaign_id":"c1"`) {
		t.Fatalf("expected campaign_id in JSON output: %s", out.String())
	}
}
