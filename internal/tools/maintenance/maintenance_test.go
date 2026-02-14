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

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
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
	cfg, err := ParseConfig(fs, nil)
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
	t.Setenv("FRACTURING_SPACE_GAME_EVENTS_DB_PATH", "env-events")
	t.Setenv("FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH", "env-projections")

	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	args := []string{
		"-events-db-path", "flag-events",
		"-projections-db-path", "flag-projections",
		"-warnings-cap", "5",
	}
	cfg, err := ParseConfig(fs, args)
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

func TestValidateCharacterStatePatchedHopeConstrainedByHopeMax(t *testing.T) {
	// When hope_max_after is set, hope_after must be <= hope_max_after.
	hopeMax := 3
	hopeOK := 3
	hopeBad := 4
	tests := []struct {
		name    string
		payload daggerheart.CharacterStatePatchedPayload
		wantErr bool
	}{
		{
			name:    "hope equals hope_max",
			payload: daggerheart.CharacterStatePatchedPayload{CharacterID: "ch1", HopeMaxAfter: &hopeMax, HopeAfter: &hopeOK},
			wantErr: false,
		},
		{
			name:    "hope exceeds custom hope_max",
			payload: daggerheart.CharacterStatePatchedPayload{CharacterID: "ch1", HopeMaxAfter: &hopeMax, HopeAfter: &hopeBad},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.payload)
			evt := event.Event{Type: daggerheart.EventTypeCharacterStatePatched, PayloadJSON: data}
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

func TestValidateCharacterStatePatchedHpNegative(t *testing.T) {
	hp := -1
	payload := daggerheart.CharacterStatePatchedPayload{CharacterID: "ch1", HpAfter: &hp}
	data, _ := json.Marshal(payload)
	evt := event.Event{Type: daggerheart.EventTypeCharacterStatePatched, PayloadJSON: data}
	if err := validateSnapshotEvent(evt); err == nil {
		t.Fatal("expected error for negative hp_after")
	}
}

func TestValidateDeathMoveResolvedDieBoundaries(t *testing.T) {
	// Valid boundaries: 1 and 12; invalid: 0 and 13.
	validLow := 1
	validHigh := 12
	invalidLow := 0
	invalidHigh := 13

	tests := []struct {
		name    string
		hopeDie *int
		fearDie *int
		wantErr bool
	}{
		{"hope_die=1 valid", &validLow, nil, false},
		{"hope_die=12 valid", &validHigh, nil, false},
		{"hope_die=0 invalid", &invalidLow, nil, true},
		{"hope_die=13 invalid", &invalidHigh, nil, true},
		{"fear_die=1 valid", nil, &validLow, false},
		{"fear_die=12 valid", nil, &validHigh, false},
		{"fear_die=0 invalid", nil, &invalidLow, true},
		{"fear_die=13 invalid", nil, &invalidHigh, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := daggerheart.DeathMoveResolvedPayload{
				CharacterID:    "ch1",
				Move:           "risk_it_all",
				LifeStateAfter: "dead",
				HopeDie:        tt.hopeDie,
				FearDie:        tt.fearDie,
			}
			data, _ := json.Marshal(payload)
			evt := event.Event{Type: daggerheart.EventTypeDeathMoveResolved, PayloadJSON: data}
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

func TestValidateGMFearChangedBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		after   int
		wantErr bool
	}{
		{"min valid", daggerheart.GMFearMin, false},
		{"max valid", daggerheart.GMFearMax, false},
		{"below min", daggerheart.GMFearMin - 1, true},
		{"above max", daggerheart.GMFearMax + 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := daggerheart.GMFearChangedPayload{After: tt.after}
			data, _ := json.Marshal(payload)
			evt := event.Event{Type: daggerheart.EventTypeGMFearChanged, PayloadJSON: data}
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

func TestRunCampaignScanWarningsCapped(t *testing.T) {
	// Build events that each produce a validation warning when validated.
	var events []event.Event
	for i := 1; i <= 5; i++ {
		events = append(events, event.Event{
			Seq:         uint64(i),
			Type:        daggerheart.EventTypeCharacterStatePatched,
			SystemID:    "dh",
			PayloadJSON: []byte(`{invalid`),
		})
	}
	store := &fakeEventStore{events: map[string][]event.Event{"c1": events}}
	result := runCampaign(t.Context(), store, nil, "c1", runOptions{DryRun: true, Validate: true, WarningsCap: 3}, io.Discard)
	if len(result.Warnings) != 3 {
		t.Fatalf("expected 3 capped warnings, got %d", len(result.Warnings))
	}
	if result.WarningsTotal != 5 {
		t.Fatalf("expected 5 total warnings, got %d", result.WarningsTotal)
	}
}

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

// --- runWithDeps tests ---

func TestRunWithDeps_MultiCampaign(t *testing.T) {
	evtStore := &fakeClosableEventStore{
		fakeEventStore: fakeEventStore{events: map[string][]event.Event{
			"c1": {{Seq: 1, Type: event.TypeCampaignCreated}},
			"c2": {{Seq: 1, Type: event.TypeCampaignCreated}},
		}},
	}
	projStore := &fakeClosableProjectionStore{}

	cfg := Config{
		CampaignIDs: "c1, c2",
		DryRun:      true,
		WarningsCap: 25,
	}
	var out, errOut bytes.Buffer
	err := runWithDeps(t.Context(), cfg, evtStore, projStore, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "c1") {
		t.Errorf("expected c1 in output: %s", output)
	}
	if !strings.Contains(output, "c2") {
		t.Errorf("expected c2 in output: %s", output)
	}
}

func TestRunWithDeps_OneCampaignFails(t *testing.T) {
	evtStore := &fakeClosableEventStore{
		fakeEventStore: fakeEventStore{events: map[string][]event.Event{
			"c1": {{Seq: 1, Type: daggerheart.EventTypeCharacterStatePatched, SystemID: "dh", PayloadJSON: []byte(`{invalid`)}},
			"c2": {{Seq: 1, Type: event.TypeCampaignCreated}},
		}},
	}
	projStore := &fakeClosableProjectionStore{}

	cfg := Config{
		CampaignIDs: "c1, c2",
		DryRun:      true,
		Validate:    true,
		WarningsCap: 25,
	}
	var out, errOut bytes.Buffer
	err := runWithDeps(t.Context(), cfg, evtStore, projStore, &out, &errOut)
	// One campaign has invalid events, so the overall run should fail.
	if err == nil {
		t.Fatal("expected error when one campaign fails")
	}
	// But c2 should still have been processed.
	if !strings.Contains(out.String(), "c2") {
		t.Errorf("expected c2 to be processed despite c1 failure: %s", out.String())
	}
}

func TestRunWithDeps_JSONOutputMode(t *testing.T) {
	evtStore := &fakeClosableEventStore{
		fakeEventStore: fakeEventStore{events: map[string][]event.Event{
			"c1": {{Seq: 1, Type: event.TypeCampaignCreated}},
		}},
	}
	projStore := &fakeClosableProjectionStore{}

	cfg := Config{
		CampaignID:  "c1",
		DryRun:      true,
		JSONOutput:  true,
		WarningsCap: 25,
	}
	var out, errOut bytes.Buffer
	err := runWithDeps(t.Context(), cfg, evtStore, projStore, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), `"campaign_id":"c1"`) {
		t.Errorf("expected JSON output with campaign_id: %s", out.String())
	}
}

func TestRunWithDeps_StoreCloseError(t *testing.T) {
	evtStore := &fakeClosableEventStore{
		fakeEventStore: fakeEventStore{events: map[string][]event.Event{
			"c1": {{Seq: 1, Type: event.TypeCampaignCreated}},
		}},
		closeErr: fmt.Errorf("event close failed"),
	}
	projStore := &fakeClosableProjectionStore{
		closeErr: fmt.Errorf("proj close failed"),
	}

	cfg := Config{
		CampaignID:  "c1",
		DryRun:      true,
		WarningsCap: 25,
	}
	var out, errOut bytes.Buffer
	err := runWithDeps(t.Context(), cfg, evtStore, projStore, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Close errors should be written to errOut.
	errOutput := errOut.String()
	if !strings.Contains(errOutput, "event close failed") {
		t.Errorf("expected event close error in errOut: %s", errOutput)
	}
	if !strings.Contains(errOutput, "proj close failed") {
		t.Errorf("expected proj close error in errOut: %s", errOutput)
	}
	// Both stores should be closed.
	if !evtStore.closed {
		t.Error("expected event store to be closed")
	}
	if !projStore.closed {
		t.Error("expected projection store to be closed")
	}
}

// --- runCampaign replay path tests ---

func TestRunCampaignReplayEmptyEvents(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	projStore := &fakeProjectionStore{}
	result := runCampaign(t.Context(), evtStore, projStore, "c1", runOptions{WarningsCap: 25}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
	if result.Mode != "replay" {
		t.Fatalf("expected mode replay, got %s", result.Mode)
	}
}

func TestRunCampaignReplayAfterSeqEmptyEvents(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	projStore := &fakeProjectionStore{}
	result := runCampaign(t.Context(), evtStore, projStore, "c1", runOptions{AfterSeq: 5, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d (error: %s)", result.ExitCode, result.Error)
	}
}

func TestRunCampaignReplayEventStoreError(t *testing.T) {
	evtStore := &fakeEventStore{listErr: fmt.Errorf("disk error")}
	projStore := &fakeProjectionStore{}
	result := runCampaign(t.Context(), evtStore, projStore, "c1", runOptions{WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "replay snapshot") {
		t.Fatalf("expected replay error, got: %s", result.Error)
	}
}

func TestRunCampaignReplayAfterSeqEventStoreError(t *testing.T) {
	evtStore := &fakeEventStore{listErr: fmt.Errorf("disk error")}
	projStore := &fakeProjectionStore{}
	result := runCampaign(t.Context(), evtStore, projStore, "c1", runOptions{AfterSeq: 5, WarningsCap: 25}, io.Discard)
	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Error, "replay snapshot") {
		t.Fatalf("expected replay error, got: %s", result.Error)
	}
}

// --- checkIntegrityWithStores tests ---

func TestCheckIntegrityWithStores_GMFearMismatch(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (campaign.Campaign, error) {
			return campaign.Campaign{ID: "c1"}, nil
		},
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 5}, nil
		},
		listCharacters: func(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
			return storage.CharacterPage{}, nil
		},
	}
	scratch := &fakeProjectionStore{
		put: func(_ context.Context, _ campaign.Campaign) error { return nil },
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 3}, nil
		},
	}
	report, _, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.GmFearMatch {
		t.Fatal("expected GM fear mismatch")
	}
	if report.GmFearSource != 5 || report.GmFearReplay != 3 {
		t.Fatalf("expected source=5 replay=3, got source=%d replay=%d", report.GmFearSource, report.GmFearReplay)
	}
}

func TestCheckIntegrityWithStores_GMFearMatch(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (campaign.Campaign, error) {
			return campaign.Campaign{ID: "c1"}, nil
		},
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 5}, nil
		},
		listCharacters: func(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
			return storage.CharacterPage{}, nil
		},
	}
	scratch := &fakeProjectionStore{
		put: func(_ context.Context, _ campaign.Campaign) error { return nil },
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 5}, nil
		},
	}
	report, _, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.GmFearMatch {
		t.Fatal("expected GM fear match")
	}
}

func TestCheckIntegrityWithStores_CharacterMismatch(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (campaign.Campaign, error) {
			return campaign.Campaign{ID: "c1"}, nil
		},
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 0}, nil
		},
		listCharacters: func(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
			return storage.CharacterPage{
				Characters: []character.Character{{ID: "ch1"}},
			}, nil
		},
		getDaggerheartCharState: func(_ context.Context, _, charID string) (storage.DaggerheartCharacterState, error) {
			return storage.DaggerheartCharacterState{Hp: 10, Hope: 5, Stress: 2}, nil
		},
	}
	scratch := &fakeProjectionStore{
		put: func(_ context.Context, _ campaign.Campaign) error { return nil },
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{GMFear: 0}, nil
		},
		getDaggerheartCharState: func(_ context.Context, _, charID string) (storage.DaggerheartCharacterState, error) {
			return storage.DaggerheartCharacterState{Hp: 8, Hope: 3, Stress: 2}, nil
		},
	}
	report, warnings, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.CharacterMismatches != 1 {
		t.Fatalf("expected 1 character mismatch, got %d", report.CharacterMismatches)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestCheckIntegrityWithStores_MissingSourceState(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (campaign.Campaign, error) {
			return campaign.Campaign{ID: "c1"}, nil
		},
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{}, nil
		},
		listCharacters: func(_ context.Context, _ string, _ int, _ string) (storage.CharacterPage, error) {
			return storage.CharacterPage{
				Characters: []character.Character{{ID: "ch1"}},
			}, nil
		},
		getDaggerheartCharState: func(_ context.Context, _, _ string) (storage.DaggerheartCharacterState, error) {
			return storage.DaggerheartCharacterState{}, storage.ErrNotFound
		},
	}
	scratch := &fakeProjectionStore{
		put: func(_ context.Context, _ campaign.Campaign) error { return nil },
		getDaggerheartSnapshot: func(_ context.Context, _ string) (storage.DaggerheartSnapshot, error) {
			return storage.DaggerheartSnapshot{}, nil
		},
	}
	report, warnings, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.MissingStates != 1 {
		t.Fatalf("expected 1 missing state, got %d", report.MissingStates)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "missing source state") {
		t.Fatalf("expected missing source state warning, got %v", warnings)
	}
}

func TestCheckIntegrityWithStores_CampaignLoadError(t *testing.T) {
	evtStore := &fakeEventStore{events: map[string][]event.Event{}}
	source := &fakeProjectionStore{
		get: func(_ context.Context, _ string) (campaign.Campaign, error) {
			return campaign.Campaign{}, fmt.Errorf("load failed")
		},
	}
	scratch := &fakeProjectionStore{}
	_, _, err := checkIntegrityWithStores(t.Context(), evtStore, source, scratch, "c1", 0, io.Discard)
	if err == nil {
		t.Fatal("expected error for campaign load failure")
	}
	if !strings.Contains(err.Error(), "load campaign") {
		t.Fatalf("expected load campaign error, got: %v", err)
	}
}
